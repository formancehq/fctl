//go:build darwin || linux

package plugin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/command"
)

func TestInheritedStdoutPipeCancellation(t *testing.T) {
	t.Parallel()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	fd := writer.Fd() // Match an inherited blocking stdout descriptor.
	duplicate, err := unix.FcntlInt(fd, unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	original := os.NewFile(uintptr(duplicate), "stdout") //nolint:gosec // F_DUPFD_CLOEXEC returns a nonnegative descriptor.
	t.Cleanup(func() {
		if err := errors.Join(reader.Close(), writer.Close(), original.Close()); err != nil {
			t.Error(err)
		}
	})
	flags, err := unix.FcntlInt(original.Fd(), unix.F_GETFL, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	cmd.SetOut(original)
	if err := command.ConfigureOutput(cmd, "json", "never"); err != nil {
		t.Fatal(err)
	}
	spec := pluginsdk.CommandSpec{Streams: &pluginsdk.StreamSpec{Output: true}, Files: &pluginsdk.FileSpec{WriteFlag: "file", WriteFormat: "ndjson"}}
	output, err := openPluginStreamOutput(cmd, spec, pluginsdk.ExecuteRequest{}, &pluginStreamErrors{})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := output.writer.Write(bytes.Repeat([]byte("x"), 1<<20))
		done <- err
	}()
	// Receiving a prefix proves Write started and exceeded the pipe capacity;
	// the remaining payload cannot finish while this reader stops consuming.
	if _, err := reader.Read(make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("inherited pipe cancellation lost: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("inherited pipe Write stayed blocked")
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	restored, err := unix.FcntlInt(original.Fd(), unix.F_GETFL, 0)
	// Darwin may add private pipe flags after IO; check the public modes.
	const modes = unix.O_ACCMODE | unix.O_NONBLOCK | unix.O_APPEND | unix.O_ASYNC
	if err != nil || restored&modes != flags&modes {
		t.Fatalf("original stdout closed or flags changed: %d -> %d (%v)", flags, restored, err)
	}
}

func TestFailedStdoutExportClosesOwnedDescriptor(t *testing.T) {
	t.Parallel()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
	})
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	cmd.SetOut(writer)
	spec := pluginsdk.CommandSpec{Streams: &pluginsdk.StreamSpec{Output: true}, Files: &pluginsdk.FileSpec{WriteFlag: "file", WriteFormat: "ndjson"}}
	output, err := openPluginStreamOutput(cmd, spec, pluginsdk.ExecuteRequest{}, &pluginStreamErrors{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(output.writer, "partial\n"); err != nil {
		t.Fatal(err)
	}
	// Failed invocations skip Commit; their deferred Close must release the
	// duplicate so stdout can subsequently reach EOF.
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Stat(); err != nil {
		t.Fatalf("host closed caller-owned stdout: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := reader.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err != nil || string(data) != "partial\n" {
		t.Fatalf("failed export leaked its descriptor: %q (%v)", data, err)
	}
}
