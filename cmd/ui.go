package cmd

import (
	"os/exec"
	"runtime"

	"github.com/formancehq/fctl/cmd/internal"
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
	return internal.NewStackCommand("ui",
		internal.WithShortDescription("Open UI"),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			organization, err := internal.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			stack, err := internal.ResolveStack(cmd, cfg, organization)
			if err != nil {
				return err
			}

			profile := internal.GetCurrentProfile(cmd, cfg)
			stackUrl := profile.ServicesBaseUrl(stack)

			return errors.Wrapf(openUrl(stackUrl.String()), "opening url: %s", stackUrl.String())
		}),
	)
}
