package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"nabu/internal/config"
	"nabu/internal/model"
	"nabu/internal/util"
	"nabu/internal/viewmodel"
)

// Model is the root Bubble Tea model for the proxy TUI.
type Model struct {
	vm            *viewmodel.ViewModel
	width, height int
	keys          KeyMap

	// Panels
	focusedPanel int // 0=requests, 1=detail
	requestList  RequestListModel
	detailView   DetailViewModel
	expanded     bool

	// Modals
	activeModal        ModalType
	searchInput        textinput.Model
	whitelistInput     textinput.Model
	whitelistEditOld   string // non-empty = edit mode
	whitelistMgr       WhitelistManagerState
	mapLocalMgr        MapLocalManagerState
	mapLocalForm       ModalFormModel
	mapLocalPatternInput textinput.Model
	mapRemoteMgr       MapRemoteManagerState
	mapRemoteForm      ModalFormModel
	mapRemoteEditID    int // 0 = add mode, >0 = edit rule ID
	alertMgr           AlertManagerState
	filePicker         FilePickerState

	// Status
	statusMsg     string
	filterType    model.FilterType
	customPattern string
	address       string

	// Help
	showHelp bool

	// Chord tracking
	lastKeyRune  rune
	lastKeyTime  time.Time
	pendingCount int // vim-style count prefix for motion keys
}

// NewModel creates a new root model.
func NewModel(vm *viewmodel.ViewModel) *Model {
	m := &Model{
		vm:          vm,
		keys:        DefaultKeyMap(),
		requestList: NewRequestListModel(vm),
		detailView:  NewDetailViewModel(vm),
		filterType:  model.FilterAll,
	}

	// Set proxy address
	localIP := getLocalIP()
	m.address = fmt.Sprintf("%s:%d", localIP, vm.Port())
	if vm.IsSecondary() {
		m.address = "IPC " + m.address
	}

	return m
}

// Init returns the initial command.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		ListenForUpdates(m.vm.Updates()),
	)
}

