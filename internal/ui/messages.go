package ui

import tea "github.com/charmbracelet/bubbletea"

// FlowsUpdatedMsg signals that the ViewModel has new/updated flows.
type FlowsUpdatedMsg struct{}

// StatusMsg sets a temporary status bar message.
type StatusMsg struct{ Text string }

// ClearStatusMsg clears the status bar message.
type ClearStatusMsg struct{}

// ModalCloseMsg closes the active modal.
type ModalCloseMsg struct{}

// EditorFinishedMsg is sent after an external editor returns.
type EditorFinishedMsg struct{ Err error }

// ReplayResultMsg carries the result of an async replay.
type ReplayResultMsg struct{ Err error }

// ListenForUpdates returns a tea.Cmd that waits for a ViewModel update
// and sends FlowsUpdatedMsg. Must be re-invoked after each message.
func ListenForUpdates(ch <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		_, ok := <-ch
		if !ok {
			return nil
		}
		return FlowsUpdatedMsg{}
	}
}
