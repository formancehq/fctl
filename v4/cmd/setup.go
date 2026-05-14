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
				fmt.Fprintln(cmd.ErrOrStderr(), "Command prompt has been deprecated, use setup or context wizard")
			}
			return renderSetupGuidance(cmd)
		},
	}
	if deprecatedPromptAlias {
		command.Use = "prompt"
		command.Hidden = true
		command.Deprecated = "use setup or context wizard"
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
			"fctl context create stack <name> --stack-url <url>",
			"fctl context use <name>",
			"fctl config migrate-v3 --from <path>",
		},
	}
	if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
		return err
	}
	for _, command := range output.Commands {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), command); err != nil {
			return err
		}
	}
	return nil
}
