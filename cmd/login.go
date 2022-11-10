package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
	"github.com/zitadel/oidc/pkg/client/rp"
	"github.com/zitadel/oidc/pkg/oidc"
)

const (
	membershipUriFlag  = "membership-uri"
	baseServiceUriFlag = "service-uri"
)

func getRelyingParty(profile *internal.Profile) (rp.RelyingParty, error) {
	return rp.NewRelyingPartyOIDC(profile.GetMembershipURI(), internal.AuthClient, "",
		"", []string{"openid", "email", "offline_access", "supertoken"}, rp.WithHTTPClient(getHttpClient()))
}

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

func newLoginCommand() *cobra.Command {
	return newCommand("login",
		withStringFlag(membershipUriFlag, internal.DefaultMemberShipUri, "service url"),
		withStringFlag(baseServiceUriFlag, internal.DefaultBaseUri, "service url"),
		withHiddenFlag(membershipUriFlag),
		withHiddenFlag(baseServiceUriFlag),
		withRunE(func(cmd *cobra.Command, args []string) error {

			config, err := getConfig()
			if err != nil {
				return err
			}

			profile, err := getCurrentProfile(config)
			if err != nil {
				return err
			}

			relyingParty, err := getRelyingParty(profile)
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

			currentProfileName, err := getCurrentProfileName()
			if err != nil {
				return err
			}

			config.SetCurrentProfile(currentProfileName, profile)

			fmt.Fprintln(cmd.OutOrStdout(), "Logged!")
			return config.Persist()
		}),
	)
}
