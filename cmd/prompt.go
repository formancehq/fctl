package cmd

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	goprompt "github.com/c-bata/go-prompt"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/iancoleman/strcase"
	"github.com/mattn/go-shellwords"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	ErrMalformedCommand = errors.New("malformed command")
)

type prompt struct {
	promptColor   goprompt.Color
	history       []string
	userEmail     string
	actualProfile string
}

func (p *prompt) completionsFromCommand(subCommand *cobra.Command, completionsArgs []string, d goprompt.Document) []goprompt.Suggest {

	defer func() {
		// The autocompletion library sometimes panic
		// As it is not critical, we just catch the error and display it only when debug enabled
		if err := recover(); err != nil {
			isDebug, _ := subCommand.Flags().GetBool(fctl.DebugFlag)
			if isDebug {
				fmt.Println(err)
				debug.PrintStack()
			}
		}
	}()

	_, completions, _, err := subCommand.GetCompletions(completionsArgs)
	if err != nil {
		return []goprompt.Suggest{}
	}
	return goprompt.FilterHasPrefix(fctl.Map(completions, func(src string) goprompt.Suggest {
		parts := strings.SplitN(src, "\t", 2)
		description := ""
		if len(parts) > 1 {
			description = parts[1]
		}
		return goprompt.Suggest{
			Text:        parts[0],
			Description: description,
		}
	}), d.GetWordBeforeCursor(), true)
}

func (p *prompt) completions(cmd *cobra.Command, d goprompt.Document) ([]goprompt.Suggest, error) {
	suggestions := make([]goprompt.Suggest, 0)
	switch {
	case strings.HasPrefix(d.Text, ":set "+fctl.ProfileFlag):
		profiles, err := fctl.ListProfiles(cmd)
		if err != nil {
			return nil, err
		}

		sort.Strings(profiles)
		for _, p := range profiles {
			suggestions = append(suggestions, goprompt.Suggest{
				Text:        p,
				Description: "Select profile",
			})
		}
	case strings.HasPrefix(d.Text, ":set "+fctl.DebugFlag) || strings.HasPrefix(d.Text, ":set "+fctl.InsecureTlsFlag):
		suggestions = append(suggestions, goprompt.Suggest{
			Text: "true",
		}, goprompt.Suggest{
			Text: "false",
		})
	case strings.HasPrefix(d.Text, ":set"):
		suggestions = append(suggestions, goprompt.Suggest{
			Text:        fctl.ProfileFlag,
			Description: "Select profile",
		}, goprompt.Suggest{
			Text:        fctl.DebugFlag,
			Description: "Set debug",
		}, goprompt.Suggest{
			Text:        fctl.InsecureTlsFlag,
			Description: "Set insecure TLS",
		})
	default:
		suggestions = append(suggestions, goprompt.Suggest{
			Text:        ":set",
			Description: "Set config",
		}, goprompt.Suggest{
			Text:        ":exit",
			Description: "Exit terminal",
		})
	}

	return goprompt.FilterHasPrefix(suggestions, d.GetWordBeforeCursor(), true), nil
}

func (p *prompt) startPrompt(cmd *cobra.Command, prompt string, opts ...goprompt.Option) (string, error) {
	return goprompt.Input(prompt, func(d goprompt.Document) []goprompt.Suggest {
		subCommand := NewRootCommand()
		subCommand.SetContext(cmd.Context())

		switch {
		case d.Text == "":
			return p.completionsFromCommand(subCommand, []string{""}, d)
		case strings.HasPrefix(d.Text, ":"):
			completions, err := p.completions(cmd, d)
			if err != nil {
				return []goprompt.Suggest{}
			}

			return completions
		default:
			parse, err := shellwords.Parse(d.Text)
			if err != nil {
				return []goprompt.Suggest{}
			}

			if strings.HasSuffix(d.Text, " ") || len(parse) == 0 {
				parse = append(parse, "")
			}
			return p.completionsFromCommand(subCommand, parse, d)
		}
	}, opts...), nil
}

