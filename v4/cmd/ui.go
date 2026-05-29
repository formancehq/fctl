package cmd

import (
	"fmt"
	"net/url"
	"os/exec"
	goruntime "runtime"

	"github.com/spf13/cobra"

	cloudcmd "github.com/formancehq/fctl/v4/internal/commands/cloud"
)

var openBrowserURL = openURL

func newUICommand(deprecatedRootAlias bool) *cobra.Command {
	var printOnly bool

	command := &cobra.Command{
		Use:   "ui",
		Short: "Open the Formance Cloud console",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if deprecatedRootAlias {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command ui has been deprecated, use cloud ui")
			}
			_, client, err := cloudRuntimeAndMembershipClientFromCommand(cmd)
			if err != nil {
				return err
			}
			response, err := client.GetServerInfo(cmd.Context())
			if err != nil {
				return err
			}
			consoleURL := cloudcmd.ConsoleURL(response)

			nonInteractive, err := cmd.Root().PersistentFlags().GetBool(nonInteractiveFlag)
			if err != nil {
				return err
			}
			opened := false
			if !printOnly && !nonInteractive {
				if err := openBrowserURL(consoleURL); err != nil {
					return err
				}
				opened = true
			}

			output := uiOutput{URL: consoleURL, Opened: opened}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			if opened {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), styledInfoLine(cmd, "Opening console", consoleURL))
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), styledInfoLine(cmd, "Console URL", consoleURL))
			return err
		},
	}
	if deprecatedRootAlias {
		command.Hidden = true
	}
	command.Flags().BoolVar(&printOnly, "print", false, "Print the console URL without opening a browser")
	return command
}

func openURL(rawURL string) error {
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return fmt.Errorf("invalid URL: %s", rawURL)
	}

	var command string
	var args []string
	switch goruntime.GOOS {
	case "windows":
		command = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		command = "open"
	default:
		command = "xdg-open"
	}
	args = append(args, rawURL)
	return exec.Command(command, args...).Start() //nolint:gosec // URL opener is selected by OS and URL is parsed above.
}

type uiOutput struct {
	URL    string `json:"url" yaml:"url"`
	Opened bool   `json:"opened" yaml:"opened"`
}
