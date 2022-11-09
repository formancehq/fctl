package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/formancehq/fctl/cmd/internal"
	membershipclient "github.com/numary/membership-api/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	profileFlag     = "profile"
	configFileFlag  = "config"
	debugFlag       = "debug"
	insecureTlsFlag = "insecure-tls"
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

func getCurrentProfile() (*internal.Profile, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	profileName, err := getCurrentProfileName()
	if err != nil {
		return nil, err
	}
	return config.GetProfileOrDefault(profileName, viper.GetString(membershipUriFlag),
		viper.GetString(baseServiceUriFlag)), nil
}

func newMembershipClient(cmd *cobra.Command) (*membershipclient.APIClient, error) {
	profile, err := getCurrentProfile()
	if err != nil {
		return nil, err
	}
	return internal.NewMembershipClientFromContext(cmd.Context(), profile, getHttpClient())
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
