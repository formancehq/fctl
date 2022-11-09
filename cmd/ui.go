package cmd

import (
	"os/exec"
	"runtime"

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

			config, err := getConfig()
			if err != nil {
				return err
			}

			organization, err := resolveOrganizationID(cmd, config)
			if err != nil {
				return err
			}

			stack, err := resolveStackID(cmd, config, organization)
			if err != nil {
				return err
			}

			profile, err := getCurrentProfile(config)
			if err != nil {
				return err
			}

			stackUrl := profile.ServicesBaseUrl(organization, stack)

			return errors.Wrapf(openUrl(stackUrl.String()), "opening url: %s", stackUrl.String())
		}),
	)
}
