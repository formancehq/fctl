package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	v4config "github.com/formancehq/fctl/v4/internal/config"
	"github.com/formancehq/fctl/v4/internal/credentials"
)

func newConfigCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "Manage fctl v4 configuration",
	}
	command.AddCommand(newConfigMigrateV3Command())
	return command
}

func newConfigMigrateV3Command() *cobra.Command {
	var fromDir string
	var dryRun bool
	var credentialDir string

	command := &cobra.Command{
		Use:   "migrate-v3",
		Short: "Migrate fctl v3 profiles into v4 contexts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if fromDir == "" {
				defaultDir, err := defaultV3ConfigDir()
				if err != nil {
					return err
				}
				fromDir = defaultDir
			}

			state, err := v4config.LoadV3State(fromDir)
			if err != nil {
				return err
			}
			plan, err := v4config.PlanV3Migration(state)
			if err != nil {
				return err
			}

			if dryRun {
				return renderMigrationPlan(cmd, plan)
			}

			var store credentials.Store
			if len(plan.CredentialMoves) > 0 {
				if credentialDir == "" {
					return fmt.Errorf("--credential-dir is required to migrate v3 tokens without a keyring backend")
				}
				store = credentials.NewInsecureFileStore(credentialDir)
			}
			path, err := configPath(cmd)
			if err != nil {
				return err
			}
			if err := v4config.WriteMigration(cmd.Context(), path, plan, store); err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, map[string]any{
				"configPath":      path,
				"contexts":        len(plan.Contexts),
				"credentialMoves": len(plan.CredentialMoves),
			}); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Migrated %d context(s) to %s.\n", len(plan.Contexts), path)
			return err
		},
	}

	command.Flags().StringVar(&fromDir, "from", "", "Path to the fctl v3 configuration directory (default $HOME/.config/formance/fctl)")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show the migration plan without writing v4 config")
	command.Flags().StringVar(&credentialDir, "credential-dir", "", "Explicit insecure credential directory for migrated v3 tokens")

	return command
}

func defaultV3ConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory for default v3 config path: %w", err)
	}
	return filepath.Join(homeDir, ".config", "formance", "fctl"), nil
}

func renderMigrationPlan(cmd *cobra.Command, plan v4config.MigrationPlan) error {
	result := migrationPlanOutput{
		CurrentContext:  plan.CurrentContext,
		Contexts:        contextNames(plan.Contexts),
		CredentialMoves: len(plan.CredentialMoves),
	}
	if handled, err := writeStructuredOutput(cmd, result); handled || err != nil {
		return err
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Current context: %s\n", plan.CurrentContext); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Contexts:"); err != nil {
		return err
	}
	for _, name := range contextNames(plan.Contexts) {
		context := plan.Contexts[name]
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "- %s (%s)\n", name, context.Kind); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Credential moves: %d\n", len(plan.CredentialMoves))
	return err
}

type migrationPlanOutput struct {
	CurrentContext  string   `json:"currentContext" yaml:"currentContext"`
	Contexts        []string `json:"contexts" yaml:"contexts"`
	CredentialMoves int      `json:"credentialMoves" yaml:"credentialMoves"`
}

func contextNames(contexts map[string]v4config.Context) []string {
	cfg := v4config.Config{Contexts: contexts}
	return cfg.ContextNames()
}
