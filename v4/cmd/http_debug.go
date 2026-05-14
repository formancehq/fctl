package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type debugRoundTripper struct {
	base   http.RoundTripper
	writer io.Writer
}

func (t debugRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	writer := t.writer
	if writer == nil {
		writer = io.Discard
	}

	if err := dumpDebugRequest(writer, req); err != nil {
		return nil, err
	}
	rsp, err := base.RoundTrip(req)
	if err != nil {
		fmt.Fprintf(writer, "<<< error: %v\n\n", err)
		return nil, err
	}
	if err := dumpDebugResponse(writer, rsp); err != nil {
		return nil, err
	}
	return rsp, nil
}

func dumpDebugRequest(writer io.Writer, req *http.Request) error {
	fmt.Fprintf(writer, ">>> %s %s\n", req.Method, req.URL.String())
	writeDebugHeaders(writer, req.Header)
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(body))
		writeDebugBody(writer, body)
	}
	fmt.Fprintln(writer)
	return nil
}

func dumpDebugResponse(writer io.Writer, rsp *http.Response) error {
	fmt.Fprintf(writer, "<<< HTTP %s\n", rsp.Status)
	writeDebugHeaders(writer, rsp.Header)
	if rsp.Body != nil {
		body, err := io.ReadAll(rsp.Body)
		if err != nil {
			return err
		}
		_ = rsp.Body.Close()
		rsp.Body = io.NopCloser(bytes.NewReader(body))
		writeDebugBody(writer, body)
	}
	fmt.Fprintln(writer)
	return nil
}

func writeDebugHeaders(writer io.Writer, headers http.Header) {
	for key, values := range headers {
		for _, value := range values {
			if debugHeaderShouldRedact(key) {
				value = "<redacted>"
			}
			fmt.Fprintf(writer, "%s: %s\n", key, value)
		}
	}
}

func debugHeaderShouldRedact(key string) bool {
	switch http.CanonicalHeaderKey(key) {
	case "Authorization", "Cookie", "Set-Cookie":
		return true
	default:
		return false
	}
}

func writeDebugBody(writer io.Writer, body []byte) {
	if len(body) == 0 {
		return
	}
	var raw any
	if err := json.Unmarshal(body, &raw); err == nil {
		pretty, err := json.MarshalIndent(raw, "", "  ")
		if err == nil {
			fmt.Fprintf(writer, "%s\n", pretty)
			return
		}
	}
	fmt.Fprintf(writer, "%s\n", body)
}
