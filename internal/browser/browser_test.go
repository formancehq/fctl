package browser

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"testing/synctest"
	"time"
)

func TestOSCommands(t *testing.T) {
	target := "https://example.test/verify?user_code=ABCD&next=%2Fhome"
	tests := []struct {
		goos, name string
		args       []string
	}{
		{"darwin", "open", []string{target}},
		{"linux", "xdg-open", []string{target}},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", target}},
	}
	for _, test := range tests {
		t.Run(test.goos, func(t *testing.T) {
			calls := 0
			err := open(t.Context(), target, test.goos, func(ctx context.Context, name string, args ...string) error {
				calls++
				if name != test.name || !reflect.DeepEqual(args, test.args) {
					t.Fatalf("command=%s %q", name, args)
				}
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > openTimeout || time.Until(deadline) <= 0 {
					t.Fatal("missing bounded deadline")
				}
				return nil
			})
			if err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestURLs(t *testing.T) {
	for _, target := range []string{
		"https://example.test/verify?user_code=ABCD&next=%2Fhome",
		"https://example.test/verify?code=$(touch%20file);echo%20x",
		"http://localhost:8080/verify?user_code=ABCD",
		"http://127.0.0.1:8080/verify?user_code=ABCD",
		"http://[::1]:8080/verify?user_code=ABCD",
	} {
		t.Run(target, func(t *testing.T) {
			calls := 0
			err := open(t.Context(), target, "linux", func(_ context.Context, _ string, args ...string) error {
				calls++
				if len(args) != 1 || args[0] != target {
					t.Fatal("URL must be one unchanged argument")
				}
				return nil
			})
			if err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestInvalidURLsNeverExecute(t *testing.T) {
	for _, target := range []string{"", "/verify", "https:///verify", "https://", "https://user:secret@example.test/verify", "http://example.test/verify", "file:///etc/passwd", "javascript:alert(1)", "https://example.test/%zz", "--help", "https://example.test/verify#fragment"} {
		t.Run(target, func(t *testing.T) {
			err := open(t.Context(), target, "linux", unexpectedRunner(t))
			if err == nil {
				t.Fatal("expected URL validation error")
			}
		})
	}
}

func unexpectedRunner(t *testing.T) commandRunner {
	t.Helper()
	return func(context.Context, string, ...string) error { t.Fatal("unexpected OS command"); return nil }
}

func TestCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := open(ctx, "https://example.test/verify", "linux", unexpectedRunner(t))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestCancelDuringRun(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	err := open(ctx, "https://example.test/verify", "linux", func(ctx context.Context, _ string, _ ...string) error {
		cancel()
		<-ctx.Done()
		return errors.New("process killed")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		err := open(t.Context(), "https://example.test/verify", "linux", func(ctx context.Context, _ string, _ ...string) error {
			<-ctx.Done()
			return errors.New("process killed")
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestCommandFailure(t *testing.T) {
	failure := errors.New("opener unavailable")
	err := open(t.Context(), "https://example.test/verify", "linux", func(context.Context, string, ...string) error { return failure })
	if !errors.Is(err, failure) {
		t.Fatalf("err=%v", err)
	}
}

func TestUnsupportedOS(t *testing.T) {
	err := open(t.Context(), "https://example.test/verify", "plan9", unexpectedRunner(t))
	if err == nil {
		t.Fatal("expected unsupported OS error")
	}
}

func TestOpenValidation(t *testing.T) {
	// Exercise the exported entry point without starting any OS process.
	if err := Open(t.Context(), "file:///etc/passwd"); err == nil {
		t.Fatal("expected invalid URL error")
	}
}

func TestCommandContextCanceledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	// A canceled CommandContext must not start even an available executable.
	err := runCommand(ctx, os.Args[0])
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}
