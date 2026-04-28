package fctl

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const (
	DefaultMembershipURI = "https://app.formance.cloud/api"
	DefaultConsoleURL    = "https://portal.formance.cloud"
)

type Config struct {
	CurrentProfile string `json:"currentProfile"`
	UniqueID       string `json:"uniqueID,omitempty"`
}

func GetCurrentProfileName(cmd *cobra.Command, config Config) string {
	if profile := GetString(cmd, ProfileFlag); profile != "" {
		return profile
	}
	currentProfileName := config.CurrentProfile
	if currentProfileName == "" {
		currentProfileName = "default"
	}
	return currentProfileName
}

func LoadConfigDir(cmd *cobra.Command) string {
	return GetString(cmd, ConfigDir)
}

func LoadConfigFilePath(cmd *cobra.Command) string {
	return GetFilePath(cmd, "config.yml")
}

func GetFilePath(cmd *cobra.Command, filename string) string {
	return filepath.Join(LoadConfigDir(cmd), filename)
}

func UpsertConfigDir(cmd *cobra.Command) error {
	return os.MkdirAll(LoadConfigDir(cmd), 0700)
}

func LoadConfig(cmd *cobra.Command) (*Config, error) {
	v, err := ReadJSONFile[Config](cmd, "config.yml")
	if os.IsNotExist(err) {
		return &Config{
			CurrentProfile: "default",
		}, nil
	}
	return v, err
}

func WriteConfig(cmd *cobra.Command, config Config) error {
	if err := UpsertConfigDir(cmd); err != nil {
		return err
	}

	return WriteJSONFile(LoadConfigFilePath(cmd), config)
}
