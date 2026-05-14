package cmd

import (
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
	contextName, err := cmd.Root().PersistentFlags().GetString(contextFlag)
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
