package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	v4config "github.com/formancehq/fctl/v4/internal/config"
)

const configFilename = "config.yaml"

func configPath(cmd *cobra.Command) (string, error) {
	configDir, err := cmd.Root().PersistentFlags().GetString(configDirFlag)
	if err != nil {
		return "", err
	}
	if configDir == "" {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve user config directory: %w", err)
		}
		configDir = filepath.Join(userConfigDir, "formance", "fctl")
	}
	return filepath.Join(configDir, configFilename), nil
}

func loadConfig(cmd *cobra.Command, allowMissing bool) (v4config.Config, string, error) {
	path, err := configPath(cmd)
	if err != nil {
		return v4config.Config{}, "", err
	}

	cfg, err := v4config.LoadFile(path)
	if err != nil {
		if allowMissing && errors.Is(err, os.ErrNotExist) {
			return v4config.New(), path, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return v4config.Config{}, "", missingConfigError(path)
		}
		return v4config.Config{}, "", err
	}
	return cfg, path, nil
}

func missingConfigError(path string) error {
	return fmt.Errorf("v4 config not found at %s; run `fctl login`, create a profile with `fctl profile create stack <name> --stack-url <url>`, `fctl profile create cloud <name> --cloud-url <url>`, or `fctl profile create cloud-stack <name> --cloud-url <url> --organization <org> --stack <stack>`; alternatively migrate an existing v3 config with `fctl config migrate-v3`", path)
}

func outputFormat(cmd *cobra.Command) (string, error) {
	return cmd.Root().PersistentFlags().GetString(outputFlag)
}