func (p *prompt) executeCommand(cmd *cobra.Command, t string) error {
	parse, err := shellwords.Parse(t)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrUnsupported, err)
	}

	subCommand := NewRootCommand()
	subCommand.SetArgs(parse)
	subCommand.SetOut(cmd.OutOrStdout())
	subCommand.SetErr(cmd.ErrOrStderr())
	subCommand.SilenceErrors = true
	subCommand.SilenceUsage = true
	return subCommand.ExecuteContext(cmd.Context())
}

func (p *prompt) executePromptCommand(cmd *cobra.Command, t string) error {
	switch {
	case strings.TrimRight(t, " ") == ":exit":
		os.Exit(0)
	case strings.HasPrefix(t, ":set "):
		v := strings.TrimPrefix(t, ":set ")
		v = strings.TrimLeft(v, " ")
		v = strings.TrimRight(v, " ")
		parts := strings.SplitN(v, " ", 2)
		if len(parts) != 2 {
			return ErrMalformedCommand
		} else {
			if v := parts[0]; v != fctl.ProfileFlag && v != fctl.DebugFlag && v != fctl.InsecureTlsFlag {
				return fmt.Errorf("unknown configuration: %s", v)
			}
			_ = cmd.Flags().Set(parts[0], parts[1])
			os.Setenv(strcase.ToScreamingSnake(parts[0]), parts[1])
			pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Set %s=%s", parts[0], parts[1])
		}
	default:
		return ErrMalformedCommand
	}
	return nil
}

func (p *prompt) refreshUserEmail(cmd *cobra.Command, cfg fctl.Config) error {
	profile, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, cfg)
	if err != nil {
		return err
	}
	if !profile.IsConnected() {
		p.userEmail = ""
		return nil
	}

	userInfo, err := fctl.UserInfo(cmd, relyingParty, profile.RootTokens.Access)
	if err != nil {
		p.userEmail = ""
		return nil
	}
	p.userEmail = userInfo.Email
	return nil
}

func (p *prompt) displayHeader(cmd *cobra.Command, cfg *fctl.Config) error {
	header := fctl.GetCurrentProfileName(cmd, *cfg)
	if p.userEmail != "" {
		currentProfile, err := fctl.LoadCurrentProfile(cmd, *cfg)
		if err != nil {
			return err
		}
		header += " / " + p.userEmail
		if organizationID := currentProfile.GetDefaultOrganization(); organizationID != "" {
			header += " / " + organizationID
		}

		if stackID := currentProfile.GetDefaultStack(); stackID != "" {
			header += " / " + stackID
		}
	}
	header += " #"
	fctl.BasicTextCyan.WithWriter(cmd.OutOrStdout()).Printfln(header)
	return nil
}

func (p *prompt) nextCommand(cmd *cobra.Command) error {

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return err
	}

	currentProfileName := fctl.GetCurrentProfileName(cmd, *cfg)
	if currentProfileName != p.actualProfile || p.userEmail == "" {
		if err := p.refreshUserEmail(cmd, *cfg); err != nil {
			return err
		}
		p.actualProfile = currentProfileName
	}

	if err := p.displayHeader(cmd, cfg); err != nil {
		return err
	}

	prompt, err := p.startPrompt(cmd, " > ",
		goprompt.OptionPrefixTextColor(p.promptColor),
		goprompt.OptionHistory(p.history),
		goprompt.OptionCompletionOnDown())
	if err != nil {
		return err
	}
	switch prompt {
	case "":
		p.promptColor = goprompt.Blue
	default:
		var err error
		if strings.HasPrefix(prompt, ":") {
			err = p.executePromptCommand(cmd, prompt)
		} else {
			err = p.executeCommand(cmd, prompt)
		}
		if err != nil {
			pterm.Error.WithWriter(cmd.OutOrStderr()).Printfln("%s", err)
			p.promptColor = goprompt.Red
		} else {
			p.promptColor = goprompt.Blue
		}
		p.history = append(p.history, prompt)
	}

	return nil
}

func (p *prompt) run(cmd *cobra.Command) error {
	for {
		if err := p.nextCommand(cmd); err != nil {
			return err
		}
	}
}

func NewPromptCommand() *cobra.Command {
	return fctl.NewCommand("prompt",
		fctl.WithShortDescription("Start a prompt"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			return (&prompt{
				promptColor: goprompt.Blue,
				history:     make([]string, 0),
			}).run(cmd)
		}),
	)
}
