package ui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	// ErrUserAbort is returned when the user cancels a prompt (ESC / Ctrl+C).
	ErrUserAbort = errors.New("user abort")

	// Theme colors
	Primary   = lipgloss.Color("#7C3AED")
	Secondary = lipgloss.Color("#A78BFA")
	Accent    = lipgloss.Color("#22D3EE")
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

// SelectOption represents a display/value pair for select prompts.
type SelectOption struct {
	Display string
	Value   string
}

// selectModel is a bubbletea model for native select with filtering.
type selectModel struct {
	title       string
	allOptions  []string
	options     []string          // filtered options
	displayMap  map[string]string // for SelectWithOptions
	cursor      int
	filter      string
	selected    string
	aborted     bool
	useDisplay  bool
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.aborted = true
			return m, tea.Quit
		case "enter":
			if len(m.options) > 0 {
				m.selected = m.options[m.cursor]
				return m, tea.Quit
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.applyFilter()
				m.cursor = 0
			}
		default:
			// Add character to filter
			if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
				m.filter += msg.String()
				m.applyFilter()
				m.cursor = 0
			}
		}
	}
	return m, nil
}

func (m *selectModel) applyFilter() {
	if m.filter == "" {
		m.options = m.allOptions
	} else {
		m.options = []string{}
		filter := strings.ToLower(m.filter)
		for _, opt := range m.allOptions {
			if strings.Contains(strings.ToLower(opt), filter) {
				m.options = append(m.options, opt)
			}
		}
	}
	if m.cursor >= len(m.options) {
		m.cursor = len(m.options) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m selectModel) View() string {
	if m.aborted {
		return ""
	}

	var s strings.Builder

	// Title
	s.WriteString(TitleStyle.Render("? " + m.title))
	s.WriteString("\n")

	// Filter bar (if items > 8)
	if len(m.allOptions) > 8 {
		filterPrompt := MutedStyle.Render("/ ")
		s.WriteString(filterPrompt + m.filter + "_\n")
		s.WriteString("\n")
	}

	// Options or no results message
	if len(m.options) == 0 && m.filter != "" {
		// No results message
		s.WriteString("\n")
		s.WriteString(ErrorStyle.Render("  No results for \"" + m.filter + "\""))
		s.WriteString("\n")
		s.WriteString(MutedStyle.Render("  (press backspace to clear filter)"))
	} else {
		// Display options
		for i, opt := range m.options {
			if i == m.cursor {
				// Selected option
				s.WriteString(lipgloss.NewStyle().Foreground(Accent).Render("▸ "))
				s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render(opt))
			} else {
				// Unselected option
				s.WriteString("  ")
				s.WriteString(MutedStyle.Render(opt))
			}
			s.WriteString("\n")
		}
	}

	// Show count
	if len(m.allOptions) > 8 {
		count := fmt.Sprintf("%d/%d", len(m.options), len(m.allOptions))
		s.WriteString("\n")
		s.WriteString(MutedStyle.Render(count))
	}

	return s.String()
}

// Select displays an interactive selection prompt with filtering and ESC support.
func Select(label string, options []string) (string, error) {
	m := selectModel{
		title:      label,
		allOptions: options,
		options:    options,
		cursor:     0,
		filter:     "",
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(selectModel)

	if result.aborted {
		return "", ErrUserAbort
	}

	return result.selected, nil
}

// SelectWithOptions displays a selection prompt with separate display/value pairs.
func SelectWithOptions(label string, options []SelectOption) (string, error) {
	displayMap := make(map[string]string)
	displays := make([]string, len(options))

	for i, opt := range options {
		displays[i] = opt.Display
		displayMap[opt.Display] = opt.Value
	}

	m := selectModel{
		title:      label,
		allOptions: displays,
		options:    displays,
		displayMap: displayMap,
		cursor:     0,
		filter:     "",
		useDisplay: true,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(selectModel)

	if result.aborted {
		return "", ErrUserAbort
	}

	// Return value instead of display
	if result.useDisplay && len(result.displayMap) > 0 {
		return result.displayMap[result.selected], nil
	}

	return result.selected, nil
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

// Confirm displays a yes/no prompt (using huh).
func Confirm(label string) (bool, error) {
	var confirmed bool

	c := huh.NewConfirm().
		Title(label).
		Value(&confirmed)

	err := huh.NewForm(huh.NewGroup(c)).Run()
	if err != nil {
		return false, ErrUserAbort
	}

	return confirmed, nil
}

// Input displays a text input prompt (using huh).
func Input(label, placeholder string) (string, error) {
	var value string

	i := huh.NewInput().
		Title(label).
		Placeholder(placeholder).
		Value(&value)

	err := huh.NewForm(huh.NewGroup(i)).Run()
	if err != nil {
		return "", ErrUserAbort
	}

	return value, nil
}
