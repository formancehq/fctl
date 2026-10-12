// Package httpdebug provides bounded HTTP traces without consuming body streams.
package httpdebug

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
)

const previewLimit = 64 << 10

type transport struct {
	base http.RoundTripper
	out  io.Writer
	mu   sync.Mutex
	next atomic.Uint64
}

// New wraps base with redacted traces written to out (normally stderr). A nil
// base uses http.DefaultTransport; nil out disables tracing. Output failures
// never change an HTTP result. Response bodies are traced as the caller reads
// them, on EOF or Close, so tracing never waits for or drains a response stream.
func New(base http.RoundTripper, out io.Writer) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if out == nil {
		return base
	}
	return &transport{base: base, out: out}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	id := t.next.Add(1)
	t.write(fmt.Sprintf("HTTP %d request %q %q\n%sbody: %s\n", id, req.Method, safeURL(req.URL), headerTrace(req.Header), requestBody(req)))
	response, err := t.base.RoundTrip(req)
	if err != nil {
		// Transport errors can include raw URLs, credentials and server text.
		t.write(fmt.Sprintf("HTTP %d transport error (details omitted)\n", id))
	}
	if response != nil {
		t.write(fmt.Sprintf("HTTP %d response %d\n%s", id, response.StatusCode, headerTrace(response.Header)))
		if response.Body != nil {
			response.Body = &tracedBody{ReadCloser: response.Body, transport: t, id: id, contentType: response.Header.Get("Content-Type")}
		} else {
			t.write(fmt.Sprintf("HTTP %d response body: empty\n", id))
		}
	}
	return response, err
}

func (t *transport) write(trace string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, err := io.WriteString(t.out, trace); err != nil {
		return
	}
}

func requestBody(req *http.Request) string {
	if req.Body == nil || req.Body == http.NoBody {
		return "empty"
	}
	if req.GetBody == nil {
		return fmt.Sprintf("omitted (non-replayable body; size=%d)", req.ContentLength)
	}
	body, err := req.GetBody()
	if err != nil || body == nil {
		return "omitted (body copy unavailable)"
	}
	data, readErr := io.ReadAll(io.LimitReader(body, previewLimit+1))
	closeErr := body.Close()
	return bodyTrace(req.Header.Get("Content-Type"), data, int64(len(data)), readErr == nil && closeErr == nil && len(data) <= previewLimit)
}

type tracedBody struct {
	io.ReadCloser
	transport   *transport
	id          uint64
	contentType string
	preview     []byte
	size        int64
	once        sync.Once
	mu          sync.Mutex
}

func (b *tracedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.mu.Lock()
	b.size += int64(n)
	remaining := previewLimit + 1 - len(b.preview)
	if remaining > 0 {
		b.preview = append(b.preview, p[:min(n, remaining)]...)
	}
	b.mu.Unlock()
	if err != nil {
		b.finish(errors.Is(err, io.EOF))
	}
	return n, err
}

func (b *tracedBody) Close() error {
	err := b.ReadCloser.Close()
	b.finish(false)
	return err
}

func (b *tracedBody) finish(complete bool) {
	b.once.Do(func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		var trace bytes.Buffer
		if _, err := fmt.Fprintf(&trace, "HTTP %d response body: %s\n", b.id, bodyTrace(b.contentType, b.preview, b.size, complete && b.size <= previewLimit)); err != nil {
			return
		}
		b.transport.write(trace.String())
	})
}