// Update handles all messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case FlowsUpdatedMsg:
		prevID := m.requestList.selectedFlowID
		m.requestList.OnFlowsUpdated()
		// Only refresh detail when the selection actually changed (stayOnTop follows newest)
		// or when there was no previous selection
		if flow := m.vm.GetSelectedFlow(); flow != nil {
			if m.requestList.IsStayOnTop() || m.requestList.selectedFlowID != prevID {
				m.detailView.SetFlow(flow)
			}
		}
		return m, ListenForUpdates(m.vm.Updates())

	case StatusMsg:
		m.statusMsg = msg.Text
		return m, nil

	case ClearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case EditorFinishedMsg:
		// Returned from editor, refresh
		return m, nil

	case ReplayResultMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Replay failed: %s", msg.Err)
		} else {
			m.statusMsg = "Request replayed"
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey routes key messages to the appropriate handler.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Help overlay intercepts all keys
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	// Active modal intercepts all keys
	if m.activeModal != ModalNone {
		return m.handleModalKey(msg)
	}

	key := msg.String()

	// Handle gg chord
	if key == "g" {
		if m.lastKeyRune == 'g' && time.Since(m.lastKeyTime) < 500*time.Millisecond {
			m.lastKeyRune = 0
			if m.focusedPanel == 0 {
				m.requestList.GoToTop()
			}
			return m, nil
		}
		m.lastKeyRune = 'g'
		m.lastKeyTime = time.Now()
		return m, nil
	}
	m.lastKeyRune = 0

	// Accumulate digit keys as count prefix (1-9 start, 0-9 continue)
	if key >= "1" && key <= "9" && m.pendingCount == 0 {
		m.pendingCount = int(key[0] - '0')
		return m, nil
	}
	if key >= "0" && key <= "9" && m.pendingCount > 0 {
		m.pendingCount = m.pendingCount*10 + int(key[0]-'0')
		if m.pendingCount > 999 {
			m.pendingCount = 999
		}
		return m, nil
	}

	// Consume pending count for motion keys, reset for anything else
	count := m.pendingCount
	m.pendingCount = 0

	// Handle G (go to bottom)
	if key == "G" {
		if m.focusedPanel == 0 {
			m.requestList.GoToBottom()
		}
		return m, nil
	}

	// Global keys
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		m.focusedPanel = (m.focusedPanel + 1) % 2
		return m, nil
	case "H":
		m.expanded = !m.expanded
		return m, nil
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	}

	// Context-dependent keys
	switch key {
	case "j", "down":
		n := max(count, 1)
		if m.focusedPanel == 0 {
			for range n {
				m.requestList.MoveDown()
			}
			if flow := m.vm.GetSelectedFlow(); flow != nil {
				m.detailView.SetFlow(flow)
			}
		} else {
			for range n {
				m.detailView.ScrollDown()
			}
		}
		return m, nil

	case "k", "up":
		n := max(count, 1)
		if m.focusedPanel == 0 {
			for range n {
				m.requestList.MoveUp()
			}
			if flow := m.vm.GetSelectedFlow(); flow != nil {
				m.detailView.SetFlow(flow)
			}
		} else {
			for range n {
				m.detailView.ScrollUp()
			}
		}
		return m, nil

	case "pgdown":
		m.detailView.PageDown()
		return m, nil

	case "pgup":
		m.detailView.PageUp()
		return m, nil

	case "1":
		m.filterType = model.FilterAll
		m.vm.SetFilterType(model.FilterAll)
		m.requestList.OnFlowsUpdated()
		return m, nil

	case "2":
		m.filterType = model.FilterWhitelist
		m.vm.SetFilterType(model.FilterWhitelist)
		m.requestList.OnFlowsUpdated()
		return m, nil

	case "3":
		m.filterType = model.FilterStarred
		m.vm.SetFilterType(model.FilterStarred)
		m.requestList.OnFlowsUpdated()
		return m, nil

	case "/":
		m.openSearch()
		return m, nil

	case "c":
		if m.focusedPanel == 0 {
			m.vm.ClearFlows()
			m.detailView.Clear()
			m.requestList.OnFlowsUpdated()
		}
		return m, nil

	case "C":
		m.vm.ClearWhitelist()
		m.requestList.OnFlowsUpdated()
		m.statusMsg = "Whitelist cleared"
		return m, nil

	case "w":
		m.openWhitelistInputFromMain()
		return m, nil

	case "W":
		m.openWhitelistManager()
		return m, nil

	case "l":
		return m, m.handleQuickMapLocal()

	case "L":
		m.openMapLocalManager()
		return m, nil

	case "r":
		m.openMapRemoteFormFromMain()
		return m, nil

	case "R":
		m.openMapRemoteManager()
		return m, nil

	case ".":
		return m, m.replaySelectedFlow()

	case "x":
		m.copyCURL()
		return m, nil

	case "e":
		m.exportHAR(false)
		return m, nil

	case "E":
		m.exportHAR(true)
		return m, nil

	case "i":
		m.openFilePicker()
		return m, nil

	case "p":
		if m.vm.TogglePause() {
			m.statusMsg = "Paused"
		} else {
			m.statusMsg = "Resumed"
		}
		m.requestList.OnFlowsUpdated()
		return m, nil

	case "s":
		flow := m.vm.GetSelectedFlow()
		if flow == nil {
			m.statusMsg = "No request selected"
		} else if m.vm.ToggleStar(flow) {
			m.statusMsg = fmt.Sprintf("Starred (%d total)", m.vm.StarredCount())
		} else {
			m.statusMsg = fmt.Sprintf("Unstarred (%d total)", m.vm.StarredCount())
		}
		m.requestList.OnFlowsUpdated()
		return m, nil

	case "S":
		flows := m.vm.GetFilteredFlows()
		count := m.vm.StarFlows(flows)
		m.statusMsg = fmt.Sprintf("Starred %d flows (%d total)", count, m.vm.StarredCount())
		m.requestList.OnFlowsUpdated()
		return m, nil

	case "a":
		m.openAlertManager()
		return m, nil

	case "T":
		if m.focusedPanel == 1 {
			m.detailView.ToggleRawMode()
		}
		return m, nil

	case "t":
		name := CycleTheme()
		m.statusMsg = fmt.Sprintf("Theme: %s", name)
		return m, nil
	}

	return m, nil
}

