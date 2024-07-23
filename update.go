package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neuralink/tsui/libts"
	"github.com/neuralink/tsui/ui"
)

// Message triggered on each poller tick.
type tickMsg struct{}

// Creates a tea.Tick command that generates tickMsg messages.
func makeTick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

// Message representing a Tailscale state update.
type stateMsg libts.State

// Command that retrives a new Tailscale state and triggers a stateMsg.
// This will be run in a goroutine by the bubbletea runtime.
func updateState() tea.Msg {
	status, _ := libts.Status(ctx)
	state := libts.MakeState(status)
	return stateMsg(state)
}

// Bubbletea update function.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Create our ticker command which will be our "default return" in the absence of any other commands.
	tick := makeTick(5 * m.tickInterval)

	switch msg := msg.(type) {
	// On tick, fetch a new state.
	case tickMsg:
		return m, updateState

	// When the state updater returns, update our model.
	case stateMsg:
		m.state = libts.State(msg)

		// Update the exit node submenu.
		{
			exitNodeItems := make([]ui.SubmenuItem, 2+len(m.state.SortedExitNodes))
			exitNodeItems[0] = &ui.ToggleableSubmenuItem{
				Label: "None",
				OnActivate: func() tea.Msg {
					libts.SetExitNode(ctx, nil)
					return updateState()
				},
				IsActive: m.state.CurrentExitNode == nil,
			}
			exitNodeItems[1] = &ui.DividerSubmenuItem{}
			for i, exitNode := range m.state.SortedExitNodes {
				// Offset for the "None" item and the divider.
				i += 2

				label := libts.PeerName(exitNode)
				if !exitNode.Online {
					label += " (offline)"
				}

				exitNodeItems[i] = &ui.ToggleableSubmenuItem{
					Label: label,
					OnActivate: func() tea.Msg {
						libts.SetExitNode(ctx, exitNode)
						return updateState()
					},
					IsActive: m.state.CurrentExitNode != nil && exitNode.ID == *m.state.CurrentExitNode,
					IsDim:    !exitNode.Online,
				}
			}

			m.exitNodes.RightLabel = m.state.CurrentExitNodeName
			m.exitNodes.Submenu.SetItems(exitNodeItems)
		}

	case tea.WindowSizeMsg:
		needsClear := msg.Width < m.terminalWidth || msg.Height > m.terminalHeight

		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

		// Needed to clear artifacts in certain terminals.
		if needsClear {
			return m, tea.ClearScreen
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.menu.IsSubmenuOpen() {
				m.menu.CloseSubmenu()
			} else {
				return m, tea.Quit
			}
		case "left":
			m.menu.CloseSubmenu()
		case "up":
			m.menu.CursorUp()
		case "down":
			m.menu.CursorDown()
		case "right":
			if !m.menu.IsSubmenuOpen() {
				return m, m.menu.Activate()
			}
		case "enter", " ":
			return m, m.menu.Activate()
		}
	}

	return m, tick
}
