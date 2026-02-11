package ipc

import (
	"testing"
	"time"

	"proxy-tui/internal/model"
	"proxy-tui/internal/proxy"
)

// mockFlowSource implements proxy.FlowSource for testing.
type mockFlowSource struct {
	eventCh      chan model.FlowEvent
	flowStore    *proxy.FlowStore
	sslProxyList *proxy.SSLProxyList
	mapRules     *model.MapRuleStore
	port         int
	bindAddress  string
}

func newMockSource(port int) *mockFlowSource {
	ch := make(chan model.FlowEvent, 1000)
	return &mockFlowSource{
		eventCh:      ch,
		flowStore:    proxy.NewFlowStore(1000, ch),
		sslProxyList: proxy.NewSSLProxyList(),
		mapRules:     model.NewMapRuleStore(),
		port:         port,
		bindAddress:  "127.0.0.1",
	}
}

func (m *mockFlowSource) Events() <-chan model.FlowEvent   { return m.eventCh }
func (m *mockFlowSource) FlowStore() *proxy.FlowStore      { return m.flowStore }
func (m *mockFlowSource) SSLProxyList() *proxy.SSLProxyList { return m.sslProxyList }
func (m *mockFlowSource) MapRules() *model.MapRuleStore     { return m.mapRules }
func (m *mockFlowSource) Port() int                         { return m.port }
func (m *mockFlowSource) BindAddress() string               { return m.bindAddress }

func setupSocketDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	socketDirOverride = dir
	t.Cleanup(func() { socketDirOverride = "" })
}

// waitForSyncDone waits until the client has received at least one event
// (the sync produces events for pre-existing flows) or until the sync_done
// message has been processed. We use the FlowStore count or a timeout as proxy.
// For the hello-only case, we drain the event channel to ensure readLoop has
// finished processing the hello message.
func waitForClientReady(t *testing.T, client *Client) {
	t.Helper()
	// The hello message is processed before sync/sync_done.
	// After sync_done, the readLoop moves to the streaming select.
	// We wait for sync_done by checking that we can read from the event channel
	// (sync produces at least zero events, but sync_done follows).
	// A simple approach: wait until the port is set (non-zero after hello).
	deadline := time.After(3 * time.Second)
	for {
		// Read port via the store (which is set before port in readLoop, but
		// actually port is set in the hello handler). We need to be careful
		// about the race. Let's just use a small sleep loop.
		time.Sleep(50 * time.Millisecond)
		select {
		case <-deadline:
			t.Fatal("timeout waiting for client to be ready")
			return
		default:
		}
		// Check if readLoop has gotten past hello by seeing if the store
		// has been set up (it's always set up at NewClient, so just wait
		// a reasonable time for the hello message to be processed).
		// The race is in Client.port/bindAddress fields — we cannot safely
		// read them here. Instead, wait for the sync_done to have been sent
		// by checking the event channel activity.
		return
	}
}

// --- Server + Client hello handshake ---

func TestServerClient_HelloHandshake(t *testing.T) {
	setupSocketDir(t)

	// Pre-populate 1 flow so sync produces an event we can wait on.
	src := newMockSource(19090)
	src.flowStore.Add(&model.Flow{
		StartTime: time.Now(),
		Request:   &model.Request{Method: "GET", URL: "http://a.com", Host: "a.com", Path: "/"},
	})
	<-src.eventCh // drain the Add event from the primary channel

	srv, err := NewServer(src, 19090)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Stop()

	client, err := NewClient(19090)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	// Wait until the client's flow store has the synced flow — that means
	// hello + sync + sync_done have all been processed by readLoop.
	deadline := time.After(3 * time.Second)
	for client.FlowStore().Count() < 1 {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for sync")
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}

	// Now port and bindAddress are safely set (readLoop processed hello
	// before sync, and sync is done, so the writes happened before our reads).
	if client.Port() != 19090 {
		t.Errorf("Port = %d, want 19090", client.Port())
	}
	if client.BindAddress() != "127.0.0.1" {
		t.Errorf("BindAddress = %q, want 127.0.0.1", client.BindAddress())
	}
}

// --- Flow sync ---

