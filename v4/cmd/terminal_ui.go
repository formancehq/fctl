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
	return withTerminalSpinnerUpdates(cmd, enabled, message, doneMessage, func(func(string)) (T, error) {
		return fn()
	})
}

func withTerminalSpinnerUpdates[T any](cmd *cobra.Command, enabled bool, message string, doneMessage string, fn func(update func(string)) (T, error)) (T, error) {
	if !enabled || !terminalSpinnerEnabled(cmd) {
		return fn(func(string) {})
	}

	spinner := startTerminalSpinner(cmd.ErrOrStderr(), message, doneMessage, commandColorEnabled(cmd))
	output, err := fn(spinner.Update)
	spinner.Stop(err == nil)
	return output, err
}

type terminalSpinner struct {
	update  chan string
	done    chan bool
	stopped chan struct{}
}

func startTerminalSpinner(writer io.Writer, message string, doneMessage string, color bool) *terminalSpinner {
	model := spinner.MiniDot
	spinnerStyle := lipgloss.NewStyle()
	messageStyle := lipgloss.NewStyle()
	doneStyle := lipgloss.NewStyle()
	if color {
		spinnerStyle = spinnerStyle.Foreground(v4render.FormancePalette.Mint)
		messageStyle = messageStyle.Foreground(v4render.FormancePalette.Text)
		doneStyle = doneStyle.Foreground(v4render.FormancePalette.Success).Bold(true)
	}

	update := make(chan string, 1)
	done := make(chan bool)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		frame := 0
		currentMessage := message
		startedAt := time.Now()
		render := func() {
			fmt.Fprintf(writer, "\r\033[2K%s %s", spinnerStyle.Render(model.Frames[frame]), messageStyle.Render(spinnerMessageWithElapsed(currentMessage, time.Since(startedAt))))
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
			case nextMessage := <-update:
				if strings.TrimSpace(nextMessage) != "" {
					currentMessage = nextMessage
				}
				render()
			case <-ticker.C:
				frame = (frame + 1) % len(model.Frames)
				render()
			}
		}
	}()
	return &terminalSpinner{update: update, done: done, stopped: stopped}
}

func (s *terminalSpinner) Update(message string) {
	select {
	case s.update <- message:
	default:
		select {
		case <-s.update:
		default:
		}
		s.update <- message
	}
}

func (s *terminalSpinner) Stop(success bool) {
	s.done <- success
	<-s.stopped
}

func spinnerMessageWithElapsed(message string, elapsed time.Duration) string {
	return fmt.Sprintf("%s (%ds)", message, int(elapsed.Seconds()))
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
	return styledKeyValueLineWithWidth(cmd, label, value, 8)
}

type styledKeyValue struct {
	Label string
	Value string
}

func writeStyledKeyValues(cmd *cobra.Command, rows ...styledKeyValue) error {
	if !terminalOutputEnabled(cmd) {
		for _, row := range rows {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", row.Label, row.Value); err != nil {
				return err
			}
		}
		return nil
	}

	width := 8
	for _, row := range rows {
		if len(row.Label) > width {
			width = len(row.Label)
		}
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), styledKeyValueLineWithWidth(cmd, row.Label, row.Value, width)); err != nil {
			return err
		}
	}
	return nil
}

func writeStyledKeyValueRows(cmd *cobra.Command, rows [][]string) error {
	values := make([]styledKeyValue, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		values = append(values, styledKeyValue{Label: row[0], Value: row[1]})
	}
	return writeStyledKeyValues(cmd, values...)
}

func writeStyledNamedKeyValueRows(cmd *cobra.Command, label string, rows [][]string) error {
	if !terminalOutputEnabled(cmd) {
		for _, row := range rows {
			if len(row) < 2 {
				continue
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", label, row[0], row[1]); err != nil {
				return err
			}
		}
		return nil
	}
	values := make([]styledKeyValue, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		values = append(values, styledKeyValue{Label: label + " " + row[0], Value: row[1]})
	}
	return writeStyledKeyValues(cmd, values...)
}

func writeStyledColonKeyValues(cmd *cobra.Command, rows ...styledKeyValue) error {
	if !terminalOutputEnabled(cmd) {
		for _, row := range rows {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", row.Label, row.Value); err != nil {
				return err
			}
		}
		return nil
	}
	return writeStyledKeyValues(cmd, rows...)
}

