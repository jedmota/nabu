package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"proxy-tui/internal/model"
	"proxy-tui/internal/viewmodel"
)

// DetailPanel shows the details of a selected flow
type DetailPanel struct {
	*tview.TextView
	viewModel   *viewmodel.ViewModel
	currentFlow *model.Flow
	rawMode     bool
}

// NewDetailPanel creates a new detail panel
func NewDetailPanel(vm *viewmodel.ViewModel) *DetailPanel {
	textView := tview.NewTextView()
	textView.SetDynamicColors(true)
	textView.SetScrollable(true)
	textView.SetWrap(true)
	textView.SetBorder(true)
	textView.SetTitle(" Detail ")
	textView.SetTitleAlign(tview.AlignLeft)

	dp := &DetailPanel{
		TextView:  textView,
		viewModel: vm,
		rawMode:   false,
	}

	dp.SetText("[gray]Select a request to view details[-]")

	return dp
}

// SetFlow updates the detail view with a new flow
func (dp *DetailPanel) SetFlow(flow *model.Flow) {
	dp.currentFlow = flow
	dp.refresh()
}

// refresh updates the display
func (dp *DetailPanel) refresh() {
	if dp.currentFlow == nil {
		dp.SetText("[gray]Select a request to view details[-]")
		return
	}

	content := dp.viewModel.FormatFlowDetail(dp.currentFlow, dp.rawMode)
	dp.SetText(content)
	dp.ScrollToBeginning()

	// Update title
	title := " Detail "
	if dp.rawMode {
		title = " Detail [RAW] "
	}
	dp.SetTitle(title)
}

// ToggleRawMode toggles between raw and pretty display
func (dp *DetailPanel) ToggleRawMode() {
	dp.rawMode = !dp.rawMode
	dp.refresh()
}

// IsRawMode returns whether raw mode is enabled
func (dp *DetailPanel) IsRawMode() bool {
	return dp.rawMode
}

// Clear clears the detail view
func (dp *DetailPanel) Clear() {
	dp.currentFlow = nil
	dp.SetText("[gray]Select a request to view details[-]")
	dp.SetTitle(" Detail ")
}

// ScrollUp scrolls up by one line
func (dp *DetailPanel) ScrollUp() {
	row, col := dp.GetScrollOffset()
	if row > 0 {
		dp.ScrollTo(row-1, col)
	}
}

// ScrollDown scrolls down by one line
func (dp *DetailPanel) ScrollDown() {
	row, col := dp.GetScrollOffset()
	dp.ScrollTo(row+1, col)
}

// PageUp scrolls up by page
func (dp *DetailPanel) PageUp() {
	row, col := dp.GetScrollOffset()
	_, _, _, height := dp.GetInnerRect()
	newRow := row - height
	if newRow < 0 {
		newRow = 0
	}
	dp.ScrollTo(newRow, col)
}

// PageDown scrolls down by page
func (dp *DetailPanel) PageDown() {
	row, col := dp.GetScrollOffset()
	_, _, _, height := dp.GetInnerRect()
	dp.ScrollTo(row+height, col)
}

// InputHandler returns an input handler for keyboard navigation
func (dp *DetailPanel) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		switch event.Key() {
		case tcell.KeyUp:
			dp.ScrollUp()
		case tcell.KeyDown:
			dp.ScrollDown()
		case tcell.KeyPgUp:
			dp.PageUp()
		case tcell.KeyPgDn:
			dp.PageDown()
		case tcell.KeyRune:
			switch event.Rune() {
			case 'j':
				dp.ScrollDown()
			case 'k':
				dp.ScrollUp()
			case 'T':
				dp.ToggleRawMode()
			}
		}
	}
}