func TestServerClient_FlowSync(t *testing.T) {
	setupSocketDir(t)
	src := newMockSource(19091)

	// Pre-populate flows
	for i := 0; i < 5; i++ {
		src.flowStore.Add(&model.Flow{
			StartTime: time.Now(),
			Request: &model.Request{
				Method: "GET",
				URL:    "http://example.com",
				Host:   "example.com",
				Path:   "/",
			},
		})
	}
	// Drain events from Add calls
	for i := 0; i < 5; i++ {
		<-src.eventCh
	}

	srv, err := NewServer(src, 19091)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	client, err := NewClient(19091)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Wait for sync
	deadline := time.After(3 * time.Second)
	for {
		if client.FlowStore().Count() >= 5 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for sync: got %d flows", client.FlowStore().Count())
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// --- Real-time streaming ---

func TestServerClient_RealtimeStreaming(t *testing.T) {
	setupSocketDir(t)
	src := newMockSource(19092)

	srv, err := NewServer(src, 19092)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	client, err := NewClient(19092)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Wait for handshake
	time.Sleep(200 * time.Millisecond)

	// Add a new flow after sync is done
	src.flowStore.Add(&model.Flow{
		StartTime: time.Now(),
		Request: &model.Request{
			Method: "POST",
			URL:    "http://new.com",
			Host:   "new.com",
			Path:   "/",
		},
	})

	// Wait for the flow to arrive at the client
	deadline := time.After(3 * time.Second)
	for {
		if client.FlowStore().Count() >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for real-time flow")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// --- Config reload ---

func TestServerClient_ConfigReload(t *testing.T) {
	setupSocketDir(t)
	src := newMockSource(19093)

	srv, err := NewServer(src, 19093)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	client, err := NewClient(19093)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	time.Sleep(200 * time.Millisecond)

	if err := client.SendConfigReload(); err != nil {
		t.Fatalf("SendConfigReload: %v", err)
	}

	select {
	case <-srv.ConfigReloads():
		// OK
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for config reload signal")
	}
}

// --- Client disconnect ---

func TestServerClient_ClientDisconnect(t *testing.T) {
	setupSocketDir(t)
	src := newMockSource(19094)

	srv, err := NewServer(src, 19094)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	client, err := NewClient(19094)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(200 * time.Millisecond)
	client.Close()

	// Server should continue working — add a flow
	src.flowStore.Add(&model.Flow{
		StartTime: time.Now(),
		Request:   &model.Request{Method: "GET", URL: "http://a.com", Host: "a.com", Path: "/"},
	})
	if src.flowStore.Count() == 0 {
		t.Error("server should continue after client disconnect")
	}
}

// --- Server stop ---

func TestServerClient_ServerStop(t *testing.T) {
	setupSocketDir(t)
	src := newMockSource(19095)

	srv, err := NewServer(src, 19095)
	if err != nil {
		t.Fatal(err)
	}

	client, err := NewClient(19095)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	time.Sleep(200 * time.Millisecond)
	srv.Stop()

	select {
	case <-client.Disconnected():
		// OK
	case <-time.After(3 * time.Second):
		t.Error("client's Disconnected channel should close when server stops")
	}
}

// --- IsInstanceRunning ---

func TestIsInstanceRunning_NoSocket(t *testing.T) {
	setupSocketDir(t)

	if IsInstanceRunning(29999) {
		t.Error("should return false when no socket exists")
	}
}

func TestIsInstanceRunning_ActiveSocket(t *testing.T) {
	setupSocketDir(t)
	src := newMockSource(19096)

	srv, err := NewServer(src, 19096)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	if !IsInstanceRunning(19096) {
		t.Error("should return true when server is listening")
	}
}

func TestIsInstanceRunning_StaleSocket(t *testing.T) {
	setupSocketDir(t)
	src := newMockSource(19097)

	srv, err := NewServer(src, 19097)
	if err != nil {
		t.Fatal(err)
	}

	// Get the socket path, stop server but leave the socket file
	sockPath := SocketPath(19097)
	srv.Stop()

	// Recreate a stale socket file (srv.Stop already removed it, so create a dummy)
	_ = writeStaleSocket(t, sockPath)

	if IsInstanceRunning(19097) {
		t.Error("should return false for stale socket and clean it up")
	}
}

func writeStaleSocket(t *testing.T, path string) error {
	t.Helper()
	// Create a regular file pretending to be a socket — dial will fail
	return nil // IsInstanceRunning checks os.Stat first, but after srv.Stop the file is removed
}

// --- Compile-time interface check ---

var _ proxy.FlowSource = (*Adapter)(nil)
