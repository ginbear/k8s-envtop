package tui

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ginbear/k8s-envtop/internal/env"
	"github.com/ginbear/k8s-envtop/internal/k8s"
)

// Pane represents the active pane
type Pane int

const (
	PaneNamespaces Pane = iota
	PaneApps
	PaneEnv
)

// ViewMode represents the current view mode
type ViewMode int

const (
	ViewModeNormal ViewMode = iota
	ViewModeSearch
	ViewModeRevealMenu
	ViewModeRevealConfirm
	ViewModeRevealShow
	ViewModeDiffSelect
	ViewModeDiffShow
)

// RevealMode represents how to display the revealed secret
type RevealMode int

const (
	RevealModeBase64 RevealMode = iota
	RevealModePlain
)

// Model is the main TUI model
type Model struct {
	// Kubernetes client and resolver
	client   *k8s.Client
	resolver *env.Resolver

	// Window dimensions
	width  int
	height int

	// State
	activePane Pane
	viewMode   ViewMode

	// Namespace pane
	namespaces      []string
	namespaceIdx    int
	namespaceCursor int

	// Apps pane
	apps      []k8s.App
	appIdx    int
	appCursor int

	// Env pane
	envVars   []k8s.EnvVar
	envIdx    int
	envCursor int

	// Search state
	searchInput        textinput.Model
	searchPane         Pane
	filteredNamespaces []int // indices into namespaces
	filteredApps       []int // indices into apps
	filteredEnvVars    []int // indices into envVars

	// Reveal state
	revealMode      RevealMode
	revealMenuIdx   int
	revealInput     textinput.Model
	revealedValue   string
	revealedEnvName string
	revealExpiry    time.Time

	// Diff state
	diffNamespaces []string
	diffNsIdx      int
	diffResults    []env.DiffResult
	diffNsA        string
	diffNsB        string
	diffAppName    string
	diffCursor     int

	// Error state
	err     error
	loading bool

	// Key bindings
	keys KeyMap

	// Context
	context       string
	cancelFunc    context.CancelFunc
}

// Messages
type (
	namespacesLoadedMsg struct {
		namespaces []string
	}
	appsLoadedMsg struct {
		apps []k8s.App
	}
	envVarsLoadedMsg struct {
		envVars []k8s.EnvVar
	}
	diffResultsMsg struct {
		results []env.DiffResult
		nsA     string
		nsB     string
		appName string
	}
	errorMsg struct {
		err error
	}
	revealTimeoutMsg struct{}
)

// NewModel creates a new TUI model
func NewModel(client *k8s.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "Type OK to confirm"
	ti.CharLimit = 10
	ti.Width = 20

	si := textinput.New()
	si.Placeholder = "Type to filter..."
	si.CharLimit = 50
	si.Width = 30

	return Model{
		client:        client,
		resolver:      env.NewResolver(client),
		keys:          DefaultKeyMap(),
		activePane:    PaneNamespaces,
		viewMode:      ViewModeNormal,
		revealInput:   ti,
		searchInput:   si,
		context:       client.GetCurrentContext(),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadNamespaces(),
		tea.EnterAltScreen,
	)
}

// loadNamespaces loads the namespace list
func (m Model) loadNamespaces() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		namespaces, err := m.client.ListNamespaces(ctx)
		if err != nil {
			return errorMsg{err: err}
		}
		return namespacesLoadedMsg{namespaces: namespaces}
	}
}

// loadApps loads the apps for the selected namespace
func (m Model) loadApps() tea.Cmd {
	if len(m.namespaces) == 0 {
		return nil
	}
	namespace := m.namespaces[m.namespaceIdx]
	return func() tea.Msg {
		ctx := context.Background()
		apps, err := m.client.ListApps(ctx, namespace)
		if err != nil {
			return errorMsg{err: err}
		}
		return appsLoadedMsg{apps: apps}
	}
}

// loadEnvVars loads the env vars for the selected app
func (m Model) loadEnvVars() tea.Cmd {
	if len(m.apps) == 0 {
		return nil
	}
	app := m.apps[m.appIdx]
	return func() tea.Msg {
		ctx := context.Background()
		envVars, err := m.resolver.ResolveAppEnvVars(ctx, app)
		if err != nil {
			return errorMsg{err: err}
		}
		return envVarsLoadedMsg{envVars: envVars}
	}
}

