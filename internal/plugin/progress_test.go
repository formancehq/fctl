package plugin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func TestPluginProgressTerminalOptOut(t *testing.T) {
	for _, tc := range []struct {
		name, ci, noInput, output string
		terminal, disabled, want  bool
	}{
		{name: "terminal", terminal: true, want: true},
		{name: "nonterminal"},
		{name: "CI", terminal: true, ci: "true"},
		{name: "numeric CI", terminal: true, ci: "1"},
		{name: "false CI", terminal: true, ci: "false", want: true},
		{name: "no-input", terminal: true, disabled: true},
		{name: "FCTL_NO_INPUT", terminal: true, noInput: "1"},
		{name: "explicit JSON", terminal: true, output: "json"},
		{name: "explicit table", terminal: true, output: "table", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CI", tc.ci)
			t.Setenv("FCTL_NO_INPUT", tc.noInput)
			root := &cobra.Command{}
			root.PersistentFlags().StringP("output", "o", "json", "output")
			root.PersistentFlags().Bool("no-input", tc.disabled, "no input")
			if tc.output != "" {
				if err := root.PersistentFlags().Set("output", tc.output); err != nil {
					t.Fatal(err)
				}
			}
			cmd := &cobra.Command{}
			root.AddCommand(cmd)
			if got := pluginProgressAllowed(cmd, tc.terminal); got != tc.want {
				t.Fatalf("progress enabled=%t, want %t", got, tc.want)
			}
		})
	}
}

func TestPluginProgressValidatesHiddenAndThrottledEvents(t *testing.T) {
	t.Parallel()
	for _, enabled := range []bool{false, true} {
		var failures pluginStreamErrors
		p := &pluginProgress{ctx: t.Context(), writer: io.Discard, enabled: enabled, failures: &failures, now: time.Now}
		if err := p.Report(pluginsdk.ProgressEvent{Phase: "import", Unit: "records"}); err != nil {
			t.Fatal(err)
		}
		for _, event := range []pluginsdk.ProgressEvent{
			{Phase: "bad\x1b[2J", Unit: "records"},
			{Phase: "import", Unit: "bad\n"},
			{Phase: "import", Unit: "records", Completed: -1},
			{Phase: "import", Unit: "records", Total: new(int64(-1))},
			{Phase: "import", Unit: "records", Completed: 2, Total: new(int64(1))},
		} {
			if err := p.Report(event); err == nil || failures.Err() == nil {
				t.Fatal("suppression bypassed progress validation")
			}
		}
	}
}

func TestPluginProgressThrottlesAndUsesOnlyStderr(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	var failures pluginStreamErrors
	now := time.Unix(100, 0)
	p := &pluginProgress{ctx: t.Context(), writer: &stderr, enabled: true, failures: &failures, now: func() time.Time { return now }}
	for i := range 10000 {
		// Phase changes cannot bypass the throttle.
		event := pluginsdk.ProgressEvent{Phase: strings.Repeat("a", 1+i%3), Completed: int64(i), Total: new(int64(10000)), Unit: "records"}
		if err := p.Report(event); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.HasPrefix(stderr.String(), "\ra [--------------------]") || !strings.HasSuffix(stderr.String(), "0% 0/10000 records") || strings.Count(stderr.String(), "\r") != 1 || strings.ContainsAny(stderr.String(), "\n\x1b") {
		t.Fatalf("progress was not bounded or contained terminal controls: %q", stderr.String())
	}
	now = now.Add(pluginProgressInterval)
	if err := p.Report(pluginsdk.ProgressEvent{Phase: "import", Completed: 10000, Unit: "records"}); err != nil {
		t.Fatal(err)
	}
	if strings.Count(stderr.String(), "\r") != 2 || !strings.Contains(stderr.String(), "\rimport / 10000 records") || strings.ContainsAny(stderr.String(), "\n\x1b") {
		t.Fatal("throttled progress did not resume")
	}
}

func TestPluginProgressFinishFlushesLatestAndNewline(t *testing.T) {
	t.Parallel()
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[failed], func(t *testing.T) {
			t.Parallel()
			testPluginProgressFinish(t, failed)
		})
	}
}

