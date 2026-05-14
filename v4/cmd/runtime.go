package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	v4config "github.com/formancehq/fctl/v4/internal/config"
	"github.com/formancehq/fctl/v4/internal/credentials"
	"github.com/formancehq/fctl/v4/internal/runtime"
)

func runtimeFromCommand(cmd *cobra.Command) (*runtime.Runtime, error) {
	path, err := configPath(cmd)
	if err != nil {
		return nil, err
	}
	contextName, err := contextNameFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	return runtime.New(cmd.Context(), runtime.Options{
		ConfigPath:      path,
		ContextOverride: v4config.ContextOverride{Name: contextName},
		Credentials:     credentials.NewMemoryStore(),
		Manifest:        capabilities.GeneratedManifest,
		Compatibility:   capabilities.DefaultComponentCompatibility,
	})
}

func contextNameFromCommand(cmd *cobra.Command) (string, error) {
	flags := cmd.Root().PersistentFlags()
	contextName, err := flags.GetString(contextFlag)
	if err != nil {
		return "", err
	}
	profileName, err := flags.GetString(profileFlag)
	if err != nil {
		return "", err
	}
	if contextName != "" && profileName != "" {
		return "", fmt.Errorf("--profile and --context are mutually exclusive")
	}
	if profileName != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "Flag --profile has been deprecated, use --context")
		return profileName, nil
	}
	return contextName, nil
}
