package cmdutils

import (
	"strings"

	"github.com/spf13/cobra"
)

func BindFlags(cmd *cobra.Command) error {
	Viper(cmd.Context()).SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	Viper(cmd.Context()).AutomaticEnv()
	return Viper(cmd.Context()).BindPFlags(cmd.Flags())
}