func testPluginProgressFinish(t *testing.T, failed bool) {
	t.Helper()
	var stderr bytes.Buffer
	var failures pluginStreamErrors
	p := &pluginProgress{ctx: t.Context(), writer: &stderr, enabled: true, failures: &failures, now: func() time.Time { return time.Unix(100, 0) }}
	event := pluginsdk.ProgressEvent{Phase: "import", Completed: 1, Total: new(int64(100)), Unit: "records"}
	if err := p.Report(event); err != nil {
		t.Fatal(err)
	}
	event.Completed = 100
	if err := p.Report(event); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stderr.String(), "100%") || strings.Count(stderr.String(), "\r") != 1 {
		t.Fatal("throttled reports claimed completion before execution ended")
	}
	*event.Total = 1000 // Finishing must use the accepted snapshot, not a retained plugin pointer.
	var execErr error
	if failed {
		execErr = errors.New("plugin failed")
	}
	if err := p.Finish(execErr); err != nil {
		t.Fatal(err)
	}
	want := "100% 100/100 records\n"
	if failed {
		want = "99% 100/100 records (failed)\n"
	}
	if !strings.HasSuffix(stderr.String(), want) || strings.Count(stderr.String(), "\n") != 1 || strings.Count(stderr.String(), "\r") != 2 {
		t.Fatalf("Finish lost pending progress or newline: %q", stderr.String())
	}
	before := stderr.String()
	if err := p.Finish(execErr); err != nil || stderr.String() != before {
		t.Fatal("Finish was not idempotent")
	}
}

func TestPluginProgressPadsShorterLinesAndFlushesUnknownCounter(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	var failures pluginStreamErrors
	now := time.Unix(100, 0)
	p := &pluginProgress{ctx: t.Context(), writer: &stderr, enabled: true, failures: &failures, now: func() time.Time { return now }}
	if err := p.Report(pluginsdk.ProgressEvent{Phase: "long import phase", Completed: 10, Unit: "records"}); err != nil {
		t.Fatal(err)
	}
	firstWidth := p.width
	now = now.Add(pluginProgressInterval)
	if err := p.Report(pluginsdk.ProgressEvent{Phase: "apply", Completed: 3, Unit: "rows"}); err != nil {
		t.Fatal(err)
	}
	want := "\rapply / 3 rows" + strings.Repeat(" ", firstWidth-len("apply / 3 rows"))
	if !strings.HasSuffix(stderr.String(), want) {
		t.Fatalf("short progress left old text: %q", stderr.String())
	}
	if err := p.Report(pluginsdk.ProgressEvent{Phase: "apply", Completed: 4, Unit: "rows"}); err != nil {
		t.Fatal(err)
	}
	if err := p.Finish(nil); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(stderr.String(), "\rapply / 4 rows\n") || strings.Contains(stderr.String(), "%") {
		t.Fatalf("indeterminate finish invented a total or lost the last counter: %q", stderr.String())
	}
}

func TestPluginProgressFinishCanceledDoesNotClaimSuccess(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var stderr bytes.Buffer
	var failures pluginStreamErrors
	p := &pluginProgress{ctx: ctx, writer: &stderr, enabled: true, failures: &failures, now: time.Now}
	if err := p.Report(pluginsdk.ProgressEvent{Phase: "apply", Completed: 1, Total: new(int64(1)), Unit: "records"}); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := p.Finish(nil); !errors.Is(err, context.Canceled) || strings.Contains(stderr.String(), "100%") || !strings.HasSuffix(stderr.String(), "(failed)\n") {
		t.Fatalf("canceled finish claimed success or lost cancellation: %q (%v)", stderr.String(), err)
	}
}

func TestPluginProgressBoundsTerminalLineAndHugeTotals(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	var failures pluginStreamErrors
	p := &pluginProgress{ctx: t.Context(), writer: &stderr, enabled: true, failures: &failures, now: time.Now, columns: 40}
	const total int64 = 1<<63 - 1
	event := pluginsdk.ProgressEvent{Phase: strings.Repeat("x", 128), Completed: total - 1, Total: new(total), Unit: "bytes"}
	if err := p.Report(event); err != nil {
		t.Fatal(err)
	}
	if strings.Count(stderr.String(), "\r") != 1 || len(stderr.String()) > 40 {
		t.Fatal("long label wrapped the terminal line")
	}
	if line := pluginProgressLine(event, 0, true); strings.Contains(line, "100%") {
		t.Fatalf("rounded float ratio invented completion: %q", line)
	}
}

type progressFailingWriter struct{}

func (progressFailingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestPluginProgressCancellationAndWriterErrors(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var failures pluginStreamErrors
	p := &pluginProgress{ctx: ctx, writer: progressFailingWriter{}, enabled: true, failures: &failures, now: time.Now}
	event := pluginsdk.ProgressEvent{Phase: "import", Unit: "records"}
	if err := p.Report(event); !errors.Is(err, io.ErrClosedPipe) || !errors.Is(failures.Err(), io.ErrClosedPipe) {
		t.Fatalf("progress write error lost: %v", err)
	}
	cancel()
	p.enabled = false
	if err := p.Report(event); !errors.Is(err, context.Canceled) {
		t.Fatalf("hidden progress lost cancellation: %v", err)
	}
}
