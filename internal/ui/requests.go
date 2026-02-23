package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"proxy-tui/internal/model"
	"proxy-tui/internal/viewmodel"
)

// Column widths (fixed columns)
const (
	colTimeWidth   = 12 // HH:MM:SS.mmm
	colMethodWidth = 8  // DELETE/TUNNEL is longest
	colHostWidth   = 25
	colStatusWidth = 5
	colDurWidth    = 10
	colStarWidth   = 1
	colMapWidth    = 1
	colPadding     = 7 // spaces between columns
)

// RequestListModel manages the requests table rendering.
type RequestListModel struct {
	vm             *viewmodel.ViewModel
	cursor         int // index into displayed rows (0-based, rows are newest-first)
	offset         int // scroll offset
	width, height  int
	selectedFlowID model.FlowID
	stayOnTop      bool
}

// NewRequestListModel creates a new request list.
func NewRequestListModel(vm *viewmodel.ViewModel) RequestListModel {
	return RequestListModel{
		vm:        vm,
		stayOnTop: true,
	}
}

func (r *RequestListModel) SetSize(w, h int) {
	r.width = w
	r.height = h
}

// visibleRows returns the number of rows that fit in the viewport (excluding header).
func (r *RequestListModel) visibleRows() int {
	return r.height - 1 // 1 for header
}

// flowCount returns the number of filtered flows.
func (r *RequestListModel) flowCount() int {
	return len(r.vm.GetFilteredFlows())
}

// MoveUp moves cursor up.
func (r *RequestListModel) MoveUp() {
	if r.cursor > 0 {
		r.cursor--
		if r.stayOnTop && r.cursor != 0 {
			r.stayOnTop = false
		}
	}
	r.ensureVisible()
	r.syncSelection()
}

// MoveDown moves cursor down.
func (r *RequestListModel) MoveDown() {
	count := r.flowCount()
	if r.cursor < count-1 {
		r.cursor++
		if r.stayOnTop {
			r.stayOnTop = false
		}
	}
	r.ensureVisible()
	r.syncSelection()
}

// GoToTop enables stay-on-top and moves to first row.
func (r *RequestListModel) GoToTop() {
	r.stayOnTop = true
	r.cursor = 0
	r.offset = 0
	r.syncSelection()
}

// GoToBottom moves to last row.
func (r *RequestListModel) GoToBottom() {
	r.stayOnTop = false
	count := r.flowCount()
	if count > 0 {
		r.cursor = count - 1
	}
	r.ensureVisible()
	r.syncSelection()
}

// OnFlowsUpdated handles a flow update.
func (r *RequestListModel) OnFlowsUpdated() {
	count := r.flowCount()
	if count == 0 {
		r.cursor = 0
		r.offset = 0
		return
	}

	if r.stayOnTop {
		r.cursor = 0
		r.offset = 0
		r.syncSelection()
		return
	}

	// Try to preserve selection by flow ID
	if r.selectedFlowID > 0 {
		flows := r.vm.GetFilteredFlows()
		for i := len(flows) - 1; i >= 0; i-- {
			displayIdx := len(flows) - 1 - i
			if flows[i].ID == r.selectedFlowID {
				r.cursor = displayIdx
				r.ensureVisible()
				r.syncSelection()
				return
			}
		}
	}

	// Selection not found, clamp
	if r.cursor >= count {
		r.cursor = count - 1
	}
	r.ensureVisible()
	r.syncSelection()
}

func (r *RequestListModel) ensureVisible() {
	vis := r.visibleRows()
	if vis <= 0 {
		return
	}
	if r.cursor < r.offset {
		r.offset = r.cursor
	}
	if r.cursor >= r.offset+vis {
		r.offset = r.cursor - vis + 1
	}
}

func (r *RequestListModel) syncSelection() {
	flows := r.vm.GetFilteredFlows()
	count := len(flows)
	if count == 0 {
		return
	}
	// Display rows are newest-first: display index 0 = flows[count-1]
	flowIdx := count - 1 - r.cursor
	if flowIdx >= 0 && flowIdx < count {
		r.selectedFlowID = flows[flowIdx].ID
		r.vm.SelectFlow(flowIdx)
	}
}

// Title returns the panel title string.
func (r *RequestListModel) Title() string {
	flows := r.vm.GetFilteredFlows()
	total := r.vm.GetFlowCount()
	title := fmt.Sprintf("Requests %d/%d", len(flows), total)
	if r.vm.IsPaused() {
		title += " PAUSED"
	}
	if r.stayOnTop {
		title += " ↑"
	}
	return title
}

