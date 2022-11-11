package subcmds

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	internal2 "github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/spf13/cobra"
	"github.com/zitadel/oidc/pkg/client/rp"
	"github.com/zitadel/oidc/pkg/oidc"
)

func open(url string) error {
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

type Dialog interface {
	DisplayURIAndCode(uri, code string)
}
type DialogFn func(uri, code string)

func (fn DialogFn) DisplayURIAndCode(uri, code string) {
	fn(uri, code)
}

func LogIn(ctx context.Context, dialog Dialog, relyingParty rp.RelyingParty) (*oidc.Tokens, error) {
	deviceCode, err := rp.GetDeviceCode(ctx, relyingParty)
	if err != nil {
		return nil, err
	}

	uri, err := url.Parse(deviceCode.GetVerificationUri())
	if err != nil {
		panic(err)
	}
	query := uri.Query()
	query.Set("user_code", deviceCode.GetUserCode())
	uri.RawQuery = query.Encode()

	dialog.DisplayURIAndCode(deviceCode.GetVerificationUri(), deviceCode.GetUserCode())

	if err := open(uri.String()); err != nil {
		return nil, err
	}

	return rp.PollDeviceCode(ctx, deviceCode.GetDeviceCode(), deviceCode.GetInterval(), relyingParty)
}

func NewLoginCommand() *cobra.Command {
	return cmdbuilder.NewCommand("login",
		cmdbuilder.WithStringFlag(internal2.MembershipUriFlag, internal2.DefaultMemberShipUri, "service url"),
		cmdbuilder.WithStringFlag(internal2.BaseServiceUriFlag, internal2.DefaultBaseUri, "service url"),
		cmdbuilder.WithHiddenFlag(internal2.MembershipUriFlag),
		cmdbuilder.WithHiddenFlag(internal2.BaseServiceUriFlag),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			config, err := internal2.Get()
			if err != nil {
				return err
			}

			profile, err := internal2.GetCurrentProfile(config)
			if err != nil {
				return err
			}

			relyingParty, err := membership.GetRelyingParty(profile)
			if err != nil {
				return err
			}

			ret, err := LogIn(cmd.Context(), DialogFn(func(uri, code string) {
				fmt.Fprintln(cmd.OutOrStdout(), "Please enter the following code on your browser:", code)
				fmt.Fprintln(cmd.OutOrStdout(), "Link:", uri)
			}), relyingParty)
			if err != nil {
				return err
			}

			profile.UpdateToken(ret.Token)

			currentProfileName, err := internal2.GetCurrentProfileName()
			if err != nil {
				return err
			}

			config.SetCurrentProfile(currentProfileName, profile)

			fmt.Fprintln(cmd.OutOrStdout(), "Logged!")
			return config.Persist()
		}),
	)
}