// loadDiff loads the diff between two namespaces
func (m Model) loadDiff(nsA, nsB, appName string, appKind k8s.AppKind) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		appA := k8s.App{Name: appName, Namespace: nsA, Kind: appKind}
		appB := k8s.App{Name: appName, Namespace: nsB, Kind: appKind}

		resolver := env.NewResolver(m.client)

		envsA, err := resolver.ResolveAppEnvVars(ctx, appA)
		if err != nil {
			return errorMsg{err: err}
		}

		envsB, err := resolver.ResolveAppEnvVars(ctx, appB)
		if err != nil {
			return errorMsg{err: err}
		}

		results := env.CompareEnvVars(envsA, envsB)
		return diffResultsMsg{
			results: results,
			nsA:     nsA,
			nsB:     nsB,
			appName: appName,
		}
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case namespacesLoadedMsg:
		m.namespaces = msg.namespaces
		m.loading = false
		if len(m.namespaces) > 0 {
			return m, m.loadApps()
		}
		return m, nil

	case appsLoadedMsg:
		m.apps = msg.apps
		m.appIdx = 0
		m.appCursor = 0
		m.loading = false
		if len(m.apps) > 0 {
			return m, m.loadEnvVars()
		}
		return m, nil

	case envVarsLoadedMsg:
		m.envVars = msg.envVars
		m.envIdx = 0
		m.envCursor = 0
		m.loading = false
		return m, nil

	case diffResultsMsg:
		m.diffResults = msg.results
		m.diffNsA = msg.nsA
		m.diffNsB = msg.nsB
		m.diffAppName = msg.appName
		m.diffCursor = 0
		m.viewMode = ViewModeDiffShow
		m.loading = false
		return m, nil

	case errorMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case revealTimeoutMsg:
		m.revealedValue = ""
		m.revealedEnvName = ""
		m.viewMode = ViewModeNormal
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	// Update text input if in reveal confirm mode
	if m.viewMode == ViewModeRevealConfirm {
		var cmd tea.Cmd
		m.revealInput, cmd = m.revealInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKeyPress handles key press events
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle quit in any mode
	if key.Matches(msg, m.keys.Quit) && m.viewMode == ViewModeNormal {
		return m, tea.Quit
	}

	// Handle search mode first (before other key bindings interfere)
	if m.viewMode == ViewModeSearch {
		// Only Esc cancels search mode
		if key.Matches(msg, m.keys.Back) {
			m.viewMode = ViewModeNormal
			m.searchInput.Reset()
			m.filteredNamespaces = nil
			m.filteredApps = nil
			m.filteredEnvVars = nil
			return m, nil
		}
		return m.handleSearchMode(msg)
	}

	// Handle escape in special modes
	if key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Cancel) {
		switch m.viewMode {
		case ViewModeRevealMenu, ViewModeRevealConfirm, ViewModeRevealShow:
			m.viewMode = ViewModeNormal
			m.revealInput.Reset()
			m.revealedValue = ""
			return m, nil
		case ViewModeDiffSelect:
			m.viewMode = ViewModeNormal
			return m, nil
		case ViewModeDiffShow:
			m.viewMode = ViewModeNormal
			m.diffResults = nil
			return m, nil
		}
	}

	// Mode-specific handling
	switch m.viewMode {
	case ViewModeNormal:
		return m.handleNormalMode(msg)
	case ViewModeRevealMenu:
		return m.handleRevealMenu(msg)
	case ViewModeRevealConfirm:
		return m.handleRevealConfirm(msg)
	case ViewModeRevealShow:
		return m.handleRevealShow(msg)
	case ViewModeDiffSelect:
		return m.handleDiffSelect(msg)
	case ViewModeDiffShow:
		return m.handleDiffShow(msg)
	}

	return m, nil
}

