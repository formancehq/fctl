package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/formancehq/fctl/cmd/auth"
	"github.com/formancehq/fctl/cmd/cloud"
	"github.com/formancehq/fctl/cmd/cloud/me"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/ledger"
	"github.com/formancehq/fctl/cmd/payments"
	"github.com/formancehq/fctl/cmd/profiles"
	"github.com/formancehq/fctl/cmd/stack"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewRootCommand() *cobra.Command {
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return cmdbuilder.NewCommand("fctl",
		cmdbuilder.WithSilenceError(),
		cmdbuilder.WithShortDescription("Formance Control CLI"),
		cmdbuilder.WithSilenceUsage(),
		cmdbuilder.WithPersistentPreRunE(func(cmd *cobra.Command, args []string) (err error) {
			viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
			viper.AutomaticEnv()
			return viper.BindPFlags(cmd.Flags())
		}),
		cmdbuilder.WithChildCommands(
			NewUICommand(),
			NewVersionCommand(),
			NewLoginCommand(),
			NewPromptCommand(),
			ledger.NewCommand(),
			payments.NewCommand(),
			profiles.NewCommand(),
			stack.NewCommand(),
			auth.NewCommand(),
			cloud.NewCommand(),
			me.NewInfoCommand(),
		),
		cmdbuilder.WithPersistentStringPFlag(config.ProfileFlag, "p", "", "config profile to use"),
		cmdbuilder.WithPersistentStringPFlag(config.FileFlag, "c", fmt.Sprintf("%s/.formance/fctl.config", homedir), "Debug mode"),
		cmdbuilder.WithPersistentBoolPFlag(config.DebugFlag, "d", false, "Debug mode"),
		cmdbuilder.WithPersistentBoolFlag(config.InsecureTlsFlag, false, "Insecure TLS"),
	)
}

func Execute() {
	err := NewRootCommand().Execute()
	if err != nil {
		cmdbuilder.Error(os.Stderr, err.Error())
	}
}
