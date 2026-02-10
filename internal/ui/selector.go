package ui

import (
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	// ErrUserAbort is returned when the user cancels a prompt (ESC / Ctrl+C).
	ErrUserAbort = errors.New("user abort")

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

// devTheme returns a custom huh theme with wheel picker focus effect.
func devTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Remove the left border for a cleaner look
	t.Focused.Base = lipgloss.NewStyle().PaddingLeft(1)
	t.Focused.Card = t.Focused.Base

	// Title
	t.Focused.Title = lipgloss.NewStyle().Foreground(Primary).Bold(true)

	// Select: wheel picker effect — bright cursor, muted rest
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(Primary).SetString("› ")
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(Primary).Bold(true)
	t.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(Muted)
	t.Focused.Option = lipgloss.NewStyle().Foreground(Muted)

	// Scroll indicators
	t.Focused.NextIndicator = lipgloss.NewStyle().Foreground(Secondary).SetString("  ↓")
	t.Focused.PrevIndicator = lipgloss.NewStyle().Foreground(Secondary).SetString("  ↑")

	// Filter input
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(Primary)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(Muted)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(Secondary).SetString("/ ")
	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(Text)

	// Buttons
	t.Focused.FocusedButton = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(Primary).Padding(0, 1)
	t.Focused.BlurredButton = lipgloss.NewStyle().Foreground(Muted).Background(lipgloss.Color("#333")).Padding(0, 1)

	// Blurred = same but hidden border
	t.Blurred = t.Focused
	t.Blurred.Base = lipgloss.NewStyle().PaddingLeft(1)
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	return t
}

func selectHeight(count int) int {
	h := count + 2
	if h > 15 {
		h = 15
	}
	if h < 5 {
		h = 5
	}
	return h
}

// SelectOption represents a display/value pair for select prompts.
type SelectOption struct {
	Display string
	Value   string
}

// Select displays an interactive selection prompt with type-to-filter.
func Select(label string, options []string) (string, error) {
	var selected string

	huhOptions := make([]huh.Option[string], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(opt, opt)
	}

	sel := huh.NewSelect[string]().
		Title(label).
		Options(huhOptions...).
		Value(&selected).
		Height(selectHeight(len(options))).
		Filtering(true)

	err := huh.NewForm(huh.NewGroup(sel)).WithTheme(devTheme()).Run()
	if err != nil {
		return "", ErrUserAbort
	}

	return selected, nil
}

// SelectWithOptions displays a selection prompt with separate display/value pairs.
func SelectWithOptions(label string, options []SelectOption) (string, error) {
	var selected string

	huhOptions := make([]huh.Option[string], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(opt.Display, opt.Value)
	}

	sel := huh.NewSelect[string]().
		Title(label).
		Options(huhOptions...).
		Value(&selected).
		Height(selectHeight(len(options))).
		Filtering(true)

	err := huh.NewForm(huh.NewGroup(sel)).WithTheme(devTheme()).Run()
	if err != nil {
		return "", ErrUserAbort
	}

	return selected, nil
}

// Confirm displays a yes/no prompt.
func Confirm(label string) (bool, error) {
	var confirmed bool

	c := huh.NewConfirm().
		Title(label).
		Value(&confirmed)

	err := huh.NewForm(huh.NewGroup(c)).WithTheme(devTheme()).Run()
	if err != nil {
		return false, ErrUserAbort
	}

	return confirmed, nil
}

// Input displays a text input prompt.
func Input(label, placeholder string) (string, error) {
	var value string

	i := huh.NewInput().
		Title(label).
		Placeholder(placeholder).
		Value(&value)

	err := huh.NewForm(huh.NewGroup(i)).WithTheme(devTheme()).Run()
	if err != nil {
		return "", ErrUserAbort
	}

	return value, nil
}

const bannerArt = `
     _                _ _
  __| | _____   _____| (_)
 / _` + "`" + ` |/ _ \ \ / / __| | |
| (_| |  __/\ V / (__| | |
 \__,_|\___| \_/ \___|_|_|`

// PrintBanner displays the application banner.
func PrintBanner(version string) {
	fmt.Println(BannerStyle.Render(bannerArt))
	fmt.Println()
	fmt.Println(MutedStyle.Render(fmt.Sprintf("  v%s — Focus on coding, not on tooling.", version)))
	fmt.Println(MutedStyle.Render("  Michael COULLERET <hello@0uf.eu>"))
	fmt.Println(MutedStyle.Render("  Contributors: Thomas Talbot"))
	fmt.Println()
}

// UpdateResult holds the result of an update check.
type UpdateResult struct {
	Latest    string
	HasUpdate bool
}

// PrintBannerWithUpdateCheck displays the banner with an inline update check.
func PrintBannerWithUpdateCheck(version string, checkFn func() (string, bool, error)) *UpdateResult {
	fmt.Println(BannerStyle.Render(bannerArt))
	fmt.Println()

	versionText := MutedStyle.Render(fmt.Sprintf("  v%s — Focus on coding, not on tooling.", version))

	var result *UpdateResult

	if checkFn != nil {
		// Show discreet loading indicator on the version line
		fmt.Printf("%s  %s", versionText, MutedStyle.Render("⟳ checking..."))

		// Run check with timeout
		type checkResult struct {
			latest    string
			hasUpdate bool
			err       error
		}
		ch := make(chan checkResult, 1)
		go func() {
			l, h, e := checkFn()
			ch <- checkResult{l, h, e}
		}()

		var cr checkResult
		select {
		case cr = <-ch:
		case <-time.After(3 * time.Second):
			cr = checkResult{err: fmt.Errorf("timeout")}
		}

		// Clear the line and reprint with status
		fmt.Print("\r\033[K")

		if cr.err != nil {
			fmt.Println(versionText)
		} else if !cr.hasUpdate {
			fmt.Printf("%s  %s\n", versionText, SuccessStyle.Render("✓ up to date"))
		} else {
			fmt.Printf("%s  %s\n", versionText, WarningStyle.Render(fmt.Sprintf("↑ v%s available", cr.latest)))
			result = &UpdateResult{Latest: cr.latest, HasUpdate: true}
		}
	} else {
		fmt.Println(versionText)
	}

	fmt.Println(MutedStyle.Render("  Michael COULLERET <hello@0uf.eu>"))
	fmt.Println(MutedStyle.Render("  Contributors: Thomas Talbot"))
	fmt.Println()

	return result
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
