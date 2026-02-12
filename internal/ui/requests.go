package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"proxy-tui/internal/model"
	"proxy-tui/internal/viewmodel"
)

// Column width constants (fixed columns - must be fully visible)
const (
	colTimeWidth   = 12 // HH:MM:SS.mmm
	colMethodWidth = 6  // DELETE/TUNNEL is longest
	colHostWidth   = 25 // reasonable host width
	colStatusWidth = 5  // 3 digits or ERR
	colDurWidth    = 10 // e.g., "123ms" or "1.23s"
)

// RequestsPanel shows the list of HTTP requests
type RequestsPanel struct {
	*tview.Table
	viewModel      *viewmodel.ViewModel
	onSelect       func(flow *model.Flow)
	selectedFlowID model.FlowID
	stayOnTop      bool
}

// NewRequestsPanel creates a new requests panel
func NewRequestsPanel(vm *viewmodel.ViewModel) *RequestsPanel {
	table := tview.NewTable()
	table.SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 0) // Fixed header row
	table.SetBorder(true)
	table.SetTitle(" Requests ")
	table.SetTitleAlign(tview.AlignLeft)

	rp := &RequestsPanel{
		Table:     table,
		viewModel: vm,
		stayOnTop: true, // Start with stay on top enabled
	}

	// Set up header
	rp.setHeader()

	// Set selection changed handler
	table.SetSelectionChangedFunc(func(row, column int) {
		if row > 0 { // Skip header
			// Disable stay on top if user navigates away from top
			if row != 1 && rp.stayOnTop {
				rp.stayOnTop = false
				rp.updateTitle()
			}
			flows := vm.GetFilteredFlows()
			// Reverse index: row 1 = last flow (newest)
			flowIdx := len(flows) - row
			if flowIdx >= 0 && flowIdx < len(flows) {
				rp.selectedFlowID = flows[flowIdx].ID
				vm.SelectFlow(flowIdx)
				if rp.onSelect != nil {
					rp.onSelect(flows[flowIdx])
				}
			}
		}
	})

	return rp
}

// setHeader sets up the table header
func (rp *RequestsPanel) setHeader() {
	headers := []string{"Time", "Mthd", "Host", "Path", "Code", "Dur"}
	widths := []int{colTimeWidth, colMethodWidth, colHostWidth, 0, colStatusWidth, colDurWidth}

	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		if widths[i] > 0 {
			cell.SetMaxWidth(widths[i])
		}
		// Path column (index 3) width is set dynamically in Refresh
		rp.SetCell(0, i, cell)
	}
}

// updateHeaderPathWidth updates the path column header width
func (rp *RequestsPanel) updateHeaderPathWidth(pathMaxWidth int) {
	cell := rp.GetCell(0, 3)
	if cell != nil {
		cell.SetMaxWidth(pathMaxWidth)
	}
}

// SetOnSelect sets the callback for when a flow is selected
func (rp *RequestsPanel) SetOnSelect(fn func(flow *model.Flow)) {
	rp.onSelect = fn
}

// updateTitle updates the panel title with count and stay on top indicator
func (rp *RequestsPanel) updateTitle() {
	flows := rp.viewModel.GetFilteredFlows()
	stayIndicator := ""
	if rp.stayOnTop {
		stayIndicator = " [yellow]⬆[-]"
	}
	rp.SetTitle(fmt.Sprintf(" Requests [%d/%d]%s ", len(flows), rp.viewModel.GetFlowCount(), stayIndicator))
}

// Refresh updates the table with current flows
func (rp *RequestsPanel) Refresh() {
	fmt.Println("Refresh")
	flows := rp.viewModel.GetFilteredFlows()

	// Clear existing rows (except header)
	rowCount := rp.GetRowCount()
	for i := rowCount - 1; i > 0; i-- {
		rp.RemoveRow(i)
	}

	// Calculate path width once for all rows
	_, _, width, _ := rp.GetInnerRect()
	fmt.Println("Width:", width)
	if width == 0 {
		width = 120 // default fallback before first render
	}
	fixedWidth := colTimeWidth + colMethodWidth + colHostWidth + colStatusWidth + colDurWidth
	pathMaxWidth := width - fixedWidth
	if pathMaxWidth < 10 {
		pathMaxWidth = 10
	}

	// Update header path column width
	rp.updateHeaderPathWidth(pathMaxWidth)

	// Add flow rows in reverse order (newest first)
	for i := len(flows) - 1; i >= 0; i-- {
		row := len(flows) - i // row 1 = newest (last in slice)
		rp.addFlowRow(row, flows[i], pathMaxWidth)
	}

	// Update title with count and stay on top indicator
	rp.updateTitle()

	if len(flows) == 0 {
		return
	}

	// If stay on top is enabled, always select row 1
	if rp.stayOnTop {
		rp.Select(1, 0)
		rp.selectedFlowID = flows[len(flows)-1].ID
		return
	}

	// Restore selection by flow ID
	selectedRow := 0
	if rp.selectedFlowID > 0 {
		for i, flow := range flows {
			if flow.ID == rp.selectedFlowID {
				selectedRow = len(flows) - i // convert to row number
				break
			}
		}
	}

	if selectedRow > 0 && selectedRow <= len(flows) {
		rp.Select(selectedRow, 0)
	} else {
		rp.Select(1, 0)
		rp.selectedFlowID = flows[len(flows)-1].ID
	}
}

