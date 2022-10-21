package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/fctl/pkg/membership"
	membershipclient "github.com/numary/membership-api/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zitadel/oidc/pkg/client/rp"
	"golang.org/x/oauth2"
)

const (
	profileFlag     = "profile"
	configFileFlag  = "config"
	debugFlag       = "debug"
	insecureTlsFlag = "insecure-tls"
)

var (
	currentProfileName string
	currentProfile     *fctl.Profile
	configManager      *fctl.ConfigManager
	config             *fctl.Config
	httpClient         = &http.Client{
		Transport: http.DefaultTransport,
	}
	apiClient *membershipclient.APIClient
)

func init() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
}

func newRootCommand() *cobra.Command {
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return newCommand("fctl",
		withShortDescription("Formance Control CLI"),
		withSilenceUsage(),
		withPersistentPreRunE(func(cmd *cobra.Command, args []string) (err error) {
			if err = viper.BindPFlags(cmd.Flags()); err != nil {
				return err
			}

			if viper.GetBool(insecureTlsFlag) {
				httpTransport := http.DefaultTransport.(*http.Transport)
				httpTransport.TLSClientConfig = &tls.Config{
					InsecureSkipVerify: true,
				}
				httpClient = &http.Client{
					Transport: httpTransport,
				}
			}

			configManager = fctl.NewConfigManager(viper.GetString(configFileFlag))

			config, err = configManager.Load()
			if err != nil {
				return err
			}

			currentProfileName = config.CurrentProfile
			if currentProfileName == "" {
				currentProfileName = "default"
				config.CurrentProfile = currentProfileName
			}
			debugln(cmd.OutOrStdout(), "Current profile:", currentProfileName)
			if selectedProfile := viper.GetString(profileFlag); selectedProfile != "" {
				debugln(cmd.OutOrStdout(), "Override profile by flag:", selectedProfile)
				currentProfileName = selectedProfile
			}

			currentProfile = config.GetProfileOrDefault(currentProfileName, &fctl.Profile{
				MembershipURI:  viper.GetString(membershipUriFlag),
				BaseServiceURI: viper.GetString(baseServiceUriFlag),
			})
			debugln(cmd.OutOrStdout(), "Selected profile membership uri:", currentProfile.MembershipURI)
			debugln(cmd.OutOrStdout(), "Selected base service uri:", currentProfile.BaseServiceURI)

			ifDebug(func() {
				debugln(cmd.OutOrStdout(), "Configure http round tripper logger")
				httpClient.Transport = fctl.DebugRoundTripper(httpClient.Transport)
			})
			if currentProfile.Token != nil {
				if currentProfile.Token.Expiry.Before(time.Now()) {
					debugln(cmd.OutOrStdout(), "Detect expired auth token against membership, trying to refresh token")
					relyingParty, err := rp.NewRelyingPartyOIDC(currentProfile.MembershipURI, authClient, "",
						"", []string{"openid", "email", "offline_access"}, rp.WithHTTPClient(httpClient))
					if err != nil {
						return err
					}

					newToken, err := relyingParty.OAuthConfig().
						TokenSource(context.WithValue(context.TODO(), oauth2.HTTPClient, httpClient), currentProfile.Token).
						Token()
					if err != nil {
						return err
					}

					currentProfile.Token = newToken

					if err := configManager.UpdateConfig(config); err != nil {
						return err
					}

				} else {
					debugln(cmd.OutOrStdout(), "Detect active auth token against membership, reuse it")
				}
			}

			apiClient = membership.NewClient(*currentProfile, httpClient, viper.GetBool(debugFlag))
			return nil
		}),
		withChildCommands(
			newLedgerCommand(),
			newPaymentsCommand(),
			newProfilesCommand(),
			newSandboxCommand(),
			newUICommand(),
			newVersionCommand(),
			newLoginCommand(),
		),
		withPersistentStringPFlag(profileFlag, "p", "", "config profile to use"),
		withPersistentStringPFlag(configFileFlag, "c", fmt.Sprintf("%s/.formance/fctl.config", homedir), "Debug mode"),
		withPersistentBoolPFlag(debugFlag, "d", false, "Debug mode"),
		withPersistentBoolFlag(insecureTlsFlag, false, "Insecure TLS"),
	)
}

func Execute() {
	_ = newRootCommand().Execute()
}