// handleModalKey routes key messages to the active modal.
func (m *Model) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.activeModal {
	case ModalSearch:
		cmd = m.updateSearch(msg)
	case ModalWhitelistInput:
		cmd = m.updateWhitelistInput(msg)
	case ModalWhitelistManager:
		cmd = m.updateWhitelistManager(msg)
	case ModalMapLocalPattern:
		cmd = m.updateMapLocalPattern(msg)
	case ModalMapLocalManager:
		cmd = m.updateMapLocalManager(msg)
	case ModalMapLocalForm:
		cmd = m.updateMapLocalForm(msg)
	case ModalMapRemoteManager:
		cmd = m.updateMapRemoteManager(msg)
	case ModalMapRemoteForm:
		cmd = m.updateMapRemoteForm(msg)
	case ModalAlertManager:
		cmd = m.updateAlertManager(msg)
	case ModalImportHAR:
		cmd = m.updateFilePicker(msg)
	}
	return m, cmd
}

// renderActiveModal renders the currently active modal.
func (m *Model) renderActiveModal() string {
	switch m.activeModal {
	case ModalSearch:
		return m.renderSearch()
	case ModalWhitelistInput:
		return m.renderWhitelistInput()
	case ModalWhitelistManager:
		return m.renderWhitelistManager()
	case ModalMapLocalPattern:
		return m.renderMapLocalPattern()
	case ModalMapLocalManager:
		return m.renderMapLocalManager()
	case ModalMapLocalForm:
		return m.renderMapLocalForm()
	case ModalMapRemoteManager:
		return m.renderMapRemoteManager()
	case ModalMapRemoteForm:
		return m.renderMapRemoteForm()
	case ModalAlertManager:
		return m.renderAlertManager()
	case ModalImportHAR:
		return m.renderFilePicker()
	}
	return ""
}

// --- Search ---

func (m *Model) openSearch() {
	m.activeModal = ModalSearch
	m.searchInput = textinput.New()
	m.searchInput.Placeholder = "Filter..."
	m.searchInput.SetValue(m.vm.GetFilter().SearchQuery)
	m.searchInput.Focus()
	m.searchInput.Width = 40
}

func (m *Model) updateSearch(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "enter":
		m.activeModal = ModalNone
		return nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	// Live-update filter
	m.vm.SetSearchQuery(m.searchInput.Value())
	m.vm.SetFilterType(model.FilterCustom)
	m.filterType = model.FilterCustom
	m.customPattern = m.searchInput.Value()
	m.requestList.OnFlowsUpdated()
	return cmd
}

func (m *Model) renderSearch() string {
	return renderInputModal("Filter", m.searchInput, 50)
}

// --- Whitelist input from main (not from manager) ---

func (m *Model) openWhitelistInputFromMain() {
	prefill := ""
	if flow := m.vm.GetSelectedFlow(); flow != nil && flow.Request != nil {
		host := flow.Request.Host
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
		prefill = host
	}
	m.activeModal = ModalWhitelistInput
	m.whitelistInput = textinput.New()
	m.whitelistInput.Placeholder = "*.example.com"
	m.whitelistInput.SetValue(prefill)
	m.whitelistInput.Focus()
	m.whitelistInput.Width = 60
	m.whitelistEditOld = ""
}

// --- Map Remote from main ---

func (m *Model) openMapRemoteFormFromMain() {
	pattern := ""
	if flow := m.vm.GetSelectedFlow(); flow != nil && flow.Request != nil && !flow.Tunneled {
		pattern = flow.Request.URL
	}
	m.openMapRemoteForm(pattern, "")
}

// --- Quick Map Local ---

func (m *Model) handleQuickMapLocal() tea.Cmd {
	flow := m.vm.GetSelectedFlow()
	if flow == nil {
		m.statusMsg = "No request selected"
		return nil
	}
	if flow.Response == nil {
		m.statusMsg = "No response to map"
		return nil
	}
	if flow.Tunneled {
		m.statusMsg = "Cannot map tunneled connections"
		return nil
	}

	pattern := flow.Request.URL
	localPath, err := m.writeJSONCMapping(flow, pattern, []string{
		fmt.Sprintf("  // Mapped from: %s", flow.Request.URL),
		fmt.Sprintf("  // Generated: %s", time.Now().Format(time.RFC3339)),
	})
	if err != nil {
		m.statusMsg = err.Error()
		return nil
	}

	m.vm.AddMapLocalRule(pattern, localPath, flow.Response.StatusCode, flow.Response.Headers.Get("Content-Type"), flow.Request.Method)
	m.statusMsg = fmt.Sprintf("Mapped to: %s", localPath)
	return m.openInEditor(localPath)
}

