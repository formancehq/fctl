package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/command"
)

// Both adapter paths use this boundary after validation and confirmation.
func executePluginCommand(cmd *cobra.Command, instance pluginsdk.Plugin, spec pluginsdk.CommandSpec, request pluginsdk.ExecuteRequest) error {
	if spec.Streams == nil {
		response, err := instance.Execute(cmd.Context(), request)
		return renderPluginOutput(cmd, spec.Files, request, response, err)
	}
	streaming, ok := instance.(pluginsdk.StreamingPlugin)
	if !ok {
		return fmt.Errorf("plugin does not support declared streams")
	}
	if err := checkPluginStreamFormat(spec, request); err != nil {
		return err
	}
	return executePluginStream(cmd, streaming, spec, request)
}

func executePluginStream(cmd *cobra.Command, instance pluginsdk.StreamingPlugin, spec pluginsdk.CommandSpec, request pluginsdk.ExecuteRequest) (err error) {
	ctx := cmd.Context()
	if err := ctx.Err(); err != nil {
		return err
	}
	var failures pluginStreamErrors
	progress := newPluginProgress(cmd, &failures)
	streams := pluginsdk.ExecutionStreams{Report: progress.Report}
	input, size, err := openPluginStreamInput(cmd, spec, request)
	if err != nil {
		return err
	}
	if input != nil {
		defer func() { err = errors.Join(err, input.Close()) }()
		streams.Input = &pluginStreamReader{ctx: ctx, reader: input, failures: &failures}
		streams.InputSize = size
	}
	output, err := openPluginStreamOutput(cmd, spec, request, &failures)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, output.Close()) }()
	streams.Output = output.writer
	response, execErr := instance.ExecuteStream(ctx, request, streams)
	// Close the source before committing an export. Input errors and close
	// errors must prevent publication even if a plugin ignores them.
	if input != nil {
		execErr = errors.Join(execErr, input.Close())
	}
	execErr = errors.Join(execErr, ctx.Err(), failures.Err())
	execErr = completePluginStream(ctx, output, progress, execErr)
	if spec.Streams.Output {
		if output.destination == "" || output.destination == "-" {
			return execErr // stdout belongs exclusively to the raw export, including on failure.
		}
		if len(response.Data) > 0 {
			return errors.Join(execErr, command.WriteJSON(cmd.OutOrStdout(), response.Data))
		}
		return execErr
	}
	return renderPluginOutput(cmd, spec.Files, request, response, execErr)
}

func completePluginStream(ctx context.Context, output *pluginStreamOutput, progress *pluginProgress, execErr error) error {
	// A failing final progress write must not publish an export.
	execErr = errors.Join(execErr, progress.Finish(execErr))
	if ctx.Err() != nil {
		// The interrupted pipe/RPC errors are consequences of cancellation.
		return ctx.Err()
	}
	if execErr == nil {
		execErr = output.Commit(ctx)
	}
	return execErr
}

// Record the first IO/report failure even when a plugin does not propagate it.
type pluginStreamErrors struct {
	mu  sync.Mutex
	err error
}

func (e *pluginStreamErrors) Record(err error) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.err == nil {
		e.err = err
	}
	return err
}

func (e *pluginStreamErrors) Err() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.err
}

type pluginStreamReader struct {
	ctx      context.Context
	reader   io.Reader
	failures *pluginStreamErrors
}

func (r *pluginStreamReader) Close() error {
	if closer, ok := r.reader.(io.Closer); ok {
		return r.failures.Record(closer.Close())
	}
	return nil
}

func (r *pluginStreamReader) Read(data []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, r.failures.Record(err)
	}
	n, err := r.reader.Read(data)
	if r.ctx.Err() != nil {
		err = r.ctx.Err()
	}
	if err != nil && !errors.Is(err, io.EOF) {
		err = r.failures.Record(err)
	}
	return n, err
}

type pluginStreamWriter struct {
	ctx      context.Context
	writer   io.Writer
	failures *pluginStreamErrors
	mu       sync.Mutex
}

func (w *pluginStreamWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ctx.Err(); err != nil {
		return 0, w.failures.Record(err)
	}
	n, err := w.writer.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	err = errors.Join(err, w.ctx.Err())
	return n, w.failures.Record(err)
}
