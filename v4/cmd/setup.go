package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type setupOutput struct {
	Commands []string `json:"commands" yaml:"commands"`
}

func newSetupCommand(deprecatedPromptAlias bool) *cobra.Command {
	command := &cobra.Command{
		Use:   "setup",
		Short: "Show setup commands",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if deprecatedPromptAlias {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command prompt has been deprecated, use setup or login")
			}
			return renderSetupGuidance(cmd)
		},
	}
	if deprecatedPromptAlias {
		command.Use = "prompt"
		command.Hidden = true
		command.Deprecated = "use setup or login"
	}
	return command
}

func newContextWizardCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "wizard",
		Short: "Show context setup commands",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return renderSetupGuidance(cmd)
		},
	}
}

func renderSetupGuidance(cmd *cobra.Command) error {
	output := setupOutput{
		Commands: []string{
			"fctl login",
			"fctl profile create stack <name> --stack-url <url>",
			"fctl profile use <name>",
			"fctl config migrate-v3",
		},
	}
	if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
		return err
	}
	for _, command := range output.Commands {
		line := command
		if terminalOutputEnabled(cmd) {
			line = styledInfoLine(cmd, "Run", command)
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), line); err != nil {
			return err
		}
	}
	return nil
}