// View renders the request table.
func (r *RequestListModel) View() string {
	if r.width <= 0 || r.height <= 0 {
		return ""
	}

	flows := r.vm.GetFilteredFlows()
	count := len(flows)

	// Calculate path column width (account for cursor indicator + left/right padding)
	pad := " "
	fixedWidth := 2 + 2 + colTimeWidth + colMethodWidth + colHostWidth + colStatusWidth + colDurWidth + colStarWidth + colMapWidth + colPadding
	pathWidth := r.width - fixedWidth
	if pathWidth < 10 {
		pathWidth = 10
	}

	var sb strings.Builder

	// Header row with subtle separator
	header := renderHeader(pathWidth)
	sb.WriteString(pad + header)
	sb.WriteByte('\n')

	// Rows
	vis := r.visibleRows()
	if vis <= 0 {
		return sb.String()
	}

	if count == 0 {
		// Empty state
		emptyMsg := lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Render("  Waiting for requests...")
		// Center it vertically
		for i := 0; i < vis/2-1; i++ {
			sb.WriteString(strings.Repeat(" ", r.width))
			sb.WriteByte('\n')
		}
		sb.WriteString(emptyMsg)
		return sb.String()
	}

	for i := 0; i < vis; i++ {
		rowIdx := r.offset + i
		if rowIdx >= count {
			sb.WriteString(strings.Repeat(" ", r.width))
			if i < vis-1 {
				sb.WriteByte('\n')
			}
			continue
		}

		// Display rows are newest-first: row 0 = flows[count-1]
		flowIdx := count - 1 - rowIdx
		flow := flows[flowIdx]
		selected := rowIdx == r.cursor

		method, host, path, status, duration := r.vm.FormatFlowSummary(flow)
		if flow.Tunneled {
			method = "TUNNEL"
			status = "-"
			path = "(encrypted)"
		}

		timestamp := flow.StartTime.Format("15:04:05.000")

		// Alert indicator
		if r.vm.CheckAlerts(flow) {
			status = "!" + status
		}

		// Star indicator
		star := ""
		if r.vm.IsStarred(flow) {
			star = "*"
		}

		// Mapped indicator
		mapped := ""
		switch flow.Mapped {
		case "local":
			mapped = "L"
		case "remote":
			mapped = "R"
		}

		row := renderRow(timestamp, method, host, path, status, duration, star, mapped, pathWidth, flow, selected)
		sb.WriteString(pad + row)
		if i < vis-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func renderHeader(pathWidth int) string {
	cursor := "  " // cursor column placeholder
	timestamp := padOrTrunc("Time", colTimeWidth)
	method := padOrTrunc("Method", colMethodWidth)
	host := padOrTrunc("Host", colHostWidth)
	path := padOrTrunc("Path", pathWidth)
	status := padOrTrunc("Code", colStatusWidth)
	dur := padOrTrunc("Dur", colDurWidth)
	star := padOrTrunc("*", colStarWidth)
	mapped := padOrTrunc("M", colMapWidth)

	line := cursor + timestamp + " " + method + " " + host + " " + path + " " + status + " " + dur + " " + star + " " + mapped
	return headerStyle.Render(line)
}

func renderRow(timestamp, method, host, path, status, dur, star, mapped string, pathWidth int, flow *model.Flow, selected bool) string {
	// Truncate columns to fit
	timestamp = padOrTrunc(timestamp, colTimeWidth)
	method = padOrTrunc(method, colMethodWidth)
	host = padOrTrunc(host, colHostWidth)
	path = padOrTrunc(path, pathWidth)
	status = padOrTrunc(status, colStatusWidth)
	dur = padOrTrunc(dur, colDurWidth)
	star = padOrTrunc(star, colStarWidth)
	mapped = padOrTrunc(mapped, colMapWidth)

	// Cursor indicator — accent bar for selected row
	cursor := "  "
	if selected {
		cursor = selectedIndicator().Render("▌ ")
	}

	tunneled := flow != nil && flow.Tunneled

	timeStr := lipgloss.NewStyle().Foreground(colorMuted).Render(timestamp)
	methodStr := methodStyle(strings.TrimSpace(method)).Render(method)

	hostColor := colorWhite
	if tunneled {
		hostColor = colorMuted
	}
	hostStr := lipgloss.NewStyle().Foreground(hostColor).Render(host)

	pathColor := colorSubtle
	if tunneled {
		pathColor = colorMuted
	}
	pathStr := lipgloss.NewStyle().Foreground(pathColor).Render(path)

	var statusStr string
	if flow != nil && flow.Response != nil {
		statusStr = statusStyle(flow.Response.StatusCode).Render(status)
	} else if flow != nil && flow.Error != nil {
		statusStr = lipgloss.NewStyle().Foreground(colorRed).Render(status)
	} else {
		statusStr = lipgloss.NewStyle().Foreground(colorMuted).Render(status)
	}

	durStr := lipgloss.NewStyle().Foreground(colorMuted).Render(dur)
	starStr := lipgloss.NewStyle().Foreground(colorStar).Render(star)

	var mappedStr string
	switch {
	case flow != nil && flow.Mapped == "local":
		mappedStr = mappedLocalBadge.Render(mapped)
	case flow != nil && flow.Mapped == "remote":
		mappedStr = mappedRemoteBadge.Render(mapped)
	default:
		mappedStr = lipgloss.NewStyle().Foreground(colorMuted).Render(mapped)
	}

	line := cursor + timeStr + " " + methodStr + " " + hostStr + " " + pathStr + " " + statusStr + " " + durStr + " " + starStr + " " + mappedStr

	return line
}

func padOrTrunc(s string, width int) string {
	if len(s) > width {
		if width <= 3 {
			return s[:width]
		}
		return s[:width-3] + "..."
	}
	return s + strings.Repeat(" ", width-len(s))
}
