package cloud

import "github.com/spf13/cobra"

func NewApiKey() *cobra.Command {
	command := &cobra.Command{
		Use: "api-key",
	}

	command.AddCommand(NewApiKeyAdd())
	command.AddCommand(NewApiKeyList())
	command.AddCommand(NewApiKeyDelete())

	return command
}

func NewApiKeyAdd() *cobra.Command {
	return &cobra.Command{
		Use: "add",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
}

func NewApiKeyList() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
}

func NewApiKeyDelete() *cobra.Command {
	return &cobra.Command{
		Use: "delete",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
}
