package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	contextFlag        = "context"
	configDirFlag      = "config-dir"
	outputFlag         = "output"
	nonInteractiveFlag = "non-interactive"
)

// NewRootCommand builds the v4 command tree. Keep this package focused on
// parsing and dispatch; runtime work belongs under internal packages.
func NewRootCommand(version string) *cobra.Command {
	if version == "" {
		version = "dev"
	}

	root := &cobra.Command{
		Use:           "fctl",
		Short:         "Formance Control CLI v4",
		Long:          "Formance Control CLI v4 targets Cloud, self-hosted, and local Formance stacks through explicit contexts.",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	root.SetVersionTemplate("fctl v4 {{.Version}}\n")
	root.PersistentFlags().String(contextFlag, "", "Context to use")
	root.PersistentFlags().String(configDirFlag, "", "Path to the v4 configuration directory")
	root.PersistentFlags().StringP(outputFlag, "o", "plain", "Output format (plain, json, yaml)")
	root.PersistentFlags().Bool(nonInteractiveFlag, false, "Disable interactive prompts")

	root.AddCommand(newVersionCommand())
	root.AddCommand(newContextCommand())
	root.AddCommand(newProfilesCommand())
	root.AddCommand(newConfigCommand())
	root.AddCommand(newSetupCommand(false))
	root.AddCommand(newSetupCommand(true))
	root.AddCommand(newTargetCommand())
	root.AddCommand(newCloudCommand())
	root.AddCommand(newCloudStacksCommand("cloud_stacks", "cloud stacks", true))
	root.AddCommand(newCloudStacksCommand("stack", "cloud stacks", true))
	root.AddCommand(newCloudStacksCommand("stacks", "cloud stacks", true))
	root.AddCommand(newLedgerCommand())
	root.AddCommand(newPaymentsCommand())
	root.AddCommand(newWalletsCommand())
	root.AddCommand(newFlowsCommand(false))
	root.AddCommand(newFlowsCommand(true))
	root.AddCommand(newReconciliationCommand())
	root.AddCommand(newAuthCommand())
	root.AddCommand(newWebhooksCommand())

	return root
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the fctl v4 version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "fctl v4 %s\n", cmd.Root().Version)
			return err
		},
	}
}

// Execute runs the v4 command tree.
func Execute(version string) {
	root := NewRootCommand(version)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
