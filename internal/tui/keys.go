package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the application
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Enter    key.Binding
	Back     key.Binding
	Reveal   key.Binding
	Diff     key.Binding
	Search   key.Binding
	Seal     key.Binding
	Quit     key.Binding
	Help     key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left pane"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right pane"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "next pane"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("Shift+Tab", "prev pane"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", "back"),
		),
		Reveal: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reveal secret"),
		),
		Diff: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "diff mode"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Seal: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "seal value"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n", "esc"),
			key.WithHelp("n/Esc", "cancel"),
		),
	}
}

// ShortHelp returns the short help text
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Up, k.Down, k.Enter, k.Search, k.Reveal, k.Seal, k.Diff, k.Quit}
}

// FullHelp returns the full help text
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Tab, k.ShiftTab, k.Enter, k.Back},
		{k.Search, k.Reveal, k.Seal, k.Diff, k.Quit},
	}
}
