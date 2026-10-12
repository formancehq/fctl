package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/muesli/cancelreader"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/command"
)

type pluginStreamInput struct {
	io.Reader
	close func() error
	once  sync.Once
	err   error
}

func (r *pluginStreamInput) Close() error {
	r.once.Do(func() { r.err = r.close() })
	return r.err
}

func openPluginStreamInput(cmd *cobra.Command, spec pluginsdk.CommandSpec, request pluginsdk.ExecuteRequest) (*pluginStreamInput, *int64, error) {
	if !streamInput(spec) {
		return nil, nil, nil
	}
	source, err := pluginInputSource(spec.Files, request)
	if err != nil || source == "" {
		return nil, nil, err
	}
	if request.Body != nil {
		return nil, nil, fmt.Errorf("file input cannot be combined with an explicit body")
	}
	if err := cmd.Context().Err(); err != nil {
		return nil, nil, err
	}
	if source == "-" {
		return openPluginStreamStdin(cmd.Context(), cmd.InOrStdin())
	}
	if scheme := strings.ToLower(source); strings.HasPrefix(scheme, "https:") || strings.HasPrefix(scheme, "http:") {
		client := newPluginURLClient()
		input, size, err := openPluginStreamURL(cmd.Context(), source, client)
		if err != nil {
			client.CloseIdleConnections()
			return nil, nil, err
		}
		closeBody := input.close
		input.close = func() error {
			err := closeBody()
			client.CloseIdleConnections()
			return err
		}
		return input, size, nil
	}
	return openPluginStreamFile(source)
}

func openPluginStreamFile(source string) (*pluginStreamInput, *int64, error) {
	info, err := os.Stat(source)
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("input must be a regular file")
	}
	file, err := os.Open(source) //nolint:gosec // The source is an explicit user-selected regular file.
	if err != nil {
		return nil, nil, err
	}
	info, err = file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, nil, errors.Join(fmt.Errorf("input must be a regular file"), err, file.Close())
	}
	return &pluginStreamInput{Reader: file, close: file.Close}, new(info.Size()), nil
}

func openPluginStreamStdin(ctx context.Context, input io.Reader) (*pluginStreamInput, *int64, error) {
	if file, ok := input.(*os.File); ok {
		info, err := file.Stat()
		if err != nil {
			return nil, nil, err
		}
		// Regular stdin needs no readiness polling and remains caller-owned.
		if info.Mode().IsRegular() {
			return &pluginStreamInput{Reader: file, close: func() error { return nil }}, nil, nil
		}
	}
	reader, err := cancelreader.NewReader(input)
	if err != nil {
		return nil, nil, err
	}
	stdin := &pluginStreamStdinReader{reader: reader}
	var cancelOnce sync.Once
	var cancelErr error
	cancelInput := func() {
		cancelOnce.Do(func() {
			if !reader.Cancel() {
				if closer, ok := input.(io.Closer); ok {
					cancelErr = closer.Close()
				}
			}
		})
	}
	canceled := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		cancelInput()
		close(canceled)
	})
	return &pluginStreamInput{Reader: stdin, close: func() error {
		if !stop() {
			<-canceled
		}
		cancelInput()
		// Cancel first, then wait for a pending read before closing the
		// readiness descriptors. stdin itself remains caller-owned.
		stdin.mu.Lock()
		defer stdin.mu.Unlock()
		return errors.Join(cancelErr, reader.Close())
	}}, nil, nil
}

type pluginStreamStdinReader struct {
	reader cancelreader.CancelReader
	mu     sync.Mutex
}

func (r *pluginStreamStdinReader) Read(data []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reader.Read(data)
}

// Use the same public client and redirect validation as bounded file input.
// A failed source fetch is returned directly; the host never retries it.
func openPluginStreamURL(ctx context.Context, source string, client *http.Client) (*pluginStreamInput, *int64, error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid HTTPS input URL")
	}
	if err := validatePluginURL(u); err != nil {
		return nil, nil, err
	}
	if strings.Contains(source, "#") {
		return nil, nil, fmt.Errorf("HTTPS input URL must not contain a fragment")
	}
	u.Scheme = "https"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create HTTPS input request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		if failure, ok := errors.AsType[*url.Error](err); ok {
			err = failure.Err
		}
		return nil, nil, fmt.Errorf("fetch HTTPS input: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, nil, errors.Join(fmt.Errorf("HTTPS input returned HTTP status %d", resp.StatusCode), resp.Body.Close())
	}
	var size *int64
	if resp.ContentLength >= 0 {
		size = new(resp.ContentLength)
	}
	return &pluginStreamInput{Reader: resp.Body, close: resp.Body.Close}, size, nil
}

type pluginStreamOutput struct {
	writer      io.Writer
	file        *os.File
	destination string
	closeFile   func() error
	closeSink   func() error
	stop        func()
}

func openPluginStreamOutput(cmd *cobra.Command, spec pluginsdk.CommandSpec, request pluginsdk.ExecuteRequest, failures *pluginStreamErrors) (*pluginStreamOutput, error) {
	output := &pluginStreamOutput{}
	if !spec.Streams.Output {
		return output, nil
	}
	destination, err := pluginOutputDestination(spec.Files, request)
	if err != nil {
		return nil, err
	}
	output.destination = destination
	writer := command.RawOutput(cmd)
	if destination != "" && destination != "-" {
		output.file, err = os.CreateTemp(filepath.Dir(destination), ".fctl-export-*")
		if err != nil {
			return nil, fmt.Errorf("create plugin export: %w", err)
		}
		writer = output.file
	} else if file, ok := writer.(*os.File); ok {
		writer, output.closeSink, err = openPluginStreamFileSink(file)
		if err != nil {
			return nil, fmt.Errorf("open interruptible output: %w", err)
		}
	}
	output.interruptOnCancel(cmd.Context(), writer, failures)
	output.writer = &pluginStreamWriter{ctx: cmd.Context(), writer: writer, failures: failures}
	return output, nil
}

func (o *pluginStreamOutput) interruptOnCancel(ctx context.Context, writer io.Writer, failures *pluginStreamErrors) {
	closer, ok := writer.(io.Closer)
	if !ok {
		return
	}
	closeWriter := sync.OnceValue(closer.Close)
	if o.file != nil {
		o.closeFile = closeWriter
	}
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		//nolint:errcheck // Record retains this close error for the invocation result.
		_ = failures.Record(closeWriter())
		close(done)
	})
	o.stop = func() {
		if !stop() {
			<-done
		}
	}
}

func (o *pluginStreamOutput) Commit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if o.file == nil {
		if o.closeSink != nil {
			return o.closeSink()
		}
		return nil
	}
	if err := o.file.Sync(); err != nil {
		return err
	}
	err := o.closeFile()
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Rename(o.file.Name(), o.destination)
}

func (o *pluginStreamOutput) Close() (err error) {
	if o.stop != nil {
		o.stop()
	}
	if o.file == nil {
		if o.closeSink != nil {
			return o.closeSink()
		}
		return nil
	}
	err = o.closeFile()
	if removeErr := os.Remove(o.file.Name()); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		err = errors.Join(err, removeErr)
	}
	return err
}
