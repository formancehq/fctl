package transport

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRPCErrorStopsPluginBeforeLocalCancellation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		code codes.Code
		want error
	}{
		{name: "deadline", code: codes.DeadlineExceeded, want: context.DeadlineExceeded},
		{name: "canceled", code: codes.Canceled, want: context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assertRPCCancellationReapsPlugin(t, test.code, test.want)
		})
	}
}

func assertRPCCancellationReapsPlugin(t *testing.T, code codes.Code, want error) {
	t.Helper()
	client := openHelper(t, nil, "")
	ctx := t.Context()
	if ctx.Err() != nil || client.process.Exited() {
		t.Fatal("RPC must start with a live context and plugin")
	}
	// Match invoke's wrapping without relying on either deadline timer.
	rpcErr := fmt.Errorf("plugin RPC: %w", status.Error(code, "peer stopped the RPC"))
	err := client.rpcError(ctx, rpcErr)
	if !errors.Is(err, want) {
		t.Fatalf("RPC cancellation = %v, want %v", err, want)
	}
	if ctx.Err() != nil {
		t.Fatalf("RPC cancellation changed the caller context: %v", ctx.Err())
	}
	if !client.process.Exited() {
		t.Fatal("RPC cancellation returned before reaping the plugin")
	}
	if _, err := client.GetManifest(ctx); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed runtime restarted: %v", err)
	}
}

func TestRPCErrorPreservesOtherFailures(t *testing.T) {
	t.Parallel()
	for _, code := range []codes.Code{codes.Unavailable, codes.FailedPrecondition} {
		t.Run(code.String(), func(t *testing.T) {
			t.Parallel()
			client := openHelper(t, nil, "")
			rpcErr := fmt.Errorf("plugin RPC: %w", status.Error(code, "peer rejected the RPC"))
			if err := client.rpcError(t.Context(), rpcErr); !errors.Is(err, rpcErr) {
				t.Fatalf("RPC failure changed: %v", err)
			}
			if client.process.Exited() {
				t.Fatal("ordinary RPC failure terminated the plugin")
			}
			if _, err := client.GetManifest(t.Context()); err != nil {
				t.Fatalf("ordinary RPC failure closed the runtime: %v", err)
			}
		})
	}
}
