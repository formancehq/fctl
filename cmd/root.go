package cmd

import (
	"fmt"
	"os"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRootCommand() *cobra.Command {
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	const (
		profileFlag     = "profile"
		configFileFlag  = "config"
		debugFlag       = "debug"
		insecureTlsFlag = "insecure-tls"
	)

	return newCommand("fctl",
		withShortDescription("Formance Control CLI"),
		withSilenceUsage(),
		withPersistentPreRunE(func(cmd *cobra.Command, args []string) (err error) {
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return err
			}

			var (
				config             *fctl.Config
				configManager      *fctl.ConfigManager
				currentProfileName string
			)

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
			fctl.DebugLn(cmd.Context(), cmd.OutOrStdout(), "Current profile:", currentProfileName)
			if selectedProfile := viper.GetString(profileFlag); selectedProfile != "" {
				fctl.DebugLn(cmd.Context(), cmd.OutOrStdout(), "Override profile by flag:", selectedProfile)
				currentProfileName = selectedProfile
			}

			currentProfile := config.GetProfileOrDefault(currentProfileName, &fctl.Profile{
				MembershipURI:  viper.GetString(membershipUriFlag),
				BaseServiceURI: viper.GetString(baseServiceUriFlag),
			})
			fctl.DebugLn(cmd.Context(), cmd.OutOrStdout(), "Selected profile membership uri:", currentProfile.MembershipURI)
			fctl.DebugLn(cmd.Context(), cmd.OutOrStdout(), "Selected base service uri:", currentProfile.BaseServiceURI)

			ctx := cmd.Context()
			ctx = fctl.WithCurrentProfile(ctx, currentProfile)
			ctx = fctl.WithConfig(ctx, config)
			ctx = fctl.WithConfigManager(ctx, configManager)
			ctx = fctl.WithCurrentProfileName(ctx, currentProfileName)
			ctx = fctl.WithDebug(ctx, viper.GetBool(debugFlag))
			cmd.SetContext(ctx)

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
			newAuthCommand(),
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
