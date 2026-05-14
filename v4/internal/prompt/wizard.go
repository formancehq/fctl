package prompt

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
)

type Choice struct {
	Title string
	Value string
}

type Wizard struct {
	in          io.Reader
	out         io.Writer
	interactive bool
}

func NewWizard(in io.Reader, out io.Writer) Wizard {
	return Wizard{
		in:          in,
		out:         out,
		interactive: isTerminal(in) && isTerminal(out),
	}
}

func (w Wizard) Available() bool {
	return w.interactive
}

func (w Wizard) Select(title string, choices []Choice) (string, error) {
	if !w.Available() {
		return "", errors.New("interactive prompt is not available")
	}
	if len(choices) == 0 {
		return "", errors.New("select prompt requires at least one choice")
	}

	value := choices[0].Value
	options := make([]huh.Option[string], len(choices))
	for i, choice := range choices {
		options[i] = huh.NewOption(choice.Title, choice.Value)
	}

	err := w.run(
		huh.NewSelect[string]().
			Title(title).
			Options(options...).
			Value(&value).
			Height(len(options) + 1),
	)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (w Wizard) Input(title string, placeholder string, secret bool) (string, error) {
	if !w.Available() {
		return "", errors.New("interactive prompt is not available")
	}

	value := ""
	input := huh.NewInput().
		Title(title).
		Value(&value)
	if placeholder != "" {
		input.Placeholder(placeholder)
	}
	if secret {
		input.EchoMode(huh.EchoModePassword)
	}

	if err := w.run(input); err != nil {
		return "", err
	}
	return value, nil
}

func (w Wizard) run(field huh.Field) error {
	form := huh.NewForm(huh.NewGroup(field)).
		WithInput(w.in).
		WithOutput(w.out).
		WithTheme(huh.ThemeBase()).
		WithShowHelp(false)
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return fmt.Errorf("prompt cancelled")
		}
		return err
	}
	return nil
}

func isTerminal(value any) bool {
	file, ok := value.(*os.File)
	return ok && isatty.IsTerminal(file.Fd())
}
