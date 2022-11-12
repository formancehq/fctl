package cmd

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/formancehq/fctl/cmd/auth"
	"github.com/formancehq/fctl/cmd/cloud"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/ledger"
	"github.com/formancehq/fctl/cmd/payments"
	"github.com/formancehq/fctl/cmd/profiles"
	"github.com/formancehq/fctl/cmd/search"
	"github.com/formancehq/fctl/cmd/stack"
	"github.com/formancehq/fctl/cmd/webhooks"
	"github.com/spf13/cobra"
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
			search.NewCommand(),
			webhooks.NewCommand(),
		),
		cmdbuilder.WithPersistentStringPFlag(config.ProfileFlag, "p", "", "config profile to use"),
		cmdbuilder.WithPersistentStringPFlag(config.FileFlag, "c", fmt.Sprintf("%s/.formance/fctl.config", homedir), "Debug mode"),
		cmdbuilder.WithPersistentBoolPFlag(config.DebugFlag, "d", false, "Debug mode"),
		cmdbuilder.WithPersistentBoolFlag(config.InsecureTlsFlag, false, "Insecure TLS"),
	)
}

func Execute() {
	defer func() {
		if e := recover(); e != nil {
			cmdbuilder.Error(os.Stderr, "%s", e)
			debug.PrintStack()
		}
	}()
	err := NewRootCommand().Execute()
	if err != nil {
		cmdbuilder.Error(os.Stderr, err.Error())
	}
}