// handleNormalMode handles key press in normal mode
func (m Model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Tab):
		m.activePane = (m.activePane + 1) % 3
		return m, nil

	case key.Matches(msg, m.keys.ShiftTab):
		m.activePane = (m.activePane + 2) % 3
		return m, nil

	case key.Matches(msg, m.keys.Left):
		if m.activePane > PaneNamespaces {
			m.activePane--
		}
		return m, nil

	case key.Matches(msg, m.keys.Right):
		if m.activePane < PaneEnv {
			m.activePane++
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		return m.handleUp()

	case key.Matches(msg, m.keys.Down):
		return m.handleDown()

	case key.Matches(msg, m.keys.Enter):
		return m.handleEnter()

	case key.Matches(msg, m.keys.Reveal):
		return m.handleRevealStart()

	case key.Matches(msg, m.keys.Diff):
		return m.handleDiffStart()

	case key.Matches(msg, m.keys.Search):
		return m.handleSearchStart()
	}

	return m, nil
}

// handleUp handles up key
func (m Model) handleUp() (tea.Model, tea.Cmd) {
	switch m.activePane {
	case PaneNamespaces:
		if m.namespaceCursor > 0 {
			m.namespaceCursor--
		}
	case PaneApps:
		if m.appCursor > 0 {
			m.appCursor--
		}
	case PaneEnv:
		if m.envCursor > 0 {
			m.envCursor--
		}
	}
	return m, nil
}

// handleDown handles down key
func (m Model) handleDown() (tea.Model, tea.Cmd) {
	switch m.activePane {
	case PaneNamespaces:
		if m.namespaceCursor < len(m.namespaces)-1 {
			m.namespaceCursor++
		}
	case PaneApps:
		if m.appCursor < len(m.apps)-1 {
			m.appCursor++
		}
	case PaneEnv:
		if m.envCursor < len(m.envVars)-1 {
			m.envCursor++
		}
	}
	return m, nil
}

// handleEnter handles enter key
func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.activePane {
	case PaneNamespaces:
		if m.namespaceCursor < len(m.namespaces) {
			m.namespaceIdx = m.namespaceCursor
			m.loading = true
			return m, m.loadApps()
		}
	case PaneApps:
		if m.appCursor < len(m.apps) {
			m.appIdx = m.appCursor
			m.loading = true
			return m, m.loadEnvVars()
		}
	}
	return m, nil
}

// handleRevealStart starts the reveal flow
func (m Model) handleRevealStart() (tea.Model, tea.Cmd) {
	// Check if reveal is disabled
	if os.Getenv("ENVTOP_DISABLE_REVEAL") == "1" {
		m.err = &revealDisabledError{}
		return m, nil
	}

	// Only work in env pane with a secret selected
	if m.activePane != PaneEnv {
		return m, nil
	}

	if len(m.envVars) == 0 || m.envCursor >= len(m.envVars) {
		return m, nil
	}

	envVar := m.envVars[m.envCursor]
	if !envVar.IsSecret() {
		return m, nil
	}

	m.viewMode = ViewModeRevealMenu
	m.revealMenuIdx = 0
	m.revealedEnvName = envVar.Name
	return m, nil
}

// handleRevealMenu handles key press in reveal menu
func (m Model) handleRevealMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.revealMenuIdx > 0 {
			m.revealMenuIdx--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.revealMenuIdx < 1 {
			m.revealMenuIdx++
		}
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		if m.revealMenuIdx == 0 {
			m.revealMode = RevealModeBase64
		} else {
			m.revealMode = RevealModePlain
		}
		m.viewMode = ViewModeRevealConfirm
		m.revealInput.Reset()
		m.revealInput.Focus()
		return m, textinput.Blink
	}

	return m, nil
}

// handleRevealConfirm handles key press in reveal confirm dialog
func (m Model) handleRevealConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Enter):
		if m.revealInput.Value() == "OK" {
			// Find the env var and reveal it
			for _, ev := range m.envVars {
				if ev.Name == m.revealedEnvName {
					if m.revealMode == RevealModeBase64 {
						m.revealedValue = k8s.EncodeBase64(ev.RawValue)
					} else {
						m.revealedValue = string(ev.RawValue)
					}
					break
				}
			}
			m.viewMode = ViewModeRevealShow
			m.revealExpiry = time.Now().Add(30 * time.Second)
			return m, tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
				return revealTimeoutMsg{}
			})
		}
		return m, nil
	}

	// Handle text input
	var cmd tea.Cmd
	m.revealInput, cmd = m.revealInput.Update(msg)
	return m, cmd
}

