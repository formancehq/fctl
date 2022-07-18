package cloud

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewDeployment() *cobra.Command {
	command := &cobra.Command{
		Use: "deployment",
	}

	command.AddCommand(NewDeploymentList())
	command.AddCommand(NewDeploymentUpgrade())
	command.AddCommand(NewDeploymentCreate())
	command.AddCommand(NewDeploymentTeardown())

	return command
}

func NewDeploymentList() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("list of deployments")
		},
	}
}

func NewDeploymentUpgrade() *cobra.Command {
	return &cobra.Command{
		Use: "upgrade",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
}

func NewDeploymentCreate() *cobra.Command {
	return &cobra.Command{
		Use: "create",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
}

func NewDeploymentTeardown() *cobra.Command {
	return &cobra.Command{
		Use: "teardown",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
}
