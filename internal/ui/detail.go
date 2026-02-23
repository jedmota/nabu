package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"proxy-tui/internal/model"
	"proxy-tui/internal/viewmodel"
)

// tviewColorRe matches tview color tags like [green], [cyan], [gray], [red], [blue], [yellow], [-], [::b], [::d], [-::-]
var tviewColorRe = regexp.MustCompile(`\[(?:[a-zA-Z]+|[-]|::[a-z]|[-]::[-])\]`)

// stripTviewColors removes tview color tags from text.
func stripTviewColors(s string) string {
	return tviewColorRe.ReplaceAllString(s, "")
}

// DetailViewModel wraps a viewport for scrollable flow detail.
type DetailViewModel struct {
	vm          *viewmodel.ViewModel
	viewport    viewport.Model
	currentFlow *model.Flow
	rawMode     bool
	width       int
	height      int
	ready       bool
}

// NewDetailViewModel creates a new detail view.
func NewDetailViewModel(vm *viewmodel.ViewModel) DetailViewModel {
	return DetailViewModel{
		vm: vm,
	}
}

func (d *DetailViewModel) SetSize(w, h int) {
	d.width = w
	d.height = h
	if !d.ready {
		d.viewport = viewport.New(w, h)
		d.ready = true
	} else {
		d.viewport.Width = w
		d.viewport.Height = h
	}
}

// SetFlow updates the displayed flow.
func (d *DetailViewModel) SetFlow(flow *model.Flow) {
	d.currentFlow = flow
	d.refresh()
}

// Clear clears the detail view.
func (d *DetailViewModel) Clear() {
	d.currentFlow = nil
	d.refresh()
}

// ToggleRawMode toggles raw/pretty display.
func (d *DetailViewModel) ToggleRawMode() {
	d.rawMode = !d.rawMode
	d.refresh()
}

func (d *DetailViewModel) refresh() {
	if !d.ready {
		return
	}
	if d.currentFlow == nil {
		emptyStyle := lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)
		d.viewport.SetContent(emptyStyle.Render("Select a request to view details"))
		d.viewport.GotoTop()
		return
	}

	// Get formatted detail, wrap to viewport width, then apply lipgloss styles
	content := d.vm.FormatFlowDetail(d.currentFlow, d.rawMode)
	wrapped := d.wrapContent(content)
	styled := d.applyStyles(wrapped)
	d.viewport.SetContent(styled)
	d.viewport.GotoTop()
}

// Title returns the panel title.
func (d *DetailViewModel) Title() string {
	if d.rawMode {
		return "Detail RAW"
	}
	return "Detail"
}

// View renders the detail viewport.
func (d *DetailViewModel) View() string {
	if !d.ready {
		return ""
	}
	return d.viewport.View()
}

// ScrollUp scrolls the viewport up one line.
func (d *DetailViewModel) ScrollUp() {
	d.viewport.LineUp(1)
}

// ScrollDown scrolls the viewport down one line.
func (d *DetailViewModel) ScrollDown() {
	d.viewport.LineDown(1)
}

// PageUp scrolls the viewport up one page.
func (d *DetailViewModel) PageUp() {
	d.viewport.HalfViewUp()
}

// PageDown scrolls the viewport down one page.
func (d *DetailViewModel) PageDown() {
	d.viewport.HalfViewDown()
}

// wrapContent wraps lines that exceed the viewport width.
// It measures display width excluding tview color tags.
func (d *DetailViewModel) wrapContent(text string) string {
	if d.width <= 0 {
		return text
	}
	maxW := d.width - 1 // leave a small margin
	if maxW < 10 {
		maxW = 10
	}

	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		plain := stripTviewColors(line)
		if len(plain) <= maxW {
			result = append(result, line)
			continue
		}
		// Wrap long lines: work on plain text, then re-insert at break points
		// Simple character-level wrap on the raw line, tracking display width
		wrapped := wrapLine(line, maxW)
		result = append(result, wrapped...)
	}
	return strings.Join(result, "\n")
}

// wrapLine wraps a single line (which may contain tview color tags) at maxW display chars.
func wrapLine(line string, maxW int) []string {
	var result []string
	var current strings.Builder
	displayW := 0
	runes := []rune(line)
	i := 0

	for i < len(runes) {
		// Check if we're at a tview color tag
		if runes[i] == '[' {
			// Find closing ]
			j := i + 1
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			if j < len(runes) {
				tag := string(runes[i : j+1])
				if tviewColorRe.MatchString(tag) {
					// Zero-width tag, always include
					current.WriteString(tag)
					i = j + 1
					continue
				}
			}
		}

		if displayW >= maxW {
			result = append(result, current.String())
			current.Reset()
			displayW = 0
		}
		current.WriteRune(runes[i])
		displayW++
		i++
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}
	if len(result) == 0 {
		result = []string{""}
	}
	return result
}

// applyStyles converts tview color-tagged text to lipgloss-styled text.
func (d *DetailViewModel) applyStyles(text string) string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		result = append(result, d.styleLine(line))
	}

	return strings.Join(result, "\n")
}

func (d *DetailViewModel) styleLine(line string) string {
	matches := tviewColorRe.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return line
	}

	var sb strings.Builder
	currentStyle := lipgloss.NewStyle()
	lastEnd := 0

	for _, match := range matches {
		if match[0] > lastEnd {
			sb.WriteString(currentStyle.Render(line[lastEnd:match[0]]))
		}

		tag := line[match[0]+1 : match[1]-1] // strip [ and ]
		switch tag {
		case "-", "-::-":
			currentStyle = lipgloss.NewStyle()
		case "green":
			currentStyle = lipgloss.NewStyle().Foreground(colorGreen)
		case "cyan":
			currentStyle = lipgloss.NewStyle().Foreground(colorCyan)
		case "blue":
			currentStyle = lipgloss.NewStyle().Foreground(colorBlue)
		case "yellow":
			currentStyle = lipgloss.NewStyle().Foreground(accentColor()).Bold(true)
		case "red":
			currentStyle = lipgloss.NewStyle().Foreground(colorRed)
		case "gray":
			currentStyle = lipgloss.NewStyle().Foreground(colorSubtle)
		case "::b":
			currentStyle = currentStyle.Bold(true)
		case "::d":
			currentStyle = currentStyle.Faint(true)
		default:
			currentStyle = lipgloss.NewStyle().Foreground(colorWhite)
		}

		lastEnd = match[1]
	}

	if lastEnd < len(line) {
		sb.WriteString(currentStyle.Render(line[lastEnd:]))
	}

	return sb.String()
}

// formatAlertRule formats an alert rule for display.
func formatAlertRule(rule model.AlertRule) string {
	switch rule.Type {
	case model.AlertStatusCode:
		return fmt.Sprintf("Status %dxx", rule.Value/100)
	case model.AlertLatency:
		if rule.Value >= 1000 {
			return fmt.Sprintf("Latency > %.1fs", float64(rule.Value)/1000)
		}
		return fmt.Sprintf("Latency > %dms", rule.Value)
	default:
		return string(rule.Type)
	}
}
