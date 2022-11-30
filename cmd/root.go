package cmd

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/formancehq/fctl/cmd/auth"
	"github.com/formancehq/fctl/cmd/cloud"
	"github.com/formancehq/fctl/cmd/internal"
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

	return internal.NewCommand("fctl",
		internal.WithSilenceError(),
		internal.WithShortDescription("Formance Control CLI"),
		internal.WithSilenceUsage(),
		internal.WithChildCommands(
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
		internal.WithPersistentStringPFlag(internal.ProfileFlag, "p", "", "config profile to use"),
		internal.WithPersistentStringPFlag(internal.FileFlag, "c", fmt.Sprintf("%s/.formance/fctl.config", homedir), "Debug mode"),
		internal.WithPersistentBoolPFlag(internal.DebugFlag, "d", false, "Debug mode"),
		internal.WithPersistentBoolFlag(internal.InsecureTlsFlag, false, "Insecure TLS"),
	)
}

func Execute() {
	defer func() {
		if e := recover(); e != nil {
			internal.Error(os.Stderr, "%s", e)
			debug.PrintStack()
		}
	}()
	err := NewRootCommand().Execute()
	if err != nil {
		internal.Error(os.Stderr, err.Error())
	}
}