func writeStyledSectionTitle(cmd *cobra.Command, title string) error {
	if !terminalOutputEnabled(cmd) {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", title)
		return err
	}
	style := lipgloss.NewStyle().Bold(true)
	if commandColorEnabled(cmd) {
		style = style.Foreground(v4render.FormancePalette.Muted)
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), style.Render(title))
	return err
}

func writeStyledBulletedPairRows(cmd *cobra.Command, headers []string, rows [][]string) error {
	if !terminalOutputEnabled(cmd) {
		for _, row := range rows {
			if len(row) < 2 {
				continue
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "- %s (%s)\n", row[0], row[1]); err != nil {
				return err
			}
		}
		return nil
	}
	return writeStyledRows(cmd, headers, rows)
}

func writeStyledRows(cmd *cobra.Command, headers []string, rows [][]string) error {
	if !terminalOutputEnabled(cmd) {
		for _, row := range rows {
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), strings.Join(row, "\t")); err != nil {
				return err
			}
		}
		return nil
	}
	return v4render.Table(cmd.OutOrStdout(), headers, rows)
}

func writeStyledTable(cmd *cobra.Command, headers []string, rows [][]string) error {
	if !terminalOutputEnabled(cmd) {
		return v4render.Table(cmd.OutOrStdout(), headers, rows)
	}
	return writeStyledRows(cmd, headers, rows)
}

func styledKeyValueLineWithWidth(cmd *cobra.Command, label string, value string, width int) string {
	if !terminalOutputEnabled(cmd) {
		return fmt.Sprintf("%s\t%s", label, value)
	}

	labelStyle := lipgloss.NewStyle().Width(width + 1)
	valueStyle := lipgloss.NewStyle().Bold(true)
	prefixStyle := lipgloss.NewStyle()
	if commandColorEnabled(cmd) {
		labelStyle = labelStyle.Foreground(v4render.FormancePalette.Muted)
		valueStyle = valueStyle.Foreground(v4render.FormancePalette.Text)
		prefixStyle = prefixStyle.Foreground(v4render.FormancePalette.Success).Bold(true)
	}
	return fmt.Sprintf("%s %s %s", prefixStyle.Render("OK"), labelStyle.Render(label), valueStyle.Render(value))
}

func styledEmptyLine(cmd *cobra.Command, message string) string {
	if !terminalOutputEnabled(cmd) {
		return message
	}

	style := lipgloss.NewStyle()
	if commandColorEnabled(cmd) {
		style = style.Foreground(v4render.FormancePalette.Muted)
	}
	return style.Render(message)
}

func styledInfoLine(cmd *cobra.Command, label string, value string) string {
	if !terminalOutputEnabled(cmd) {
		return fmt.Sprintf("%s: %s", label, value)
	}

	labelStyle := lipgloss.NewStyle()
	valueStyle := lipgloss.NewStyle().Bold(true)
	if commandColorEnabled(cmd) {
		labelStyle = labelStyle.Foreground(v4render.FormancePalette.Muted)
		valueStyle = valueStyle.Foreground(v4render.FormancePalette.Text)
	}
	return fmt.Sprintf("%s %s", labelStyle.Render(label), valueStyle.Render(value))
}

func writeStyledAPIVersion(cmd *cobra.Command, version any) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledInfoLine(cmd, "API version", fmt.Sprint(version)))
	return err
}

func writeStyledNext(cmd *cobra.Command, next string) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledInfoLine(cmd, "Next", next))
	return err
}

func styledSuccessLine(cmd *cobra.Command, message string) string {
	if !terminalOutputEnabled(cmd) {
		return message
	}

	prefixStyle := lipgloss.NewStyle()
	messageStyle := lipgloss.NewStyle().Bold(true)
	if commandColorEnabled(cmd) {
		prefixStyle = prefixStyle.Foreground(v4render.FormancePalette.Success).Bold(true)
		messageStyle = messageStyle.Foreground(v4render.FormancePalette.Text)
	}
	return fmt.Sprintf("%s %s", prefixStyle.Render("OK"), messageStyle.Render(message))
}