// handleRevealShow handles key press in reveal show mode
func (m Model) handleRevealShow(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key returns to normal mode
	m.viewMode = ViewModeNormal
	m.revealedValue = ""
	m.revealedEnvName = ""
	return m, nil
}

// handleDiffStart starts the diff flow
func (m Model) handleDiffStart() (tea.Model, tea.Cmd) {
	if len(m.apps) == 0 || m.appCursor >= len(m.apps) {
		return m, nil
	}

	m.diffNamespaces = make([]string, 0, len(m.namespaces))
	currentNs := m.namespaces[m.namespaceIdx]
	for _, ns := range m.namespaces {
		if ns != currentNs {
			m.diffNamespaces = append(m.diffNamespaces, ns)
		}
	}

	if len(m.diffNamespaces) == 0 {
		return m, nil
	}

	m.viewMode = ViewModeDiffSelect
	m.diffNsIdx = 0
	return m, nil
}

// handleDiffSelect handles key press in diff select mode
func (m Model) handleDiffSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.diffNsIdx > 0 {
			m.diffNsIdx--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.diffNsIdx < len(m.diffNamespaces)-1 {
			m.diffNsIdx++
		}
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		nsA := m.namespaces[m.namespaceIdx]
		nsB := m.diffNamespaces[m.diffNsIdx]
		app := m.apps[m.appIdx]
		m.loading = true
		return m, m.loadDiff(nsA, nsB, app.Name, app.Kind)
	}

	return m, nil
}

// handleDiffShow handles key press in diff show mode
func (m Model) handleDiffShow(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.diffCursor > 0 {
			m.diffCursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.diffCursor < len(m.diffResults)-1 {
			m.diffCursor++
		}
		return m, nil
	}

	return m, nil
}

// handleSearchStart starts the search mode
func (m Model) handleSearchStart() (tea.Model, tea.Cmd) {
	m.viewMode = ViewModeSearch
	m.searchPane = m.activePane
	m.searchInput.Reset()
	m.searchInput.Focus()
	m.updateFilter("")
	return m, textinput.Blink
}

// handleSearchMode handles key press in search mode
func (m Model) handleSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Select current item and exit search
		m.applySearchSelection()
		m.viewMode = ViewModeNormal
		m.searchInput.Reset()
		// Load data based on pane
		switch m.searchPane {
		case PaneNamespaces:
			m.loading = true
			return m, m.loadApps()
		case PaneApps:
			m.loading = true
			return m, m.loadEnvVars()
		}
		return m, nil

	case tea.KeyUp, tea.KeyCtrlP:
		m.searchMoveUp()
		return m, nil

	case tea.KeyDown, tea.KeyCtrlN:
		m.searchMoveDown()
		return m, nil

	case tea.KeyCtrlC:
		m.viewMode = ViewModeNormal
		m.searchInput.Reset()
		m.filteredNamespaces = nil
		m.filteredApps = nil
		m.filteredEnvVars = nil
		return m, nil
	}

	// Handle text input
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)

	// Update filter on every keystroke
	m.updateFilter(m.searchInput.Value())

	return m, cmd
}

// updateFilter updates the filtered indices based on search query
func (m *Model) updateFilter(query string) {
	query = strings.ToLower(query)

	switch m.searchPane {
	case PaneNamespaces:
		m.filteredNamespaces = m.filterStrings(m.namespaces, query)
		if len(m.filteredNamespaces) > 0 {
			m.namespaceCursor = 0
		}
	case PaneApps:
		m.filteredApps = nil
		for i, app := range m.apps {
			if query == "" || strings.Contains(strings.ToLower(app.Name), query) {
				m.filteredApps = append(m.filteredApps, i)
			}
		}
		if len(m.filteredApps) > 0 {
			m.appCursor = 0
		}
	case PaneEnv:
		m.filteredEnvVars = nil
		for i, ev := range m.envVars {
			if query == "" || strings.Contains(strings.ToLower(ev.Name), query) {
				m.filteredEnvVars = append(m.filteredEnvVars, i)
			}
		}
		if len(m.filteredEnvVars) > 0 {
			m.envCursor = 0
		}
	}
}