// createMapLocalWithPattern creates a mapping from the selected flow using a custom pattern.
func (m *Model) createMapLocalWithPattern(pattern string) {
	flow := m.vm.GetSelectedFlow()
	if flow == nil || flow.Response == nil || flow.Tunneled {
		return
	}

	localPath, err := m.writeJSONCMapping(flow, pattern, []string{
		fmt.Sprintf("  // Mapped from: %s", flow.Request.URL),
		fmt.Sprintf("  // Pattern: %s", pattern),
		fmt.Sprintf("  // Generated: %s", time.Now().Format(time.RFC3339)),
	})
	if err != nil {
		m.statusMsg = err.Error()
		return
	}

	m.vm.AddMapLocalRule(pattern, localPath, flow.Response.StatusCode, flow.Response.Headers.Get("Content-Type"), flow.Request.Method)
	m.statusMsg = fmt.Sprintf("Mapped %s -> %s", pattern, localPath)
}

// --- Replay ---

func (m *Model) replaySelectedFlow() tea.Cmd {
	flow := m.vm.GetSelectedFlow()
	if flow == nil {
		m.statusMsg = "No request selected"
		return nil
	}
	if flow.Tunneled {
		m.statusMsg = "Cannot replay tunneled connections"
		return nil
	}

	m.statusMsg = "Replaying request..."
	return func() tea.Msg {
		err := m.vm.ReplayFlow(flow)
		return ReplayResultMsg{Err: err}
	}
}

// --- Copy cURL ---

func (m *Model) copyCURL() {
	flow := m.vm.GetSelectedFlow()
	if flow == nil {
		m.statusMsg = "No request selected"
		return
	}

	curl, err := viewmodel.FormatCURL(flow)
	if err != nil {
		m.statusMsg = err.Error()
		return
	}

	if err := copyToClipboard(curl); err != nil {
		m.statusMsg = fmt.Sprintf("Clipboard error: %s", err)
		return
	}
	m.statusMsg = "cURL command copied to clipboard"
}

// --- Export HAR ---

func (m *Model) exportHAR(all bool) {
	var flows []*model.Flow
	var label string

	if all {
		flows = m.vm.GetFilteredFlows()
		label = fmt.Sprintf("%d flows", len(flows))
	} else {
		flow := m.vm.GetSelectedFlow()
		if flow == nil {
			m.statusMsg = "No request selected"
			return
		}
		flows = []*model.Flow{flow}
		label = "selected flow"
	}

	if len(flows) == 0 {
		m.statusMsg = "No flows to export"
		return
	}

	data, err := viewmodel.FormatHAR(flows)
	if err != nil {
		m.statusMsg = fmt.Sprintf("HAR export failed: %s", err)
		return
	}

	dir := os.TempDir()
	filename := fmt.Sprintf("nabu-%s.har", time.Now().Format("20060102-150405"))
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		m.statusMsg = fmt.Sprintf("Failed to write HAR: %s", err)
		return
	}

	m.statusMsg = fmt.Sprintf("Exported %s to %s", label, path)
}

// --- Import HAR ---

func (m *Model) doImportHAR(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Failed to read file: %s", err)
		return
	}

	flows, err := viewmodel.ParseHAR(data)
	if err != nil {
		m.statusMsg = err.Error()
		return
	}

	if len(flows) == 0 {
		m.statusMsg = "HAR file contains no entries"
		return
	}

	count := m.vm.ImportFlows(flows)
	m.requestList.OnFlowsUpdated()
	m.statusMsg = fmt.Sprintf("Imported %d flows from %s", count, filepath.Base(path))
}

// --- Helpers ---

