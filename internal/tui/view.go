package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ginbear/k8s-envtop/internal/env"
	"github.com/ginbear/k8s-envtop/internal/k8s"
)

// View renders the TUI
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Handle different view modes
	switch m.viewMode {
	case ViewModeRevealMenu:
		return m.renderRevealMenu()
	case ViewModeRevealConfirm:
		return m.renderRevealConfirm()
	case ViewModeRevealShow:
		return m.renderRevealShow()
	case ViewModeDiffSelect:
		return m.renderDiffSelect()
	case ViewModeDiffShow:
		return m.renderDiffView()
	}

	// Normal view with 3 panes
	return m.renderNormalView()
}

// renderNormalView renders the main 3-pane layout
func (m Model) renderNormalView() string {
	// Calculate pane widths
	totalWidth := m.width - 4 // Account for borders
	nsWidth := totalWidth / 5
	appsWidth := totalWidth / 5
	envWidth := totalWidth - nsWidth - appsWidth

	// Calculate pane height
	paneHeight := m.height - 6 // Account for header, help, and borders

	// Render each pane
	nsPane := m.renderNamespacesPane(nsWidth, paneHeight)
	appsPane := m.renderAppsPane(appsWidth, paneHeight)
	envPane := m.renderEnvPane(envWidth, paneHeight)

	// Join panes horizontally
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, nsPane, appsPane, envPane)

	// Render header
	header := m.renderHeader()

	// Render help
	help := m.renderHelp()

	// Render error if any
	errorLine := ""
	if m.err != nil {
		errorLine = errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Join all parts vertically
	parts := []string{header, mainContent, help}
	if errorLine != "" {
		parts = append(parts, errorLine)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderHeader renders the top header bar
func (m Model) renderHeader() string {
	title := titleStyle.Render("envtop")
	ctx := statusBarStyle.Render(fmt.Sprintf("Context: %s", m.context))

	var status string
	if m.loading {
		status = statusBarStyle.Render("Loading...")
	} else if len(m.namespaces) > 0 {
		ns := m.namespaces[m.namespaceIdx]
		appName := ""
		if len(m.apps) > 0 && m.appIdx < len(m.apps) {
			appName = m.apps[m.appIdx].Name
		}
		if appName != "" {
			status = statusBarStyle.Render(fmt.Sprintf("%s / %s", ns, appName))
		} else {
			status = statusBarStyle.Render(ns)
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", ctx, "  ", status)
}

// renderHelp renders the help bar at the bottom
func (m Model) renderHelp() string {
	keys := []string{
		helpKeyStyle.Render("Tab") + helpStyle.Render(": switch pane"),
		helpKeyStyle.Render("↑↓") + helpStyle.Render(": move"),
		helpKeyStyle.Render("Enter") + helpStyle.Render(": select"),
		helpKeyStyle.Render("r") + helpStyle.Render(": reveal"),
		helpKeyStyle.Render("d") + helpStyle.Render(": diff"),
		helpKeyStyle.Render("q") + helpStyle.Render(": quit"),
	}
	return helpStyle.Render(strings.Join(keys, "  "))
}

// renderNamespacesPane renders the namespaces pane
func (m Model) renderNamespacesPane(width, height int) string {
	style := GetPaneStyle(m.activePane == PaneNamespaces)
	style = style.Width(width).Height(height)

	title := titleStyle.Render("Namespaces")
	content := []string{title}

	maxItems := height - 3
	startIdx := 0
	if m.namespaceCursor >= maxItems {
		startIdx = m.namespaceCursor - maxItems + 1
	}

	for i := startIdx; i < len(m.namespaces) && i < startIdx+maxItems; i++ {
		ns := m.namespaces[i]
		prefix := "  "
		itemStyle := itemStyle

		if i == m.namespaceCursor {
			prefix = "> "
			itemStyle = selectedItemStyle
		}

		// Mark selected namespace
		if i == m.namespaceIdx {
			ns = ns + " *"
		}

		// Truncate if too long
		maxLen := width - 5
		if len(ns) > maxLen {
			ns = ns[:maxLen-3] + "..."
		}

		content = append(content, itemStyle.Render(prefix+ns))
	}

	return style.Render(strings.Join(content, "\n"))
}

// renderAppsPane renders the apps pane
func (m Model) renderAppsPane(width, height int) string {
	style := GetPaneStyle(m.activePane == PaneApps)
	style = style.Width(width).Height(height)

	title := titleStyle.Render("Apps")
	content := []string{title}

	if len(m.apps) == 0 {
		content = append(content, mutedStyle.Render("  No apps found"))
	} else {
		maxItems := height - 3
		startIdx := 0
		if m.appCursor >= maxItems {
			startIdx = m.appCursor - maxItems + 1
		}

		for i := startIdx; i < len(m.apps) && i < startIdx+maxItems; i++ {
			app := m.apps[i]
			prefix := "  "
			itemStyle := itemStyle

			if i == m.appCursor {
				prefix = "> "
				itemStyle = selectedItemStyle
			}

			// Format: name (kind)
			kindBadge := ""
			if app.Kind == k8s.AppKindStatefulSet {
				kindBadge = " [sts]"
			} else {
				kindBadge = " [dep]"
			}

			name := app.Name
			maxLen := width - 10
			if len(name) > maxLen {
				name = name[:maxLen-3] + "..."
			}

			// Mark selected app
			marker := ""
			if i == m.appIdx {
				marker = " *"
			}

			content = append(content, itemStyle.Render(prefix+name+kindBadge+marker))
		}
	}

	return style.Render(strings.Join(content, "\n"))
}

// renderEnvPane renders the env pane
func (m Model) renderEnvPane(width, height int) string {
	style := GetPaneStyle(m.activePane == PaneEnv)
	style = style.Width(width).Height(height)

	title := titleStyle.Render("Environment Variables")
	content := []string{title}

	// Header
	header := fmt.Sprintf("%-20s %-15s %-12s %s", "NAME", "SOURCE", "KIND", "VALUE")
	content = append(content, helpStyle.Render(header))

	if len(m.envVars) == 0 {
		content = append(content, mutedStyle.Render("  No env vars found"))
	} else {
		maxItems := height - 5
		startIdx := 0
		if m.envCursor >= maxItems {
			startIdx = m.envCursor - maxItems + 1
		}

		for i := startIdx; i < len(m.envVars) && i < startIdx+maxItems; i++ {
			ev := m.envVars[i]
			content = append(content, m.renderEnvVarRow(ev, i == m.envCursor, width))
		}
	}

	return style.Render(strings.Join(content, "\n"))
}

// renderEnvVarRow renders a single env var row
func (m Model) renderEnvVarRow(ev k8s.EnvVar, selected bool, width int) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}

	// Name column
	name := ev.Name
	if len(name) > 18 {
		name = name[:15] + "..."
	}

	// Source column
	source := ""
	switch ev.SourceKind {
	case k8s.EnvSourceConfigMap:
		source = "cm/" + ev.SourceName
	case k8s.EnvSourceSecret, k8s.EnvSourceSealedSecret:
		source = "sec/" + ev.SourceName
	case k8s.EnvSourceInline:
		source = "(inline)"
	case k8s.EnvSourceFieldRef:
		source = "(fieldRef)"
	case k8s.EnvSourceResourceRef:
		source = "(resourceRef)"
	default:
		source = "(unknown)"
	}
	if len(source) > 13 {
		source = source[:10] + "..."
	}

	// Kind column
	kind := string(ev.SourceKind)
	if len(kind) > 10 {
		kind = kind[:10]
	}

	// Value column
	value := ev.Value
	maxValueLen := width - 55
	if maxValueLen < 10 {
		maxValueLen = 10
	}
	if len(value) > maxValueLen {
		value = value[:maxValueLen-3] + "..."
	}

	// Add notes for secrets
	notes := ""
	if ev.IsSecret() {
		notes = fmt.Sprintf(" len=%d", ev.ValueLen)
		if ev.IsSealed {
			notes += " sealed"
		}
	}

	// Format the row
	row := fmt.Sprintf("%-18s %-13s %-10s %s%s", name, source, kind, value, notes)

	// Apply styling
	style := itemStyle
	if selected {
		style = selectedItemStyle
	}

	// Color the kind badge
	kindStyle := GetSourceKindStyle(string(ev.SourceKind))
	if ev.IsSecret() {
		row = fmt.Sprintf("%-18s %-13s %s %s%s", name, source, kindStyle.Render(fmt.Sprintf("%-10s", kind)), envSecretStyle.Render(value), envHashStyle.Render(notes))
	} else {
		row = fmt.Sprintf("%-18s %-13s %s %s", name, source, kindStyle.Render(fmt.Sprintf("%-10s", kind)), envValueStyle.Render(value))
	}

	return style.Render(prefix + row)
}

// renderRevealMenu renders the reveal mode selection menu
func (m Model) renderRevealMenu() string {
	dialog := dialogStyle.Width(50)

	title := dialogTitleStyle.Render("Reveal Secret: " + m.revealedEnvName)

	options := []string{
		"Display as Base64",
		"Display as Plain Text",
	}

	content := []string{title, "", "Select display mode:"}

	for i, opt := range options {
		prefix := "  "
		style := dialogTextStyle
		if i == m.revealMenuIdx {
			prefix = "> "
			style = selectedItemStyle
		}
		content = append(content, style.Render(prefix+opt))
	}

	content = append(content, "", helpStyle.Render("↑↓: select  Enter: confirm  Esc: cancel"))

	return m.centerDialog(dialog.Render(strings.Join(content, "\n")))
}

// renderRevealConfirm renders the reveal confirmation dialog
func (m Model) renderRevealConfirm() string {
	dialog := dialogStyle.Width(60)

	title := dialogTitleStyle.Render("⚠️  Security Warning")

	warning := []string{
		title,
		"",
		dialogTextStyle.Render("This operation will display the secret value on screen."),
		"",
		dialogTextStyle.Render("Before proceeding, please confirm:"),
		dialogTextStyle.Render("  • You are not sharing your screen"),
		dialogTextStyle.Render("  • Terminal logging is disabled"),
		dialogTextStyle.Render("  • No one is looking over your shoulder"),
		"",
		dialogTextStyle.Render("Type 'OK' to confirm:"),
		m.revealInput.View(),
		"",
		helpStyle.Render("Enter: confirm  Esc: cancel"),
	}

	return m.centerDialog(dialog.Render(strings.Join(warning, "\n")))
}

// renderRevealShow renders the revealed secret value
func (m Model) renderRevealShow() string {
	dialog := dialogStyle.Width(70)

	modeLabel := "Base64"
	if m.revealMode == RevealModePlain {
		modeLabel = "Plain Text"
	}

	title := dialogTitleStyle.Render("Secret Value: " + m.revealedEnvName + " (" + modeLabel + ")")

	content := []string{
		title,
		"",
		envValueStyle.Render(m.revealedValue),
		"",
		warningStyle.Render("Press any key to close (auto-closes in 30s)"),
	}

	return m.centerDialog(dialog.Render(strings.Join(content, "\n")))
}

// renderDiffSelect renders the namespace selection for diff
func (m Model) renderDiffSelect() string {
	dialog := dialogStyle.Width(50)

	title := dialogTitleStyle.Render("Select namespace to compare with")

	currentNs := m.namespaces[m.namespaceIdx]
	app := ""
	if len(m.apps) > 0 && m.appIdx < len(m.apps) {
		app = m.apps[m.appIdx].Name
	}

	content := []string{
		title,
		"",
		dialogTextStyle.Render(fmt.Sprintf("Compare: %s/%s", currentNs, app)),
		"",
		dialogTextStyle.Render("With namespace:"),
	}

	maxItems := 10
	startIdx := 0
	if m.diffNsIdx >= maxItems {
		startIdx = m.diffNsIdx - maxItems + 1
	}

	for i := startIdx; i < len(m.diffNamespaces) && i < startIdx+maxItems; i++ {
		prefix := "  "
		style := dialogTextStyle
		if i == m.diffNsIdx {
			prefix = "> "
			style = selectedItemStyle
		}
		content = append(content, style.Render(prefix+m.diffNamespaces[i]))
	}

	content = append(content, "", helpStyle.Render("↑↓: select  Enter: compare  Esc: cancel"))

	return m.centerDialog(dialog.Render(strings.Join(content, "\n")))
}

// renderDiffView renders the diff comparison view
func (m Model) renderDiffView() string {
	// Full screen diff view
	title := titleStyle.Render(fmt.Sprintf("Diff: %s vs %s / %s", m.diffNsA, m.diffNsB, m.diffAppName))

	// Header
	header := fmt.Sprintf("%-20s %-20s %-20s %s", "NAME", m.diffNsA, m.diffNsB, "STATUS")

	content := []string{title, "", helpStyle.Render(header), ""}

	maxItems := m.height - 10
	startIdx := 0
	if m.diffCursor >= maxItems {
		startIdx = m.diffCursor - maxItems + 1
	}

	for i := startIdx; i < len(m.diffResults) && i < startIdx+maxItems; i++ {
		result := m.diffResults[i]
		content = append(content, m.renderDiffRow(result, i == m.diffCursor))
	}

	// Help line
	content = append(content, "", helpStyle.Render("↑↓: scroll  Esc: back to main view"))

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// renderDiffRow renders a single diff result row
func (m Model) renderDiffRow(result env.DiffResult, selected bool) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}

	name := result.Name
	if len(name) > 18 {
		name = name[:15] + "..."
	}

	valueA := "(not present)"
	valueB := "(not present)"

	if result.EnvA != nil {
		if result.EnvA.IsSecret() {
			valueA = fmt.Sprintf("HASH: %s", result.EnvA.Hash)
		} else {
			valueA = result.EnvA.Value
		}
	}

	if result.EnvB != nil {
		if result.EnvB.IsSecret() {
			valueB = fmt.Sprintf("HASH: %s", result.EnvB.Hash)
		} else {
			valueB = result.EnvB.Value
		}
	}

	// Truncate values
	if len(valueA) > 18 {
		valueA = valueA[:15] + "..."
	}
	if len(valueB) > 18 {
		valueB = valueB[:15] + "..."
	}

	// Status styling
	statusStyle := diffSameStyle
	switch result.Status {
	case env.DiffStatusValueDiff:
		statusStyle = diffChangedStyle
	case env.DiffStatusOnlyInA:
		statusStyle = diffRemovedStyle
	case env.DiffStatusOnlyInB:
		statusStyle = diffAddedStyle
	}

	status := statusStyle.Render(string(result.Status))

	row := fmt.Sprintf("%-18s %-18s %-18s %s", name, valueA, valueB, status)

	if selected {
		return selectedItemStyle.Render(prefix + row)
	}
	return itemStyle.Render(prefix + row)
}

// centerDialog centers a dialog on the screen
func (m Model) centerDialog(dialog string) string {
	dialogHeight := strings.Count(dialog, "\n") + 1
	dialogWidth := lipgloss.Width(dialog)

	paddingTop := (m.height - dialogHeight) / 2
	paddingLeft := (m.width - dialogWidth) / 2

	if paddingTop < 0 {
		paddingTop = 0
	}
	if paddingLeft < 0 {
		paddingLeft = 0
	}

	// Create vertical padding
	verticalPadding := strings.Repeat("\n", paddingTop)

	// Create horizontal padding
	lines := strings.Split(dialog, "\n")
	paddedLines := make([]string, len(lines))
	for i, line := range lines {
		paddedLines[i] = strings.Repeat(" ", paddingLeft) + line
	}

	return verticalPadding + strings.Join(paddedLines, "\n")
}
