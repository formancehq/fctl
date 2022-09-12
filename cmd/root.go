package cmd

import (
	"fmt"
	"os"
	"strings"

	fctl "github.com/numary/fctl/pkg"
	"github.com/numary/fctl/pkg/membership"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	profileFlag        = "profile"
	configFileFlag     = "config"
	debugFlag          = "debug"
	membershipUriFlag  = "membership-uri"
	baseServiceUriFlag = "service-uri"
)

var (
	currentProfileName string
	currentProfile     *fctl.Profile
	configManager      *fctl.ConfigManager
	config             *fctl.Config
)

var rootCommand = &cobra.Command{
	Use:          "fctl",
	Long:         "Formance Control CLI",
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		if err = viper.BindPFlags(cmd.Flags()); err != nil {
			return err
		}
		configManager = fctl.NewConfigManager(viper.GetString(configFileFlag))
		currentProfileName = viper.GetString(profileFlag)

		config, err = configManager.Load()
		if err != nil {
			return err
		}
		currentProfile = config.GetProfileOrDefault(currentProfileName, &fctl.Profile{
			MembershipURI:  viper.GetString(membershipUriFlag),
			BaseServiceURI: viper.GetString(baseServiceUriFlag),
		})

		apiClient = membership.NewClient(*currentProfile, viper.GetBool(debugFlag))
		return nil
	},
}

func init() {
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	rootCommand.PersistentFlags().StringP(profileFlag, "p", "default", "config profile to use")
	rootCommand.PersistentFlags().StringP(configFileFlag, "c", fmt.Sprintf("%s/.formance/fctl.config", homedir), "Debug mode")
	rootCommand.PersistentFlags().String(membershipUriFlag, fctl.DefaultMemberShipUri, "service url")
	rootCommand.PersistentFlags().String(baseServiceUriFlag, fctl.DefaultBaseUri, "service url")
	rootCommand.PersistentFlags().BoolP(debugFlag, "d", false, "Debug mode")
	_ = rootCommand.PersistentFlags().MarkHidden(membershipUriFlag)
	_ = rootCommand.PersistentFlags().MarkHidden(baseServiceUriFlag)

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
}

func Execute() {
	_ = rootCommand.Execute()
}
