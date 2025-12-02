package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#06B6D4") // Cyan
	accentColor    = lipgloss.Color("#F59E0B") // Amber
	errorColor     = lipgloss.Color("#EF4444") // Red
	successColor   = lipgloss.Color("#10B981") // Green
	warningColor   = lipgloss.Color("#F59E0B") // Amber
	mutedColor     = lipgloss.Color("#6B7280") // Gray
	bgColor        = lipgloss.Color("#1F2937") // Dark gray
	fgColor        = lipgloss.Color("#F9FAFB") // Almost white

	// Base styles
	baseStyle = lipgloss.NewStyle().
			Foreground(fgColor)

	// Pane styles
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(0, 1)

	activePaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)

	// Title styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	// List item styles
	itemStyle = lipgloss.NewStyle().
			Foreground(fgColor)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	// Status styles
	statusBarStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 1)

	// Help styles
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	// Env table styles
	envNameStyle = lipgloss.NewStyle().
			Foreground(fgColor)

	envValueStyle = lipgloss.NewStyle().
			Foreground(successColor)

	envSecretStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	envHashStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Diff styles
	diffSameStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	diffChangedStyle = lipgloss.NewStyle().
				Foreground(warningColor)

	diffAddedStyle = lipgloss.NewStyle().
			Foreground(successColor)

	diffRemovedStyle = lipgloss.NewStyle().
				Foreground(errorColor)

	// Dialog styles
	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(warningColor).
			Padding(1, 2).
			Width(60)

	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(warningColor).
				MarginBottom(1)

	dialogTextStyle = lipgloss.NewStyle().
			Foreground(fgColor)

	// Error styles
	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	// Muted and warning styles (for use with .Render())
	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	// Source kind badge styles
	configMapBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981")).
				Bold(true)

	secretBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B")).
				Bold(true)

	sealedSecretBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#EF4444")).
				Bold(true)
)

// GetPaneStyle returns the style for a pane based on whether it's active
func GetPaneStyle(active bool) lipgloss.Style {
	if active {
		return activePaneStyle
	}
	return paneStyle
}

// GetSourceKindStyle returns the style for a source kind badge
func GetSourceKindStyle(kind string) lipgloss.Style {
	switch kind {
	case "ConfigMap":
		return configMapBadgeStyle
	case "Secret":
		return secretBadgeStyle
	case "SealedSecret":
		return sealedSecretBadgeStyle
	default:
		return itemStyle
	}
}