// addFlowRow adds a single flow row to the table
func (rp *RequestsPanel) addFlowRow(row int, flow *model.Flow, pathMaxWidth int) {
	method, host, path, status, duration := rp.viewModel.FormatFlowSummary(flow)

	// For tunneled flows, override display
	if flow.Tunneled {
		method = "TUNNEL"
		status = "—"
		path = "(encrypted)"
	}

	// Timestamp column (HH:MM:SS.mmm) - fixed width
	timestamp := flow.StartTime.Format("15:04:05.000")
	rp.SetCell(row, 0, tview.NewTableCell(timestamp).
		SetTextColor(tcell.ColorGray).
		SetMaxWidth(colTimeWidth))

	// Method column - fixed width
	methodColor := tcell.ColorWhite
	if flow.Tunneled {
		methodColor = tcell.ColorDarkGray
	} else {
		switch method {
		case "GET":
			methodColor = tcell.ColorGreen
		case "POST":
			methodColor = tcell.ColorBlue
		case "PUT":
			methodColor = tcell.ColorYellow
		case "DELETE":
			methodColor = tcell.ColorRed
		case "PATCH":
			methodColor = tcell.ColorOrange
		}
	}
	rp.SetCell(row, 1, tview.NewTableCell(method).
		SetTextColor(methodColor).
		SetMaxWidth(colMethodWidth))

	// Host column - fixed width
	hostColor := tcell.ColorWhite
	if flow.Tunneled {
		hostColor = tcell.ColorDarkGray
	}
	rp.SetCell(row, 2, tview.NewTableCell(host).
		SetTextColor(hostColor).
		SetMaxWidth(colHostWidth))

	// Path column - dynamic width based on available space
	pathColor := tcell.ColorGray
	if flow.Tunneled {
		pathColor = tcell.ColorDarkGray
	}
	// Set MaxWidth on path to prevent horizontal scroll, let tview clip visually
	rp.SetCell(row, 3, tview.NewTableCell(path).
		SetTextColor(pathColor).
		SetMaxWidth(pathMaxWidth))

	// Status column - fixed width, with alert indicator
	statusColor := tcell.ColorWhite
	if flow.Tunneled {
		statusColor = tcell.ColorDarkGray
	} else if flow.Response != nil {
		switch {
		case flow.Response.StatusCode >= 200 && flow.Response.StatusCode < 300:
			statusColor = tcell.ColorGreen
		case flow.Response.StatusCode >= 300 && flow.Response.StatusCode < 400:
			statusColor = tcell.ColorYellow
		case flow.Response.StatusCode >= 400 && flow.Response.StatusCode < 500:
			statusColor = tcell.ColorOrange
		case flow.Response.StatusCode >= 500:
			statusColor = tcell.ColorRed
		}
	} else if flow.Error != nil {
		statusColor = tcell.ColorRed
	}
	if rp.viewModel.CheckAlerts(flow) {
		status = "!" + status
	}
	rp.SetCell(row, 4, tview.NewTableCell(status).
		SetTextColor(statusColor).
		SetMaxWidth(colStatusWidth))

	// Duration column - fixed width
	durationColor := tcell.ColorGray
	if flow.Tunneled {
		durationColor = tcell.ColorDarkGray
	}
	rp.SetCell(row, 5, tview.NewTableCell(duration).
		SetTextColor(durationColor).
		SetMaxWidth(colDurWidth))
}

// MoveUp moves selection up
func (rp *RequestsPanel) MoveUp() {
	row, _ := rp.GetSelection()
	if row > 1 {
		rp.Select(row-1, 0)
	}
}

// MoveDown moves selection down
func (rp *RequestsPanel) MoveDown() {
	row, _ := rp.GetSelection()
	if row < rp.GetRowCount()-1 {
		rp.Select(row+1, 0)
	}
}

// GoToTop goes to the top and enables stay on top
func (rp *RequestsPanel) GoToTop() {
	rp.stayOnTop = true
	if rp.GetRowCount() > 1 {
		rp.Select(1, 0)
	}
	rp.Refresh()
}

// GoToBottom goes to the bottom of the list
func (rp *RequestsPanel) GoToBottom() {
	rp.stayOnTop = false
	rp.updateTitle()
	rowCount := rp.GetRowCount()
	if rowCount > 1 {
		rp.Select(rowCount-1, 0)
	}
}

// SetFocused updates the border style based on focus state
func (rp *RequestsPanel) SetFocused(focused bool) {
	if focused {
		rp.SetBorderColor(tcell.ColorWhite)
	} else {
		rp.SetBorderColor(tcell.ColorGray)
	}
}

// truncate truncates a string to maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
