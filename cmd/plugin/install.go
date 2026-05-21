package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
	pluginpkg "github.com/formancehq/fctl/v3/pkg/plugin"
)

const (
	versionFlag = "version"
	pathFlag    = "path"
)

func NewInstallCommand() *cobra.Command {
	return fctl.NewCommand("install",
		fctl.WithShortDescription("Install a plugin from the registry or local path"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithStringFlag(versionFlag, "", "Specific version to install"),
		fctl.WithStringFlag(pathFlag, "", "Install from a local path (binary or Go module directory)"),
		fctl.WithRunE(runInstall),
	)
}

func runInstall(cmd *cobra.Command, args []string) error {
	name := args[0]
	localPath := fctl.GetString(cmd, pathFlag)

	if localPath != "" {
		return installFromPath(cmd, name, localPath)
	}

	return installFromRegistry(cmd, name)
}

func installFromPath(cmd *cobra.Command, name, localPath string) error {
	configDir := fctl.GetString(cmd, fctl.ConfigDir)

	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("path %q not found: %w", absPath, err)
	}

	destPath := pluginpkg.PluginBinaryPath(configDir, name, "dev")

	if info.IsDir() {
		// Check if it's a Go module directory
		goMod := filepath.Join(absPath, "go.mod")
		if _, err := os.Stat(goMod); os.IsNotExist(err) {
			return fmt.Errorf("directory %q has no go.mod — expected a Go module or a binary", absPath)
		}

		pterm.Info.Printfln("Detected Go module. Building...")

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("failed to create plugin directory: %w", err)
		}

		buildCmd := exec.CommandContext(cmd.Context(), "go", "build", "-o", destPath, ".")
		buildCmd.Dir = absPath
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("go build failed: %w", err)
		}
	} else {
		// It's a binary — copy it
		pterm.Info.Printfln("Installing from binary...")

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("failed to create plugin directory: %w", err)
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("failed to read binary: %w", err)
		}

		if err := os.WriteFile(destPath, data, 0o755); err != nil {
			return fmt.Errorf("failed to write plugin binary: %w", err)
		}
	}

	// Fetch and cache manifest
	loaded, err := pluginpkg.LoadPlugin(name, destPath)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}
	defer loaded.Kill()

	manifest, err := loaded.Client.GetManifest(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}

	if err := pluginpkg.SaveCachedManifest(configDir, name, "dev", manifest); err != nil {
		return fmt.Errorf("failed to cache manifest: %w", err)
	}

	// Save config
	cfg, err := pluginpkg.LoadPluginsConfig(configDir)
	if err != nil {
		return err
	}

	cfg.AddPluginVersion(name, "dev", pluginpkg.InstalledPluginVersion{
		CompatibleWith: ">= 0.0.0",
		Path:           destPath,
	})

	if err := pluginpkg.SavePluginsConfig(configDir, cfg); err != nil {
		return err
	}

	pterm.Success.Printfln("Plugin %q installed (version: dev)", name)
	return nil
}

func installFromRegistry(cmd *cobra.Command, name string) error {
	version := fctl.GetString(cmd, versionFlag)
	configDir := fctl.GetString(cmd, fctl.ConfigDir)

	registry := pluginpkg.NewRegistryClient(fctl.GetHttpClient(cmd))
	pm := pluginpkg.NewPluginManager(configDir, false)

	reg, err := registry.FetchRegistry()
	if err != nil {
		return fmt.Errorf("failed to fetch registry: %w", err)
	}

	pluginInfo, ok := reg.Plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q not found in registry", name)
	}

	if version == "" {
		version, _, err = pluginInfo.FindBestVersion("999.999.999")
		if err != nil {
			return fmt.Errorf("no versions available for plugin %q", name)
		}
	}

	pterm.Info.Printfln("Installing plugin %s v%s...", name, version)

	if err := pm.InstallFromRegistry(name, version, pluginInfo, registry); err != nil {
		return fmt.Errorf("failed to install plugin %s: %w", name, err)
	}

	pterm.Success.Printfln("Plugin %s v%s installed successfully", name, version)
	return nil
}
