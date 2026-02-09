package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Theme colors
	Primary   = lipgloss.Color("#7C3AED")
	Secondary = lipgloss.Color("#A78BFA")
	Success   = lipgloss.Color("#10B981")
	Warning   = lipgloss.Color("#F59E0B")
	Error     = lipgloss.Color("#EF4444")
	Muted     = lipgloss.Color("#6B7280")
	Text      = lipgloss.Color("#E5E7EB")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Secondary)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(0, 1)

	BannerStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)
)

// Select displays an interactive selection prompt with fuzzy filtering.
func Select(label string, options []string) (string, error) {
	var selected string

	huhOptions := make([]huh.Option[string], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(opt, opt)
	}

	err := huh.NewSelect[string]().
		Title(label).
		Options(huhOptions...).
		Value(&selected).
		Height(20).
		Run()

	if err != nil {
		return "", err
	}

	return selected, nil
}

// SelectWithFilter displays a filterable selection prompt.
func SelectWithFilter(label string, options []string) (string, error) {
	return Select(label, options)
}

// Confirm displays a yes/no prompt.
func Confirm(label string) (bool, error) {
	var confirmed bool

	err := huh.NewConfirm().
		Title(label).
		Value(&confirmed).
		Run()

	if err != nil {
		return false, err
	}

	return confirmed, nil
}

// Input displays a text input prompt.
func Input(label, placeholder string) (string, error) {
	var value string

	err := huh.NewInput().
		Title(label).
		Placeholder(placeholder).
		Value(&value).
		Run()

	if err != nil {
		return "", err
	}

	return value, nil
}

// PrintBanner displays the application banner.
func PrintBanner(version string) {
	banner := `
     _                _ _
  __| | _____   _____| (_)
 / _` + "`" + ` |/ _ \ \ / / __| | |
| (_| |  __/\ V / (__| | |
 \__,_|\___| \_/ \___|_|_|`

	fmt.Println(BannerStyle.Render(banner))
	fmt.Println()
	fmt.Println(MutedStyle.Render(fmt.Sprintf("  v%s — ECS Container Access Tool", version)))
	fmt.Println(MutedStyle.Render("  Maintainer: Michael COULLERET <hello@0uf.eu>"))
	fmt.Println()
}

// PrintStep displays a styled step message.
func PrintStep(icon, message string) {
	fmt.Printf("%s %s\n", TitleStyle.Render(icon), message)
}

// PrintSuccess displays a success message.
func PrintSuccess(message string) {
	fmt.Println(SuccessStyle.Render("✓ " + message))
}

// PrintWarning displays a warning message.
func PrintWarning(message string) {
	fmt.Println(WarningStyle.Render("! " + message))
}

// PrintError displays an error message.
func PrintError(message string) {
	fmt.Println(ErrorStyle.Render("✗ " + message))
}

// PrintInfo displays an info box.
func PrintInfo(title, content string) {
	header := TitleStyle.Render(title)
	body := BoxStyle.Render(content)
	fmt.Printf("%s\n%s\n", header, body)
}
