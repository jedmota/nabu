package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FilePickerState holds file picker modal state.
type FilePickerState struct {
	input   textinput.Model
	listing string // rendered directory listing
}

func (m *Model) openFilePicker() {
	m.activeModal = ModalImportHAR
	m.filePicker.input = textinput.New()
	m.filePicker.input.Focus()
	m.filePicker.input.Width = 70

	startDir, _ := os.Getwd()
	if !strings.HasSuffix(startDir, string(os.PathSeparator)) {
		startDir += string(os.PathSeparator)
	}
	m.filePicker.input.SetValue(startDir)
	m.filePicker.listing = m.filePickerRefresh(startDir)
}

func (m *Model) updateFilePicker(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.activeModal = ModalNone
		return nil
	case "tab":
		m.filePickerTabComplete()
		return nil
	case "enter":
		text := strings.TrimSpace(m.filePicker.input.Value())
		if text != "" {
			text = fpExpandHome(text)
			m.activeModal = ModalNone
			m.doImportHAR(text)
		}
		return nil
	}

	var cmd tea.Cmd
	m.filePicker.input, cmd = m.filePicker.input.Update(msg)
	m.filePicker.listing = m.filePickerRefresh(m.filePicker.input.Value())
	return cmd
}

func (m *Model) renderFilePicker() string {
	var sb strings.Builder
	sb.WriteString(modalTitleStyle().Render("Import HAR"))
	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(colorSubtle).Bold(true).Render("Path:"))
	sb.WriteString("\n")
	sb.WriteString(m.filePicker.input.View())
	sb.WriteString("\n\n")
	sb.WriteString(m.filePicker.listing)
	sb.WriteString("\n\n")
	sb.WriteString(modalHintStyle.Render("Tab:complete | Enter:select | Esc:cancel"))
	return modalStyle().Width(80).Render(sb.String())
}

func (m *Model) filePickerRefresh(text string) string {
	text = fpExpandHome(text)
	if text == "" {
		return ""
	}

	var dir, prefix string
	if strings.HasSuffix(text, string(os.PathSeparator)) {
		dir = text
		prefix = ""
	} else {
		dir = filepath.Dir(text)
		prefix = filepath.Base(text)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	sort.Slice(entries, func(i, j int) bool {
		di, dj := entries[i].IsDir(), entries[j].IsDir()
		if di != dj {
			return di
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	dirStyle := lipgloss.NewStyle().Foreground(colorBlue).Bold(true)
	fileStyle := lipgloss.NewStyle().Foreground(colorWhite)
	overflowStyle := lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

	var sb strings.Builder
	count := 0
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			continue
		}
		if count >= 20 {
			sb.WriteString(overflowStyle.Render("  ..."))
			break
		}
		if e.IsDir() {
			sb.WriteString("  " + dirStyle.Render(name+"/"))
		} else {
			sb.WriteString("  " + fileStyle.Render(name))
		}
		sb.WriteString("\n")
		count++
	}

	return sb.String()
}

func (m *Model) filePickerTabComplete() {
	text := fpExpandHome(m.filePicker.input.Value())
	if text == "" || strings.HasSuffix(text, string(os.PathSeparator)) {
		return
	}

	dir := filepath.Dir(text) + string(os.PathSeparator)
	prefix := filepath.Base(text)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var matches []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			if e.IsDir() {
				matches = append(matches, name+string(os.PathSeparator))
			} else {
				matches = append(matches, name)
			}
		}
	}

	if len(matches) == 0 {
		return
	}

	if len(matches) == 1 {
		completed := filepath.Join(dir, matches[0])
		if strings.HasSuffix(matches[0], string(os.PathSeparator)) {
			completed += string(os.PathSeparator)
		}
		m.filePicker.input.SetValue(completed)
		m.filePicker.listing = m.filePickerRefresh(completed)
		return
	}

	// Multiple matches - complete to longest common prefix
	lcp := matches[0]
	for _, match := range matches[1:] {
		lcp = fpCommonPrefix(lcp, match)
	}
	if len(lcp) > len(prefix) {
		m.filePicker.input.SetValue(filepath.Join(dir, lcp))
		m.filePicker.listing = m.filePickerRefresh(filepath.Join(dir, lcp))
	}
}

func fpExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

func fpCommonPrefix(a, b string) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return a[:i]
}
