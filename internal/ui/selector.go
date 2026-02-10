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
	Accent    = lipgloss.Color("#22D3EE") // Cyan — used for interactive elements
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

// devTheme returns a custom huh theme — cyan accent, no purple.
func devTheme() *huh.Theme {
	t := huh.ThemeBase()

	t.Focused.Base = lipgloss.NewStyle().PaddingLeft(1)
	t.Focused.Card = t.Focused.Base

	// Title in white bold
	t.Focused.Title = lipgloss.NewStyle().Foreground(Text).Bold(true)

	// Select: cyan arrow, white selected, gray others
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(Accent).SetString("▸ ")
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	t.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(Muted)
	t.Focused.Option = lipgloss.NewStyle().Foreground(Muted)

	// Scroll indicators
	t.Focused.NextIndicator = lipgloss.NewStyle().Foreground(Muted).SetString("  ↓")
	t.Focused.PrevIndicator = lipgloss.NewStyle().Foreground(Muted).SetString("  ↑")

	// Filter input
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(Accent)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(Muted)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(Accent).SetString("/ ")
	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(Text)

	// Buttons
	t.Focused.FocusedButton = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(Accent).Padding(0, 1)
	t.Focused.BlurredButton = lipgloss.NewStyle().Foreground(Muted).Background(lipgloss.Color("#333")).Padding(0, 1)

	// Blurred = same but no indicators
	t.Blurred = t.Focused
	t.Blurred.Base = lipgloss.NewStyle().PaddingLeft(1)
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	return t
}

func selectHeight(count int) int {
	// Generous height so all items stay visible
	if count <= 8 {
		return count + 6
	}
	h := count + 5
	if h > 20 {
		h = 20
	}
	return h
}

// SelectOption represents a display/value pair for select prompts.
type SelectOption struct {
	Display string
	Value   string
}

// Select displays an interactive selection prompt.
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
		Filtering(false)

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
		Filtering(false)

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
	versionPart := MutedStyle.Render(fmt.Sprintf("  v%s", version))
	sloganPart := lipgloss.NewStyle().Foreground(Text).Render(" — Focus on coding, not on tooling.")
	fmt.Println(versionPart + sloganPart)
	fmt.Println(MutedStyle.Render("  Michael COULLERET, Thomas Talbot and contributors."))
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

	versionPart := MutedStyle.Render(fmt.Sprintf("  v%s", version))
	sloganPart := lipgloss.NewStyle().Foreground(Text).Render(" — Focus on coding, not on tooling.")
	versionText := versionPart + sloganPart

	var result *UpdateResult

	if checkFn != nil {
		// Show discreet loading indicator
		fmt.Printf("%s  %s", versionText, MutedStyle.Render("⟳"))

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

		// Clear line, reprint with status
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

	fmt.Println(MutedStyle.Render("  Michael COULLERET, Thomas Talbot and contributors."))
	fmt.Println()
	usageLabel := lipgloss.NewStyle().Foreground(Text).Bold(true).Render("  Usage:")
	usageDetail := MutedStyle.Render(" devcli [command] [options]")
	fmt.Println(usageLabel + usageDetail)
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
