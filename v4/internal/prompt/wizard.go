package prompt

import (
	"errors"
	"io"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"

	v4render "github.com/formancehq/fctl/v4/internal/render"
)

type Choice struct {
	Title string
	Value string
}

var ErrCancelled = errors.New("prompt cancelled")

type Wizard struct {
	in          io.Reader
	out         io.Writer
	interactive bool
	color       bool
}

func NewWizard(in io.Reader, out io.Writer) Wizard {
	return NewWizardWithColor(in, out, true)
}

func NewWizardWithColor(in io.Reader, out io.Writer, color bool) Wizard {
	return Wizard{
		in:          in,
		out:         out,
		interactive: isTerminal(in) && isTerminal(out),
		color:       color,
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
		WithTheme(w.theme()).
		WithShowHelp(false)
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return ErrCancelled
		}
		return err
	}
	return nil
}

func (w Wizard) theme() *huh.Theme {
	if !w.color {
		return huh.ThemeBase()
	}

	theme := huh.ThemeBase()
	text := v4render.FormancePalette.Text
	muted := v4render.FormancePalette.Muted
	accent := v4render.FormancePalette.Mint
	warning := v4render.FormancePalette.Gold
	errorColor := v4render.FormancePalette.Error

	theme.Focused.Base = theme.Focused.Base.
		BorderForeground(accent).
		PaddingLeft(1)
	theme.Focused.Card = theme.Focused.Base
	theme.Focused.Title = theme.Focused.Title.
		Foreground(accent).
		Bold(true)
	theme.Focused.NoteTitle = theme.Focused.NoteTitle.
		Foreground(accent).
		Bold(true)
	theme.Focused.Description = theme.Focused.Description.Foreground(muted)
	theme.Focused.ErrorIndicator = theme.Focused.ErrorIndicator.Foreground(errorColor)
	theme.Focused.ErrorMessage = theme.Focused.ErrorMessage.Foreground(errorColor)
	theme.Focused.SelectSelector = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		SetString("› ")
	theme.Focused.Option = theme.Focused.Option.Foreground(text)
	theme.Focused.NextIndicator = theme.Focused.NextIndicator.Foreground(warning)
	theme.Focused.PrevIndicator = theme.Focused.PrevIndicator.Foreground(warning)
	theme.Focused.MultiSelectSelector = theme.Focused.MultiSelectSelector.Foreground(accent)
	theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(accent).Bold(true)
	theme.Focused.SelectedPrefix = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		SetString("✓ ")
	theme.Focused.UnselectedOption = theme.Focused.UnselectedOption.Foreground(text)
	theme.Focused.UnselectedPrefix = lipgloss.NewStyle().
		Foreground(muted).
		SetString("• ")
	theme.Focused.TextInput.Cursor = theme.Focused.TextInput.Cursor.Foreground(accent)
	theme.Focused.TextInput.CursorText = theme.Focused.TextInput.CursorText.Foreground(text)
	theme.Focused.TextInput.Placeholder = theme.Focused.TextInput.Placeholder.Foreground(muted)
	theme.Focused.TextInput.Prompt = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		SetString("› ")
	theme.Focused.FocusedButton = theme.Focused.FocusedButton.
		Foreground(lipgloss.Color("#081A16")).
		Background(accent).
		Bold(true)
	theme.Focused.BlurredButton = theme.Focused.BlurredButton.
		Foreground(text).
		Background(v4render.FormancePalette.Slate)
	theme.Focused.Next = theme.Focused.FocusedButton

	theme.Blurred = theme.Focused
	theme.Blurred.Base = theme.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	theme.Blurred.Card = theme.Blurred.Base
	theme.Blurred.Title = theme.Blurred.Title.Foreground(muted).Bold(false)
	theme.Blurred.TextInput.Prompt = theme.Blurred.TextInput.Prompt.Foreground(muted).Bold(false)
	theme.Blurred.SelectSelector = lipgloss.NewStyle().SetString("  ")
	theme.Blurred.NextIndicator = lipgloss.NewStyle()
	theme.Blurred.PrevIndicator = lipgloss.NewStyle()

	theme.Group.Title = theme.Focused.Title
	theme.Group.Description = theme.Focused.Description
	return theme
}

func isTerminal(value any) bool {
	file, ok := value.(*os.File)
	return ok && isatty.IsTerminal(file.Fd())
}
