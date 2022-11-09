package cmd

import (
	"os/exec"
	"runtime"

	fctl "github.com/formancehq/fctl/cmd/internal"
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

			organization, err := resolveOrganizationID(cmd)
			if err != nil {
				return err
			}

			stack, err := resolveStackID(cmd, organization)
			if err != nil {
				return err
			}

			profile, err := getCurrentProfile()
			if err != nil {
				return err
			}

			stackUrl, err := fctl.ServicesBaseUrl(*profile, organization, stack)
			if err != nil {
				return err
			}

			return errors.Wrapf(openUrl(stackUrl.String()), "opening url: %s", stackUrl.String())
		}),
	)
}
