package command_test

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/command"
)

const outputEnvelope = `{"data":[{"name":"books","amount":90071992547409930001}],"next":"cursor/+=","hasMore":true}`

func configuredOutput(t *testing.T, format, color string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "fixture"}
	cmd.Flags().String("output", format, "Output format")
	cmd.Flags().String("color", color, "Color policy")
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := command.ConfigureOutput(cmd, format, color); err != nil {
		t.Fatal(err)
	}
	return cmd, &out
}

func TestConfigureOutputMachineJSONNeverHasANSI(t *testing.T) {
	t.Parallel()
	var want bytes.Buffer
	if err := json.Indent(&want, []byte(outputEnvelope), "", "  "); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"auto", "json"} {
		for _, color := range []string{"auto", "always", "never"} {
			t.Run(format+"/"+color, func(t *testing.T) {
				cmd, out := configuredOutput(t, format, color)
				if err := command.WriteJSON(cmd.OutOrStdout(), json.RawMessage(outputEnvelope)); err != nil {
					t.Fatal(err)
				}
				if out.String() != want.String()+"\n" || strings.Contains(out.String(), "\x1b") {
					t.Fatalf("machine JSON changed: %q", out.String())
				}
			})
		}
	}
}

func TestConfigureOutputTableUsesBufferWidthAndExplicitColor(t *testing.T) {
	t.Parallel()
	for _, color := range []string{"auto", "never", "always"} {
		t.Run(color, func(t *testing.T) {
			cmd, out := configuredOutput(t, "table", color)
			data := json.RawMessage(`{"data":[{"name":"` + strings.Repeat("long", 35) + `","amount":90071992547409930001}]}`)
			if err := command.WriteJSON(cmd.OutOrStdout(), data); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "\x1b[") != (color == "always") {
				t.Fatal("buffer table color ignored explicit policy")
			}
			plain := stripOutputANSI(out.String())
			if !strings.Contains(plain, "90071992547409930001") {
				t.Fatal("table changed a large integer")
			}
			assertOutputWidth(t, plain, 80)
		})
	}
}

func assertOutputWidth(t *testing.T, out string, width int) {
	t.Helper()
	for line := range strings.SplitSeq(out, "\n") {
		if utf8.RuneCountInString(line) > width {
			t.Errorf("buffer output exceeded %d columns: %q", width, line)
		}
	}
}

func TestConfigureOutputCanBeReconfiguredWithoutNestedWriters(t *testing.T) {
	t.Parallel()
	cmd, out := configuredOutput(t, "table", "always")
	if err := command.ConfigureOutput(cmd, "json", "never"); err != nil {
		t.Fatal(err)
	}
	if err := command.WriteJSON(cmd.OutOrStdout(), json.RawMessage(outputEnvelope)); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out.Bytes()) || strings.Contains(out.String(), "\x1b") || !strings.Contains(out.String(), "90071992547409930001") {
		t.Fatal("reconfiguration retained colored table output")
	}
}

func TestConfigureOutputRejectsInvalidOptionsWithoutChangingWriter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ format, color string }{{"yaml", "auto"}, {"", "auto"}, {"json", "invalid"}, {"table", ""}} {
		t.Run(tc.format+"/"+tc.color, func(t *testing.T) {
			cmd := &cobra.Command{}
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := command.ConfigureOutput(cmd, tc.format, tc.color); err == nil {
				t.Fatal("invalid output configuration accepted")
			}
			if cmd.OutOrStdout() != &out || out.Len() != 0 {
				t.Fatal("invalid configuration wrote output or replaced the writer")
			}
		})
	}
}

func TestOutputNoColorAutoAndExplicitHelpStyles(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, color := range []string{"auto", "never", "always"} {
		t.Run(color, func(t *testing.T) {
			cmd, table := configuredOutput(t, "table", color)
			if err := command.WriteJSON(cmd.OutOrStdout(), json.RawMessage(outputEnvelope)); err != nil {
				t.Fatal(err)
			}
			root := &cobra.Command{Use: "fixture"}
			root.AddGroup(&cobra.Group{ID: "cloud", Title: "Cloud:"})
			root.AddCommand(&cobra.Command{Use: "cloud", Short: "Cloud commands", GroupID: "cloud"})
			root.PersistentFlags().StringVar(&color, "color", color, "Color policy")
			command.InstallHelp(root, &color)
			var help bytes.Buffer
			root.SetOut(&help)
			root.SetArgs([]string{"--help"})
			if err := root.ExecuteContext(t.Context()); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(table.String(), "\x1b[") != (color == "always") || strings.Contains(help.String(), "\x1b[") != (color == "always") {
				t.Fatal("NO_COLOR/explicit color policy changed table or help behavior")
			}
			if !strings.Contains(stripOutputANSI(help.String()), "Cloud:") || !strings.Contains(stripOutputANSI(help.String()), "Usage:") {
				t.Fatal("help styling removed command groups or usage")
			}
		})
	}
}

func stripOutputANSI(out string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(out, "")
}
