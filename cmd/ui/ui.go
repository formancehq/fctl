package ui

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

type UiStruct struct {
	UIUrl        string `json:"stackUrl"`
	FoundBrowser bool   `json:"browserFound"`
}

type UiController struct {
	store *UiStruct
}

var _ fctl.Controller[*UiStruct] = (*UiController)(nil)

func NewDefaultUiStore() *UiStruct {
	return &UiStruct{
		UIUrl:        fctl.DefaultConsoleURL,
		FoundBrowser: false,
	}
}

func NewUiController() *UiController {
	return &UiController{
		store: NewDefaultUiStore(),
	}
}

func openUrl(urlString string) error {
	var (
		cmd  string
		args []string
	)

	if _, err := url.Parse(urlString); err != nil {
		return fmt.Errorf("invalid URL: %s", urlString)
	}

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, urlString)
	return exec.Command(cmd, args...).Start() //nolint:gosec
}

func (c *UiController) GetStore() *UiStruct {
	return c.store
}

func (c *UiController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	_, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	serverInfo, err := fctl.MembershipServerInfo(cmd.Context(), apiClient)
	if err != nil {
		return nil, err
	}

	if v := serverInfo.GetConsoleURL(); v != nil {
		c.store.UIUrl = *v
	}

	if err := openUrl(c.store.UIUrl); err != nil {
		c.store.FoundBrowser = true
	}

	return c, nil
}

func (c *UiController) Render(cmd *cobra.Command, args []string) error {
	fmt.Println("Opening url: ", c.store.UIUrl)

	return nil
}

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("ui",
		fctl.WithShortDescription("Open UI"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*UiStruct](NewUiController()),
	)
}
