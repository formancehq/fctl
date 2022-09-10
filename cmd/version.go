package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "develop"
	BuildDate = "-"
	Commit    = "-"
)

func PrintVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("Version: %s \n", Version)
	fmt.Printf("Date: %s \n", BuildDate)
	fmt.Printf("Commit: %s \n", Commit)
}

var versionCommand = &cobra.Command{
	Use:   "version",
	Short: "Get version",
	Run:   PrintVersion,
}

func init() {
	rootCommand.AddCommand(versionCommand)
}
