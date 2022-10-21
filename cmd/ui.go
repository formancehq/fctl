package cmd

import (
	"os/exec"
	"runtime"

	fctl "github.com/formancehq/fctl/pkg"
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

func newUICommand() *cobra.Command {
	return newStackCommand("ui",
		withShortDescription("Open UI"),
		withRunE(func(cmd *cobra.Command, args []string) error {
			organization, stack, err := findDefaultStackAndOrganizationId(cmd.Context())
			if err != nil {
				return err
			}

			stackUrl, err := fctl.ServicesBaseUrl(*currentProfile, organization, stack)
			if err != nil {
				return err
			}

			return errors.Wrapf(openUrl(stackUrl.String()), "opening url: %s", stackUrl.String())
		}),
	)
}
