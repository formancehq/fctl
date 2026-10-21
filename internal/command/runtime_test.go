package command_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/command"
)

func TestReadBody(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	cmd.SetIn(strings.NewReader(`{"amount":9007199254740993}`))
	body, err := command.ReadBody(cmd, "-")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"amount":9007199254740993}` {
		t.Fatalf("body=%s", body)
	}
	for _, bad := range []string{"", `{} {}`, `{"invalid":`, strings.Repeat(" ", 4<<20+1)} {
		if _, err := command.ReadBody(cmd, bad); err == nil {
			t.Error("invalid/oversized input accepted")
		}
	}
}

func TestWriteJSONPreservesNumbers(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	if err := command.WriteJSON(&out, []byte(`{"amount":9007199254740993}`)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "9007199254740993") {
		t.Fatal(out.String())
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Fatal("missing newline")
	}
	want := errors.New("writer failed")
	if err := command.WriteJSON(failingWriter{want}, []byte(`{}`)); !errors.Is(err, want) {
		t.Fatalf("error=%v", err)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestStdinReadCancels(t *testing.T) {
	t.Parallel()
	reader, writer := io.Pipe()
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Error(err)
		}
	})
	t.Cleanup(func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := make(chan struct{})
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	cmd.SetIn(startReader{reader, started})
	finished := make(chan error, 1)
	go func() { _, err := command.ReadBody(cmd, "-"); finished <- err }()
	<-started
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("read error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stdin read did not cancel")
	}
}

type startReader struct {
	reader  io.Reader
	started chan struct{}
}

func (r startReader) Read(buf []byte) (int, error) { close(r.started); return r.reader.Read(buf) }
