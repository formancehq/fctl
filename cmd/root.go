package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/subcmds"
	"github.com/formancehq/fctl/cmd/subcmds/auth"
	"github.com/formancehq/fctl/cmd/subcmds/ledger"
	"github.com/formancehq/fctl/cmd/subcmds/organizations"
	"github.com/formancehq/fctl/cmd/subcmds/payments"
	"github.com/formancehq/fctl/cmd/subcmds/profiles"
	"github.com/formancehq/fctl/cmd/subcmds/stack"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRootCommand() *cobra.Command {
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return cmdbuilder.NewCommand("fctl",
		cmdbuilder.WithShortDescription("Formance Control CLI"),
		cmdbuilder.WithSilenceUsage(),
		cmdbuilder.WithPersistentPreRunE(func(cmd *cobra.Command, args []string) (err error) {
			viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
			viper.AutomaticEnv()
			return viper.BindPFlags(cmd.Flags())
		}),
		cmdbuilder.WithChildCommands(
			ledger.NewLedgerCommand(),
			payments.NewPaymentsCommand(),
			profiles.NewProfilesCommand(),
			stack.NewSandboxCommand(),
			subcmds.NewUICommand(),
			subcmds.NewVersionCommand(),
			subcmds.NewLoginCommand(),
			auth.NewAuthCommand(),
			organizations.NewOrganizationsCommand(),
			subcmds.NewWhoamiCommand(),
		),
		cmdbuilder.WithPersistentStringPFlag(config.ProfileFlag, "p", "", "config profile to use"),
		cmdbuilder.WithPersistentStringPFlag(config.FileFlag, "c", fmt.Sprintf("%s/.formance/fctl.config", homedir), "Debug mode"),
		cmdbuilder.WithPersistentBoolPFlag(config.DebugFlag, "d", false, "Debug mode"),
		cmdbuilder.WithPersistentBoolFlag(config.InsecureTlsFlag, false, "Insecure TLS"),
	)
}

func Execute() {
	_ = newRootCommand().Execute()
}
