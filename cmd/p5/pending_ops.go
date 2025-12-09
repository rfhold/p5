package main

import tea "github.com/charmbracelet/bubbletea"

// executePendingOps converts pending operations into tea.Cmds
func (m *Model) executePendingOps(ops []PendingOperation) tea.Cmd {
	if len(ops) == 0 {
		return nil
	}

	var cmds []tea.Cmd
	for _, op := range ops {
		if cmd := m.executePendingOp(op); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// executePendingOp converts a single pending operation into a tea.Cmd
func (m *Model) executePendingOp(op PendingOperation) tea.Cmd {
	switch op.Type {
	case "preview":
		return m.initPreview(m.state.Operation)
	case "load_resources":
		return m.loadStackResources()
	case "init_load_resources":
		return m.initLoadStackResources()
	default:
		return nil
	}
}
