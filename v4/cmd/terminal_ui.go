package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	v4render "github.com/formancehq/fctl/v4/internal/render"
)

func withTerminalSpinner[T any](cmd *cobra.Command, enabled bool, message string, doneMessage string, fn func() (T, error)) (T, error) {
	if !enabled || !terminalSpinnerEnabled(cmd) {
		return fn()
	}

	stop := startTerminalSpinner(cmd.ErrOrStderr(), message, doneMessage, commandColorEnabled(cmd))
	output, err := fn()
	stop(err == nil)
	return output, err
}

func startTerminalSpinner(writer io.Writer, message string, doneMessage string, color bool) func(bool) {
	model := spinner.MiniDot
	spinnerStyle := lipgloss.NewStyle()
	messageStyle := lipgloss.NewStyle()
	doneStyle := lipgloss.NewStyle()
	if color {
		spinnerStyle = spinnerStyle.Foreground(v4render.FormancePalette.Mint)
		messageStyle = messageStyle.Foreground(v4render.FormancePalette.Text)
		doneStyle = doneStyle.Foreground(v4render.FormancePalette.Success).Bold(true)
	}

	done := make(chan bool)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		frame := 0
		render := func() {
			fmt.Fprintf(writer, "\r\033[2K%s %s", spinnerStyle.Render(model.Frames[frame]), messageStyle.Render(message))
		}
		render()
		ticker := time.NewTicker(model.FPS)
		defer ticker.Stop()
		for {
			select {
			case success := <-done:
				fmt.Fprint(writer, "\r\033[2K")
				if success && strings.TrimSpace(doneMessage) != "" {
					fmt.Fprintf(writer, "%s %s\n", doneStyle.Render("OK"), doneMessage)
				}
				return
			case <-ticker.C:
				frame = (frame + 1) % len(model.Frames)
				render()
			}
		}
	}()
	return func(success bool) {
		done <- success
		<-stopped
	}
}

func terminalSpinnerEnabled(cmd *cobra.Command) bool {
	if nonInteractive, err := cmd.Root().PersistentFlags().GetBool(nonInteractiveFlag); err != nil || nonInteractive {
		return false
	}
	format, err := outputFormat(cmd)
	if err != nil || format != "plain" {
		return false
	}
	return terminalWriter(cmd.ErrOrStderr())
}

func terminalOutputEnabled(cmd *cobra.Command) bool {
	format, err := outputFormat(cmd)
	if err != nil || format != "plain" {
		return false
	}
	return terminalWriter(cmd.OutOrStdout())
}

func terminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	return ok && isatty.IsTerminal(file.Fd())
}

func commandColorEnabled(cmd *cobra.Command) bool {
	noColor, err := cmd.Root().PersistentFlags().GetBool(noColorFlag)
	return err == nil && !noColor
}

func styledKeyValueLine(cmd *cobra.Command, label string, value string) string {
	if !terminalOutputEnabled(cmd) {
		return fmt.Sprintf("%s\t%s", label, value)
	}

	labelStyle := lipgloss.NewStyle().Width(8).PaddingRight(1)
	valueStyle := lipgloss.NewStyle().Bold(true)
	prefixStyle := lipgloss.NewStyle()
	if commandColorEnabled(cmd) {
		labelStyle = labelStyle.Foreground(v4render.FormancePalette.Muted)
		valueStyle = valueStyle.Foreground(v4render.FormancePalette.Text)
		prefixStyle = prefixStyle.Foreground(v4render.FormancePalette.Success).Bold(true)
	}
	return fmt.Sprintf("%s %s %s", prefixStyle.Render("OK"), labelStyle.Render(label), valueStyle.Render(value))
}
