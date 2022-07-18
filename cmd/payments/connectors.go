package payments

import "github.com/spf13/cobra"

func NewConnectors() *cobra.Command {
	command := &cobra.Command{
		Use: "connectors",
	}

	command.AddCommand(NewEnable())
	command.AddCommand(NewDisable())
	command.AddCommand(NewDelete())

	return command
}

func NewEnable() *cobra.Command {
	return &cobra.Command{
		Use: "enable",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
}

func NewDisable() *cobra.Command {
	return &cobra.Command{
		Use: "disable",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
}

func NewDelete() *cobra.Command {
	return &cobra.Command{
		Use: "delete",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
}
