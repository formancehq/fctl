package plugin

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
	pluginpkg "github.com/formancehq/fctl/v3/pkg/plugin"
)

func NewListCommand() *cobra.Command {
	return fctl.NewCommand("list",
		fctl.WithAliases("ls"),
		fctl.WithShortDescription("List installed plugins"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithRunE(runList),
	)
}

func runList(cmd *cobra.Command, args []string) error {
	configDir := fctl.GetString(cmd, fctl.ConfigDir)

	cfg, err := pluginpkg.LoadPluginsConfig(configDir)
	if err != nil {
		return err
	}

	if len(cfg.Plugins) == 0 {
		pterm.Info.Println("No plugins installed")
		pterm.Info.Println("Use 'fctl plugin install <name>' to install a plugin")
		return nil
	}

	tableData := [][]string{{"Name", "Version", "Path"}}
	for _, p := range cfg.Plugins {
		tableData = append(tableData, []string{p.Name, p.Version, p.Path})
	}

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
