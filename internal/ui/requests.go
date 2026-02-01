package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"proxy-tui/internal/model"
	"proxy-tui/internal/viewmodel"
)

// RequestsPanel shows the list of HTTP requests
type RequestsPanel struct {
	*tview.Table
	viewModel     *viewmodel.ViewModel
	onSelect      func(flow *model.Flow)
	selectedRow   int
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
	}

	// Set up header
	rp.setHeader()

	// Set selection changed handler
	table.SetSelectionChangedFunc(func(row, column int) {
		if row > 0 { // Skip header
			rp.selectedRow = row
			flows := vm.GetFilteredFlows()
			if row-1 < len(flows) {
				vm.SelectFlow(row - 1)
				if rp.onSelect != nil {
					rp.onSelect(flows[row-1])
				}
			}
		}
	})

	return rp
}

// setHeader sets up the table header
func (rp *RequestsPanel) setHeader() {
	headers := []string{"#", "Method", "Host", "Path", "Status", "Time"}
	widths := []int{6, 8, 30, 40, 8, 10}

	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetMaxWidth(widths[i])
		rp.SetCell(0, i, cell)
	}
}

// SetOnSelect sets the callback for when a flow is selected
func (rp *RequestsPanel) SetOnSelect(fn func(flow *model.Flow)) {
	rp.onSelect = fn
}

// Refresh updates the table with current flows
func (rp *RequestsPanel) Refresh() {
	flows := rp.viewModel.GetFilteredFlows()

	// Clear existing rows (except header)
	rowCount := rp.GetRowCount()
	for i := rowCount - 1; i > 0; i-- {
		rp.RemoveRow(i)
	}

	// Add flow rows
	for i, flow := range flows {
		rp.addFlowRow(i+1, flow)
	}

	// Update title with count
	rp.SetTitle(fmt.Sprintf(" Requests [%d/%d] ", len(flows), rp.viewModel.GetFlowCount()))

	// Restore selection
	if rp.selectedRow > 0 && rp.selectedRow <= len(flows) {
		rp.Select(rp.selectedRow, 0)
	} else if len(flows) > 0 {
		rp.Select(1, 0)
		rp.selectedRow = 1
	}
}

// addFlowRow adds a single flow row to the table
func (rp *RequestsPanel) addFlowRow(row int, flow *model.Flow) {
	method, host, path, status, duration := rp.viewModel.FormatFlowSummary(flow)

	// ID column
	rp.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", flow.ID)).
		SetTextColor(tcell.ColorGray).
		SetMaxWidth(6))

	// Method column - color by method
	methodColor := tcell.ColorWhite
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
	rp.SetCell(row, 1, tview.NewTableCell(method).
		SetTextColor(methodColor).
		SetMaxWidth(8))

	// Host column
	rp.SetCell(row, 2, tview.NewTableCell(host).
		SetTextColor(tcell.ColorWhite).
		SetMaxWidth(30))

	// Path column
	rp.SetCell(row, 3, tview.NewTableCell(truncate(path, 40)).
		SetTextColor(tcell.ColorGray).
		SetMaxWidth(40))

	// Status column - color by status code
	statusColor := tcell.ColorWhite
	if flow.Response != nil {
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
	rp.SetCell(row, 4, tview.NewTableCell(status).
		SetTextColor(statusColor).
		SetMaxWidth(8))

	// Duration column
	rp.SetCell(row, 5, tview.NewTableCell(duration).
		SetTextColor(tcell.ColorGray).
		SetMaxWidth(10))
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