func (m *Model) openInEditor(filePath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	c := exec.Command(editor, filePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func (m *Model) writeJSONCMapping(flow *model.Flow, filenameSource string, comments []string) (string, error) {
	mappingsDir := filepath.Join(config.GetConfigDir(), "mappings")
	if err := os.MkdirAll(mappingsDir, 0o755); err != nil {
		return "", fmt.Errorf("Failed to create mappings dir: %s", err)
	}

	filename := generateFilename(filenameSource) + ".jsonc"
	localPath := filepath.Join(mappingsDir, filename)

	var buf bytes.Buffer
	buf.WriteString("{\n")
	for _, c := range comments {
		buf.WriteString(c)
		buf.WriteByte('\n')
	}
	buf.WriteString("\n")

	statusText := strings.TrimPrefix(flow.Response.Status, fmt.Sprintf("%d ", flow.Response.StatusCode))
	buf.WriteString(fmt.Sprintf("  \"status\": %d,\n", flow.Response.StatusCode))
	buf.WriteString(fmt.Sprintf("  \"statusText\": %q,\n", statusText))
	buf.WriteString("\n")

	buf.WriteString("  \"headers\": {\n")
	headerCount := 0
	totalHeaders := len(flow.Response.Headers)
	for key, values := range flow.Response.Headers {
		headerCount++
		if strings.EqualFold(key, "Content-Length") {
			totalHeaders--
			continue
		}
		comma := ","
		if headerCount >= totalHeaders {
			comma = ""
		}
		if len(values) == 1 {
			buf.WriteString(fmt.Sprintf("    %q: %q%s\n", key, values[0], comma))
		} else {
			for i, v := range values {
				c := ","
				if headerCount >= totalHeaders && i == len(values)-1 {
					c = ""
				}
				buf.WriteString(fmt.Sprintf("    %q: %q%s\n", key, v, c))
			}
		}
	}
	buf.WriteString("  },\n\n")

	buf.WriteString("  // Response body - edit below\n")
	body := flow.Response.Body
	contentType := flow.Response.Headers.Get("Content-Type")

	if strings.Contains(contentType, "json") || util.IsJSON(body) {
		var jsonObj interface{}
		if err := json.Unmarshal(body, &jsonObj); err == nil {
			if prettyBody, err := json.MarshalIndent(jsonObj, "  ", "  "); err == nil {
				buf.WriteString("  \"body\": ")
				buf.Write(prettyBody)
				buf.WriteString("\n")
			} else {
				buf.WriteString(fmt.Sprintf("  \"body\": %q\n", string(body)))
			}
		} else {
			buf.WriteString(fmt.Sprintf("  \"body\": %q\n", string(body)))
		}
	} else {
		buf.WriteString(fmt.Sprintf("  \"body\": %q\n", string(body)))
	}

	buf.WriteString("}\n")

	if err := os.WriteFile(localPath, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("Failed to write file: %s", err)
	}
	return localPath, nil
}

func generateFilename(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Sprintf("response_%d", time.Now().Unix())
	}

	name := strings.ReplaceAll(parsed.Host, ":", "_")
	path := strings.Trim(parsed.Path, "/")
	if path != "" {
		path = strings.ReplaceAll(path, "/", "_")
		path = strings.ReplaceAll(path, "\\", "_")
		name += "_" + path
	}

	if parsed.RawQuery != "" {
		name += fmt.Sprintf("_%x", hash(parsed.RawQuery))
	}

	if len(name) > 100 {
		name = name[:100]
	}

	return name
}

func hash(s string) uint32 {
	var h uint32
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}

// copyToClipboard copies text to the system clipboard.
func copyToClipboard(text string) error {
	cmds := []struct {
		name string
		args []string
	}{
		{"wl-copy", nil},
		{"pbcopy", nil},
		{"xclip", []string{"-selection", "clipboard"}},
		{"xsel", []string{"--clipboard", "--input"}},
	}

	for _, c := range cmds {
		path, err := exec.LookPath(c.name)
		if err != nil {
			continue
		}
		cmd := exec.Command(path, c.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no clipboard utility found (install xclip, xsel, or wl-copy)")
}

// getLocalIP returns the preferred outbound IP of this machine.
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// ShowMessage sets a status message (used by secondary instance disconnect).
func (m *Model) ShowMessage(format string, args ...interface{}) {
	m.statusMsg = fmt.Sprintf(format, args...)
}
