package render

import "github.com/charmbracelet/lipgloss"

type Palette struct {
	Emerald lipgloss.Color
	Slate   lipgloss.Color
	Gold    lipgloss.Color
	Lilac   lipgloss.Color
	Cobalt  lipgloss.Color
	Mint    lipgloss.Color

	Text    lipgloss.Color
	Muted   lipgloss.Color
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
}

var FormancePalette = Palette{
	Emerald: lipgloss.Color("#007A5E"),
	Slate:   lipgloss.Color("#416266"),
	Gold:    lipgloss.Color("#D6A84F"),
	Lilac:   lipgloss.Color("#B9A7FF"),
	Cobalt:  lipgloss.Color("#3366FF"),
	Mint:    lipgloss.Color("#7FE7C4"),

	Text:    lipgloss.Color("#EAF2EF"),
	Muted:   lipgloss.Color("#7A8A8D"),
	Success: lipgloss.Color("#7FE7C4"),
	Warning: lipgloss.Color("#D6A84F"),
	Error:   lipgloss.Color("#FF6B6B"),
}

var Styles = struct {
	TableHeader lipgloss.Style
	TableCell   lipgloss.Style
	Muted       lipgloss.Style
	Success     lipgloss.Style
	Warning     lipgloss.Style
	Error       lipgloss.Style
}{
	TableHeader: lipgloss.NewStyle().Foreground(FormancePalette.Mint).PaddingRight(1),
	TableCell:   lipgloss.NewStyle().PaddingRight(1),
	Muted:       lipgloss.NewStyle().Foreground(FormancePalette.Muted),
	Success:     lipgloss.NewStyle().Foreground(FormancePalette.Success),
	Warning:     lipgloss.NewStyle().Foreground(FormancePalette.Warning),
	Error:       lipgloss.NewStyle().Foreground(FormancePalette.Error),
}
