package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/formancehq/fctl/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	file = "inventory.json"
)

type Flag struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Deprecated   string `json:"deprecated,omitempty"`
	Usage        string `json:"usage"`
	DefaultValue string `json:"default_value,omitempty"`
	Type         string `json:"type,omitempty"`
}

type DocCommand struct {
	Name        string   `json:"name"`
	Usage       string   `json:"usage"`
	Description string   `json:"description"`
	Deprecated  string   `json:"deprecated,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`

	Flags       []Flag `json:"flags,omitempty"`
	GlobalFlags []Flag `json:"global_flags,omitempty"`

	SubCommands []DocCommand `json:"subcommands,omitempty"`
}

func flagSetToFlags(flagSet *pflag.FlagSet) []Flag {
	flags := make([]Flag, 0)

	flagSet.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}

		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		if strings.Contains(f.DefValue, userHomeDir) {
			f.DefValue = strings.ReplaceAll(f.DefValue, userHomeDir, "~")
		}

		flags = append(flags, Flag{
			Name:         f.Name,
			Description:  f.Usage,
			Deprecated:   f.Deprecated,
			DefaultValue: f.DefValue,
			Usage:        f.Usage,
			Type:         f.Value.Type(),
		})
	})

	return flags
}

func cobraCommandAsDocCommand(parentCommandName string, command *cobra.Command) DocCommand {
	return DocCommand{
		Name: func() string {
			if parentCommandName != "" {
				return fmt.Sprintf("%s %s", parentCommandName, command.Name())
			}
			return command.Name()
		}(),
		Usage:       command.Use,
		Description: command.Short,
		Deprecated:  command.Deprecated,
		Aliases:     command.Aliases,
		Flags:       flagSetToFlags(command.Flags()),
		GlobalFlags: flagSetToFlags(command.Root().PersistentFlags()),
	}

}

func getFullCommandPath(cmd *cobra.Command) string {
	if cmd == nil || cmd == cmd.Root() {
		return ""
	}
	parent := getFullCommandPath(cmd.Parent())
	if parent == "" {
		return cmd.Name()
	}
	return parent + " " + cmd.Name()
}

func visitSubCommands(command *cobra.Command) []DocCommand {
	parentName := getFullCommandPath(command.Parent())
	docs := []DocCommand{cobraCommandAsDocCommand(parentName, command)}
	for _, subCommand := range command.Commands() {
		docs = append(docs, visitSubCommands(subCommand)...)
	}
	return docs
}

//go:generate rm -rf docs
//go:generate go run ./
func main() {
	_, err := os.Stat(file)
	if err == nil {
		if err := os.Remove(file); err != nil {
			panic(err)
		}
	}

	commands := visitSubCommands(cmd.NewRootCommand())
	file, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	b, err := json.Marshal(commands)
	if err != nil {
		panic(err)
	}
	if _, err := file.Write(b); err != nil {
		panic(err)
	}
}
