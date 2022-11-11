package cmd

import (
	"os/exec"
	"runtime"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func openUrl(url string) error {
	var (
		cmd  string
		args []string
	)

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func NewUICommand() *cobra.Command {
	return cmdbuilder.NewStackCommand("ui",
		cmdbuilder.WithShortDescription("Open UI"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get()
			if err != nil {
				return err
			}

			organization, err := cmdbuilder.ResolveOrganizationID(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			stack, err := cmdbuilder.ResolveStackID(cmd.Context(), cfg, organization)
			if err != nil {
				return err
			}

			profile := config.GetCurrentProfile(cfg)
			stackUrl := profile.ServicesBaseUrl(organization, stack)

			return errors.Wrapf(openUrl(stackUrl.String()), "opening url: %s", stackUrl.String())
		}),
	)
}
