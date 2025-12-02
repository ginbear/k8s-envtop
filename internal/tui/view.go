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

// renderNormalView renders the 2-row layout
// Top row: [Namespaces] [Apps]
// Bottom row: [Environment Variables]
func (m Model) renderNormalView() string {
	// Render header first
	header := m.renderHeader()

	// Render help
	help := m.renderHelp()

	// Render error if any
	errorLine := ""
	if m.err != nil {
		errorLine = errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Calculate available height for panes
	// Total height minus: header(1) + help(1) + error(0-1) + padding(1)
	usedHeight := 4
	if errorLine != "" {
		usedHeight++
	}
	availableHeight := m.height - usedHeight
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Calculate dimensions
	totalWidth := m.width - 4 // Account for borders

	// Top row: NS and Apps split equally, use ~1/3 of available height
	topRowWidth := totalWidth / 2
	topRowHeight := availableHeight / 3
	if topRowHeight < 5 {
		topRowHeight = 5
	}

	// Bottom row: Env takes full width and remaining height
	envWidth := totalWidth
	envHeight := availableHeight - topRowHeight - 2 // -2 for spacing
	if envHeight < 5 {
		envHeight = 5
	}

	// Render top row panes
	nsPane := m.renderNamespacesPane(topRowWidth-1, topRowHeight)
	appsPane := m.renderAppsPane(topRowWidth-1, topRowHeight)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, nsPane, appsPane)

	// Render bottom row (env pane)
	envPane := m.renderEnvPane(envWidth, envHeight)

	// Join all parts vertically
	parts := []string{header, topRow, envPane, help}
	if errorLine != "" {
		parts = append(parts, errorLine)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderHeader renders the top header bar
func (m Model) renderHeader() string {
	title := titleStyle.Render("envtop")
	ctx := fmt.Sprintf("Context: %s", m.context)

	var status string
	if m.loading {
		status = "Loading..."
	} else if len(m.namespaces) > 0 {
		ns := m.namespaces[m.namespaceIdx]
		appName := ""
		if len(m.apps) > 0 && m.appIdx < len(m.apps) {
			appName = m.apps[m.appIdx].Name
		}
		if appName != "" {
			status = fmt.Sprintf("| %s / %s", ns, appName)
		} else {
			status = fmt.Sprintf("| %s", ns)
		}
	}

	return fmt.Sprintf("%s  %s  %s", title, ctx, status)
}

// renderHelp renders the help bar at the bottom
func (m Model) renderHelp() string {
	if m.viewMode == ViewModeSearch {
		keys := []string{
			helpKeyStyle.Render("Type") + helpStyle.Render(": filter"),
			helpKeyStyle.Render("↑↓") + helpStyle.Render(": move"),
			helpKeyStyle.Render("Enter") + helpStyle.Render(": select"),
			helpKeyStyle.Render("Esc") + helpStyle.Render(": cancel"),
		}
		return helpStyle.Render(strings.Join(keys, "  "))
	}
	keys := []string{
		helpKeyStyle.Render("Tab") + helpStyle.Render(": switch pane"),
		helpKeyStyle.Render("↑↓") + helpStyle.Render(": move"),
		helpKeyStyle.Render("Enter") + helpStyle.Render(": select"),
		helpKeyStyle.Render("/") + helpStyle.Render(": search"),
		helpKeyStyle.Render("r") + helpStyle.Render(": reveal"),
		helpKeyStyle.Render("d") + helpStyle.Render(": diff"),
		helpKeyStyle.Render("q") + helpStyle.Render(": quit"),
	}
	return helpStyle.Render(strings.Join(keys, "  "))
}

// renderNamespacesPane renders the namespaces pane
func (m Model) renderNamespacesPane(width, height int) string {
	isSearching := m.IsSearchingPane(PaneNamespaces)
	style := GetPaneStyle(m.activePane == PaneNamespaces || isSearching)
	style = style.Width(width).Height(height)

	title := titleStyle.Render("Namespaces")
	content := []string{title}

	// Show search input if searching this pane
	if isSearching {
		content = append(content, m.searchInput.View())
	}

	// Get filtered indices
	filteredIndices := m.GetFilteredNamespaces()

	maxItems := height - 3
	if isSearching {
		maxItems-- // Account for search input
	}
	startIdx := 0
	if m.namespaceCursor >= maxItems {
		startIdx = m.namespaceCursor - maxItems + 1
	}

	for cursorPos := startIdx; cursorPos < len(filteredIndices) && cursorPos < startIdx+maxItems; cursorPos++ {
		i := filteredIndices[cursorPos]
		ns := m.namespaces[i]
		prefix := "  "
		style := itemStyle

		if cursorPos == m.namespaceCursor {
			prefix = "> "
			style = selectedItemStyle
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

		content = append(content, style.Render(prefix+ns))
	}

	if len(filteredIndices) == 0 {
		content = append(content, mutedStyle.Render("  No matches"))
	}

	return GetPaneStyle(m.activePane == PaneNamespaces || isSearching).Width(width).Height(height).Render(strings.Join(content, "\n"))
}

// renderAppsPane renders the apps pane
func (m Model) renderAppsPane(width, height int) string {
	isSearching := m.IsSearchingPane(PaneApps)
	style := GetPaneStyle(m.activePane == PaneApps || isSearching)
	style = style.Width(width).Height(height)

	title := titleStyle.Render("Apps")
	content := []string{title}

	// Show search input if searching this pane
	if isSearching {
		content = append(content, m.searchInput.View())
	}

	// Get filtered indices
	filteredIndices := m.GetFilteredApps()

	if len(m.apps) == 0 {
		content = append(content, mutedStyle.Render("  No apps found"))
	} else if len(filteredIndices) == 0 {
		content = append(content, mutedStyle.Render("  No matches"))
	} else {
		maxItems := height - 3
		if isSearching {
			maxItems--
		}
		startIdx := 0
		if m.appCursor >= maxItems {
			startIdx = m.appCursor - maxItems + 1
		}

		for cursorPos := startIdx; cursorPos < len(filteredIndices) && cursorPos < startIdx+maxItems; cursorPos++ {
			i := filteredIndices[cursorPos]
			app := m.apps[i]
			prefix := "  "
			style := itemStyle

			if cursorPos == m.appCursor {
				prefix = "> "
				style = selectedItemStyle
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

			content = append(content, style.Render(prefix+name+kindBadge+marker))
		}
	}

	return GetPaneStyle(m.activePane == PaneApps || isSearching).Width(width).Height(height).Render(strings.Join(content, "\n"))
}

// renderEnvPane renders the env pane
func (m Model) renderEnvPane(width, height int) string {
	isSearching := m.IsSearchingPane(PaneEnv)
	style := GetPaneStyle(m.activePane == PaneEnv || isSearching)
	style = style.Width(width).Height(height)

	title := titleStyle.Render("Environment Variables")
	content := []string{title}

	// Show search input if searching this pane
	if isSearching {
		content = append(content, m.searchInput.View())
	}

	// Header
	header := fmt.Sprintf("%-30s %-25s %-14s %s", "NAME", "SOURCE", "KIND", "VALUE")
	content = append(content, helpStyle.Render(header))

	// Get filtered indices
	filteredIndices := m.GetFilteredEnvVars()

	if len(m.envVars) == 0 {
		content = append(content, mutedStyle.Render("  No env vars found"))
	} else if len(filteredIndices) == 0 {
		content = append(content, mutedStyle.Render("  No matches"))
	} else {
		maxItems := height - 5
		if isSearching {
			maxItems--
		}
		startIdx := 0
		if m.envCursor >= maxItems {
			startIdx = m.envCursor - maxItems + 1
		}

		for cursorPos := startIdx; cursorPos < len(filteredIndices) && cursorPos < startIdx+maxItems; cursorPos++ {
			i := filteredIndices[cursorPos]
			ev := m.envVars[i]
			content = append(content, m.renderEnvVarRow(ev, cursorPos == m.envCursor, width))
		}
	}

	return GetPaneStyle(m.activePane == PaneEnv || isSearching).Width(width).Height(height).Render(strings.Join(content, "\n"))
}

// renderEnvVarRow renders a single env var row
func (m Model) renderEnvVarRow(ev k8s.EnvVar, selected bool, width int) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}

	// Name column (max 28 chars)
	name := ev.Name
	if len(name) > 28 {
		name = name[:25] + "..."
	}

	// Source column (max 23 chars)
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
	if len(source) > 23 {
		source = source[:20] + "..."
	}

	// Kind column (max 12 chars)
	kind := string(ev.SourceKind)
	if len(kind) > 12 {
		kind = kind[:12]
	}

	// Value column (use remaining width)
	value := ev.Value
	maxValueLen := width - 75 // Adjusted for wider columns
	if maxValueLen < 20 {
		maxValueLen = 20
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
	row := fmt.Sprintf("%-28s %-23s %-12s %s%s", name, source, kind, value, notes)

	// Apply styling
	style := itemStyle
	if selected {
		style = selectedItemStyle
	}

	// Color the kind badge
	kindStyle := GetSourceKindStyle(string(ev.SourceKind))
	if ev.IsSecret() {
		row = fmt.Sprintf("%-28s %-23s %s %s%s", name, source, kindStyle.Render(fmt.Sprintf("%-12s", kind)), envSecretStyle.Render(value), envHashStyle.Render(notes))
	} else {
		row = fmt.Sprintf("%-28s %-23s %s %s", name, source, kindStyle.Render(fmt.Sprintf("%-12s", kind)), envValueStyle.Render(value))
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