// filterStrings returns indices of strings that match the query
func (m *Model) filterStrings(items []string, query string) []int {
	var result []int
	for i, item := range items {
		if query == "" || strings.Contains(strings.ToLower(item), query) {
			result = append(result, i)
		}
	}
	return result
}

// searchMoveUp moves cursor up in filtered list
func (m *Model) searchMoveUp() {
	switch m.searchPane {
	case PaneNamespaces:
		if m.namespaceCursor > 0 {
			m.namespaceCursor--
		}
	case PaneApps:
		if m.appCursor > 0 {
			m.appCursor--
		}
	case PaneEnv:
		if m.envCursor > 0 {
			m.envCursor--
		}
	}
}

// searchMoveDown moves cursor down in filtered list
func (m *Model) searchMoveDown() {
	switch m.searchPane {
	case PaneNamespaces:
		if len(m.filteredNamespaces) > 0 && m.namespaceCursor < len(m.filteredNamespaces)-1 {
			m.namespaceCursor++
		}
	case PaneApps:
		if len(m.filteredApps) > 0 && m.appCursor < len(m.filteredApps)-1 {
			m.appCursor++
		}
	case PaneEnv:
		if len(m.filteredEnvVars) > 0 && m.envCursor < len(m.filteredEnvVars)-1 {
			m.envCursor++
		}
	}
}

// applySearchSelection applies the current search selection
func (m *Model) applySearchSelection() {
	switch m.searchPane {
	case PaneNamespaces:
		if len(m.filteredNamespaces) > 0 && m.namespaceCursor < len(m.filteredNamespaces) {
			m.namespaceIdx = m.filteredNamespaces[m.namespaceCursor]
		}
		m.filteredNamespaces = nil
	case PaneApps:
		if len(m.filteredApps) > 0 && m.appCursor < len(m.filteredApps) {
			m.appIdx = m.filteredApps[m.appCursor]
		}
		m.filteredApps = nil
	case PaneEnv:
		if len(m.filteredEnvVars) > 0 && m.envCursor < len(m.filteredEnvVars) {
			m.envIdx = m.filteredEnvVars[m.envCursor]
		}
		m.filteredEnvVars = nil
	}
}

// GetFilteredNamespaces returns filtered namespace indices or all if not filtering
func (m *Model) GetFilteredNamespaces() []int {
	if m.viewMode == ViewModeSearch && m.searchPane == PaneNamespaces && m.filteredNamespaces != nil {
		return m.filteredNamespaces
	}
	// Return all indices
	result := make([]int, len(m.namespaces))
	for i := range m.namespaces {
		result[i] = i
	}
	return result
}

// GetFilteredApps returns filtered app indices or all if not filtering
func (m *Model) GetFilteredApps() []int {
	if m.viewMode == ViewModeSearch && m.searchPane == PaneApps && m.filteredApps != nil {
		return m.filteredApps
	}
	// Return all indices
	result := make([]int, len(m.apps))
	for i := range m.apps {
		result[i] = i
	}
	return result
}

// GetFilteredEnvVars returns filtered env var indices or all if not filtering
func (m *Model) GetFilteredEnvVars() []int {
	if m.viewMode == ViewModeSearch && m.searchPane == PaneEnv && m.filteredEnvVars != nil {
		return m.filteredEnvVars
	}
	// Return all indices
	result := make([]int, len(m.envVars))
	for i := range m.envVars {
		result[i] = i
	}
	return result
}

// IsSearchingPane returns true if currently searching in the given pane
func (m *Model) IsSearchingPane(pane Pane) bool {
	return m.viewMode == ViewModeSearch && m.searchPane == pane
}

// Custom errors
type revealDisabledError struct{}

func (e *revealDisabledError) Error() string {
	return "Reveal is disabled (ENVTOP_DISABLE_REVEAL=1)"
}
