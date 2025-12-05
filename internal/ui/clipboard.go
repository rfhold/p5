package ui

import (
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// CopiedToClipboardMsg is sent after text is copied to the clipboard
type CopiedToClipboardMsg struct {
	Success bool
	Count   int // Number of items copied (for visual feedback)
}

// CopyToClipboardWithCountCmd returns a command to copy text to the clipboard with a count
func CopyToClipboardWithCountCmd(text string, count int) tea.Cmd {
	return func() tea.Msg {
		success := copyToClipboard(text)
		return CopiedToClipboardMsg{Success: success, Count: count}
	}
}

// copyToClipboard copies text to the system clipboard
func copyToClipboard(text string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return false
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return false
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return false
	}

	if err := cmd.Start(); err != nil {
		return false
	}

	_, err = stdin.Write([]byte(text))
	stdin.Close()
	if err != nil {
		return false
	}

	return cmd.Wait() == nil
}
