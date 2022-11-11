package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	_ "github.com/athul/shelby/mods"
	goprompt "github.com/c-bata/go-prompt"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/mattn/go-shellwords"
	_ "github.com/mattn/go-shellwords"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newOverrideCommand() *cobra.Command {
	subCommand := NewRootCommand()
	subCommand.AddCommand(cmdbuilder.NewCommand("exit",
		cmdbuilder.WithAliases("q", "quit"),
		cmdbuilder.WithShortDescription("Exit prompt"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			os.Exit(0)
			return nil
		}),
	))
	return subCommand
}

func startPrompt(ctx context.Context, prompt string, opts ...goprompt.Option) string {
	return goprompt.Input(prompt, func(d goprompt.Document) []goprompt.Suggest {

		completionsArgs := make([]string, 0)
		if d.Text == "" {
			completionsArgs = append(completionsArgs, "")
		} else {
			parse, err := shellwords.Parse(d.Text)
			if err != nil {
				panic(err)
			}

			completionsArgs = append(completionsArgs, parse...)
			if strings.HasSuffix(d.Text, " ") {
				completionsArgs = append(completionsArgs, "")
			}
		}

		subCommandOut := bytes.NewBufferString("")

		subCommand := newOverrideCommand()
		subCommand.SetArgs(append([]string{
			cobra.ShellCompRequestCmd,
		}, completionsArgs...))
		subCommand.SetOut(subCommandOut)
		subCommand.SetErr(io.Discard)
		if err := subCommand.ExecuteContext(ctx); err != nil {
			panic(err)
		}

		suggestions := make([]goprompt.Suggest, 0)
		scanner := bufio.NewScanner(subCommandOut)

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, ":") || strings.HasPrefix(line, "[Debug]") {
				break
			}
			parts := strings.SplitN(line, "\t", 2)
			text := parts[0]
			description := ""
			if len(parts) == 2 {
				description = parts[1]
			}
			suggestions = append(suggestions, goprompt.Suggest{
				Text:        text,
				Description: description,
			})
		}

		return goprompt.FilterHasPrefix(suggestions, d.GetWordBeforeCursor(), true)
	}, opts...)
}

func executeCommand(cmd *cobra.Command, t string) error {
	parse, err := shellwords.Parse(t)
	if err != nil {
		panic(err)
	}

	subCommand := newOverrideCommand()
	subCommand.SetArgs(parse)
	subCommand.SetOut(cmd.OutOrStdout())
	subCommand.SetErr(cmd.ErrOrStderr())
	subCommand.SilenceErrors = true
	subCommand.SilenceUsage = true
	return subCommand.ExecuteContext(cmd.Context())
}

func NewPromptCommand() *cobra.Command {
	return cmdbuilder.NewCommand("prompt",
		cmdbuilder.WithShortDescription("Start a prompt"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			promptColor := goprompt.Blue
			history := make([]string, 0)

			for {
				if err := viper.BindPFlags(cmd.Flags()); err != nil {
					panic(err)
				}

				cfg, err := config.Get()
				if err != nil {
					return err
				}

				prompt := "> "
				organizationID, err := cmdbuilder.RetrieveOrganizationIDFromFlagOrProfile(cfg)
				if err != nil && !errors.Is(err, cmdbuilder.ErrOrganizationNotSpecified) {
					return err
				}
				if organizationID != "" {
					prompt = fmt.Sprintf("%s %s", organizationID, prompt)
				}
				prompt = fmt.Sprintf("%s / %s", config.GetCurrentProfileName(cfg), prompt)

				switch t := startPrompt(cmd.Context(), prompt,
					goprompt.OptionPrefixTextColor(promptColor),
					goprompt.OptionHistory(history),
					goprompt.OptionCompletionOnDown()); t {
				case "":
					promptColor = goprompt.Blue
				default:
					err := executeCommand(cmd, t)
					if err != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), err)
						promptColor = goprompt.Red
					} else {
						promptColor = goprompt.Blue
					}
					history = append(history, t)
				}
			}
		}),
	)
}
