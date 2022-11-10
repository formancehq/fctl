package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	profileFlag     = "profile"
	configFileFlag  = "config"
	debugFlag       = "debug"
	insecureTlsFlag = "insecure-tls"
	// TODO: Make configurable at build

)

func getConfigManager() *internal.ConfigManager {
	return internal.NewConfigManager(viper.GetString(configFileFlag))
}

func getConfig() (*internal.Config, error) {
	return getConfigManager().Load()
}

func getCurrentProfileName() (string, error) {
	if profile := viper.GetString(profileFlag); profile != "" {
		return profile, nil
	}
	config, err := getConfig()
	if err != nil {
		return "", err
	}
	currentProfileName := config.GetCurrentProfileName()
	if currentProfileName == "" {
		currentProfileName = "default"
	}
	return currentProfileName, nil
}

func getCurrentProfile(config *internal.Config) (*internal.Profile, error) {
	profileName, err := getCurrentProfileName()
	if err != nil {
		return nil, err
	}
	return config.GetProfileOrDefault(profileName, viper.GetString(membershipUriFlag),
		viper.GetString(baseServiceUriFlag)), nil
}

func newMembershipClient(cmd *cobra.Command, config *internal.Config) (*membershipclient.APIClient, error) {
	profile, err := getCurrentProfile(config)
	if err != nil {
		return nil, err
	}

	httpClient := getHttpClient()
	configuration := membershipclient.NewConfiguration()
	token, err := profile.GetToken(cmd.Context(), httpClient)
	if err != nil {
		return nil, err
	}
	configuration.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	configuration.HTTPClient = httpClient
	configuration.Servers[0].URL = profile.GetMembershipURI()
	return membershipclient.NewAPIClient(configuration), nil
}

func getHttpClient() *http.Client {
	return internal.NewHTTPClient(viper.GetBool(insecureTlsFlag), viper.GetBool(debugFlag))
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
			viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
			viper.AutomaticEnv()
			return viper.BindPFlags(cmd.Flags())
		}),
		withChildCommands(
			newLedgerCommand(),
			newPaymentsCommand(),
			newProfilesCommand(),
			newSandboxCommand(),
			newUICommand(),
			newVersionCommand(),
			newLoginCommand(),
			newAuthCommand(),
			newOrganizationsCommand(),
			newWhoamiCommand(),
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
