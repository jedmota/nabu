package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// FilePicker is a modal with an InputField and a read-only file listing
// that filters in real-time as the user types a path. Tab completes.
type FilePicker struct {
	*tview.Flex

	input *tview.InputField
	list  *tview.TextView

	onSelect func(path string)
	onClose  func()
}

// NewFilePicker creates a new file picker.
func NewFilePicker() *FilePicker {
	fp := &FilePicker{}

	fp.input = tview.NewInputField()
	fp.input.SetLabel(" Path: ")
	fp.input.SetFieldWidth(0)
	fp.input.SetFieldBackgroundColor(tcell.ColorDefault)
	fp.input.SetFieldTextColor(tcell.ColorWhite)

	fp.list = tview.NewTextView()
	fp.list.SetDynamicColors(true)
	fp.list.SetScrollable(true)

	inner := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(fp.input, 1, 0, true).
		AddItem(fp.list, 0, 1, false)
	inner.SetBorder(true)
	inner.SetTitle(" Import HAR ")
	inner.SetTitleAlign(tview.AlignCenter)

	fp.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(inner, 90, 0, true).
			AddItem(nil, 0, 1, false), 20, 0, true).
		AddItem(nil, 0, 1, false)

	fp.input.SetInputCapture(fp.handleKey)
	fp.input.SetChangedFunc(func(text string) {
		fp.refresh(text)
	})

	return fp
}

// SetOnSelect sets the callback for when a file is selected.
func (fp *FilePicker) SetOnSelect(fn func(path string)) {
	fp.onSelect = fn
}

// SetOnClose sets the callback for Esc / cancel.
func (fp *FilePicker) SetOnClose(fn func()) {
	fp.onClose = fn
}

// Reset clears and pre-fills with startDir.
func (fp *FilePicker) Reset(startDir string) {
	if startDir == "" {
		startDir, _ = os.Getwd()
	}
	if !strings.HasSuffix(startDir, string(os.PathSeparator)) {
		startDir += string(os.PathSeparator)
	}
	fp.input.SetText(startDir)
	fp.refresh(startDir)
}

// Focus always focuses the input field.
func (fp *FilePicker) Focus(delegate func(p tview.Primitive)) {
	delegate(fp.input)
}

// handleKey handles keys on the input field.
func (fp *FilePicker) handleKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		if fp.onClose != nil {
			fp.onClose()
		}
		return nil

	case tcell.KeyTab:
		fp.tabComplete()
		return nil

	case tcell.KeyEnter:
		text := strings.TrimSpace(fp.input.GetText())
		if text == "" {
			return nil
		}
		text = expandHome(text)
		if fp.onSelect != nil {
			fp.onSelect(text)
		}
		return nil
	}

	return event
}

// tabComplete performs shell-style tab completion.
func (fp *FilePicker) tabComplete() {
	text := expandHome(fp.input.GetText())
	if text == "" || strings.HasSuffix(text, string(os.PathSeparator)) {
		return
	}

	dir := dirOf(text)
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
		fp.input.SetText(completed)
		fp.refresh(completed)
		return
	}

	// Multiple matches — complete to longest common prefix.
	lcp := matches[0]
	for _, m := range matches[1:] {
		lcp = commonPrefix(lcp, m)
	}
	if len(lcp) > len(prefix) {
		fp.input.SetText(filepath.Join(dir, lcp))
		fp.refresh(filepath.Join(dir, lcp))
	}
}

// refresh updates the file listing based on current input.
func (fp *FilePicker) refresh(text string) {
	fp.list.Clear()
	text = expandHome(text)
	if text == "" {
		return
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
		return
	}

	// Sort: directories first, then files.
	sort.Slice(entries, func(i, j int) bool {
		di, dj := entries[i].IsDir(), entries[j].IsDir()
		if di != dj {
			return di
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	var b strings.Builder
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			continue
		}
		if e.IsDir() {
			b.WriteString("[::d]" + tview.Escape(name) + "/[-:-:-]\n")
		} else {
			b.WriteString(tview.Escape(name) + "\n")
		}
	}
	fp.list.SetText(b.String())
	fp.list.ScrollToBeginning()
}

// --- helpers ---

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// dirOf returns the directory portion of a path.
func dirOf(text string) string {
	if strings.HasSuffix(text, string(os.PathSeparator)) {
		return text
	}
	return filepath.Dir(text) + string(os.PathSeparator)
}

// commonPrefix returns the longest common prefix of two strings.
func commonPrefix(a, b string) string {
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
