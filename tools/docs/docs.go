package main

import (
	"encoding/json"
	"fmt"
	"os"

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

func writeDocs(parentCommandName string, command *cobra.Command, file *os.File) {
	flags := flagSetToFlags(command.Flags())
	globalFlags := flagSetToFlags(command.Root().PersistentFlags())

	doc := DocCommand{
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
		Flags:       flags,
		GlobalFlags: globalFlags,
	}

	docAsJson, err := json.Marshal(&doc)
	if err != nil {
		panic(err) // Handle error appropriately in production code
	}

	if _, err := file.Write(docAsJson); err != nil {
		panic(err)
	}
}

func visitSubCommands(command *cobra.Command, file *os.File) {
	var (
		parentCommand *cobra.Command
		parentName    string
	)

	if command != command.Root() {
		parentCommand = command.Parent()
		parentName = parentCommand.Name()
	}

label:
	if command.HasParent() && command.Parent() != command.Root() {
		parentCommand = parentCommand.Parent()
		if parentCommand != nil {
			parentName = parentCommand.Name() + " " + parentName
			goto label
		}
	}

	writeDocs(parentName, command, file)
	for _, subCommand := range command.Commands() {
		writeDocs(parentName, subCommand, file)
		visitSubCommands(subCommand, file)
	}
}

//go:generate rm -rf docs
//go:generate go run ./
func main() {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		if err := os.WriteFile(file, []byte("[]"), 0644); err != nil {
			panic(err)
		}
	} else {
		os.Remove(file)
	}

	file, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	visitSubCommands(cmd.NewRootCommand(), file)
}
