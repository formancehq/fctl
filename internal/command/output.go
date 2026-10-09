package command

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/presentation"
)

type outputWriter struct {
	io.Writer
	options presentation.Options
}

func (w *outputWriter) RenderJSON(value json.RawMessage) error {
	return presentation.Render(w.Writer, value, w.options)
}

// ConfigureOutput is called after flags are parsed, before any command work.
func ConfigureOutput(cmd *cobra.Command, format, color string) error {
	if err := presentation.ValidateFormat(format); err != nil {
		return err
	}
	if color != "auto" && color != "always" && color != "never" {
		return fmt.Errorf("color must be auto, always or never")
	}
	out := cmd.OutOrStdout()
	if wrapped, ok := out.(*outputWriter); ok {
		out = wrapped.Writer
	}
	terminal, width := terminalInfo(out)
	if format == "auto" {
		format = "json"
		if terminal {
			format = "table"
		}
	}
	useColor := color == "always" || (color == "auto" && terminal && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb")
	cmd.SetOut(&outputWriter{Writer: out, options: presentation.Options{Format: format, Color: useColor, Width: width}})
	return nil
}

// InstallHelp preserves Cobra's full help while emphasizing its section titles.
func InstallHelp(root *cobra.Command, color *string) {
	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		terminal, _ := terminalInfo(out)
		colored := *color == "always" || (*color == "auto" && terminal && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb")
		if !colored {
			defaultHelp(cmd, args)
			return
		}
		var help strings.Builder
		cmd.SetOut(&help)
		defaultHelp(cmd, args)
		cmd.SetOut(out)
		text := help.String()
		for _, label := range []string{"Usage:", "Examples:", "Available Commands:", "Flags:", "Global Flags:", "Cloud:", "Modules:", "Profiles:"} {
			text = strings.ReplaceAll(text, label, "\x1b[1;36m"+label+"\x1b[0m")
		}
		if _, err := io.WriteString(out, text); err != nil {
			return
		}
	})
}

func terminalInfo(out io.Writer) (bool, int) {
	file, ok := out.(*os.File)
	if !ok {
		return false, 80
	}
	return consoleInfo(file.Fd())
}
