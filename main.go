package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ginbear/k8s-envtop/internal/k8s"
	"github.com/ginbear/k8s-envtop/internal/tui"
)

func main() {
	// Initialize Kubernetes client
	client, err := k8s.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize Kubernetes client: %v\n", err)
		fmt.Fprintln(os.Stderr, "Please ensure your kubeconfig is properly configured.")
		os.Exit(1)
	}

	// Create TUI model
	model := tui.NewModel(client)

	// Create and run the Bubble Tea program
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running envtop: %v\n", err)
		os.Exit(1)
	}
}
