package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

const pluginProgressInterval = 200 * time.Millisecond

type pluginProgress struct {
	ctx      context.Context
	writer   io.Writer
	enabled  bool
	failures *pluginStreamErrors
	mu       sync.Mutex
	last     time.Time
	now      func() time.Time
	latest   *pluginsdk.ProgressEvent
	width    int
	columns  int
	frame    int
	finished bool
}

func newPluginProgress(cmd *cobra.Command, failures *pluginStreamErrors) *pluginProgress {
	columns := 80
	if file, ok := cmd.ErrOrStderr().(*os.File); ok {
		if width, _, err := term.GetSize(file.Fd()); err == nil && width > 1 {
			columns = width
		}
	}
	return &pluginProgress{ctx: cmd.Context(), writer: cmd.ErrOrStderr(), enabled: pluginProgressEnabled(cmd), failures: failures, now: time.Now, columns: columns}
}

func pluginProgressEnabled(cmd *cobra.Command) bool {
	file, ok := cmd.ErrOrStderr().(*os.File)
	return pluginProgressAllowed(cmd, ok && term.IsTerminal(file.Fd()))
}

func pluginProgressAllowed(cmd *cobra.Command, terminal bool) bool {
	if flag := lookupFlag(cmd, "no-input"); flag != nil && flag.Value.String() == "true" {
		return false
	}
	if pluginProgressTruthy(os.Getenv("CI")) || pluginProgressTruthy(os.Getenv("FCTL_NO_INPUT")) {
		return false
	}
	if flag := lookupFlag(cmd, "output"); flag != nil && flag.Changed && flag.Value.String() == "json" {
		return false
	}
	return terminal
}

func pluginProgressTruthy(value string) bool {
	return value != "" && value != "0" && !strings.EqualFold(value, "false")
}

func (p *pluginProgress) Report(event pluginsdk.ProgressEvent) error {
	// Validation is unconditional, including suppressed and throttled events.
	if err := event.Validate(); err != nil {
		return p.failures.Record(err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ctx.Err(); err != nil {
		return p.failures.Record(err)
	}
	if p.finished {
		return p.failures.Record(fmt.Errorf("plugin progress already finished"))
	}
	if event.Total != nil {
		event.Total = new(*event.Total)
	}
	p.latest = &event
	if !p.enabled {
		return nil
	}
	now := p.now()
	if !p.last.IsZero() && now.Sub(p.last) < pluginProgressInterval {
		return nil
	}
	p.last = now
	line := pluginProgressLine(event, p.frame, false)
	p.frame++
	return p.writeLine(line)
}

// Finish bypasses throttling once, then terminates the stderr line before the
// command renders a result. Only a successful invocation can display 100%.
func (p *pluginProgress) Finish(execErr error) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.finished {
		return nil
	}
	p.finished = true
	if !p.enabled || p.latest == nil {
		return p.ctx.Err()
	}
	failed := execErr != nil || p.ctx.Err() != nil || p.failures.Err() != nil
	line := pluginProgressLine(*p.latest, max(0, p.frame-1), !failed)
	if failed {
		line += " (failed)"
	}
	lineErr := p.writeLine(line)
	n, newlineErr := io.WriteString(p.writer, "\n")
	if newlineErr == nil && n != 1 {
		newlineErr = io.ErrShortWrite
	}
	return errors.Join(lineErr, p.failures.Record(newlineErr), p.ctx.Err())
}

func (p *pluginProgress) writeLine(line string) error {
	if p.columns > 1 {
		line = ansi.Truncate(line, p.columns-1, "...")
	}
	width := ansi.StringWidth(line)
	output := "\r" + line + strings.Repeat(" ", max(0, p.width-width))
	n, err := io.WriteString(p.writer, output)
	if err == nil && n != len(output) {
		err = io.ErrShortWrite
	}
	p.width = width
	return p.failures.Record(err)
}

func pluginProgressLine(event pluginsdk.ProgressEvent, frame int, finished bool) string {
	if event.Total == nil {
		return fmt.Sprintf("%s %c %d %s", event.Phase, "|/-\\"[frame%4], event.Completed, event.Unit)
	}
	percent := 0
	if *event.Total > 0 {
		percent = int(float64(event.Completed) / float64(*event.Total) * 100)
	} else if finished {
		percent = 100
	}
	if !finished || event.Completed != *event.Total {
		percent = min(percent, 99)
	}
	const barWidth = 20
	filled := percent * barWidth / 100
	bar := strings.Repeat("=", filled) + strings.Repeat("-", barWidth-filled)
	return fmt.Sprintf("%s [%s] %3d%% %d/%d %s", event.Phase, bar, percent, event.Completed, *event.Total, event.Unit)
}
