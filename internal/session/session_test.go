package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"nabu/internal/config"
	"nabu/internal/model"
	"nabu/internal/proxy"
)

func init() {
	// Use a temp dir for config during tests.
	config.SetConfigDirOverride("")
}

func setupTestConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	config.SetConfigDirOverride(dir)
	t.Cleanup(func() { config.SetConfigDirOverride("") })
}

func TestSession_FlushAndLoad(t *testing.T) {
	setupTestConfig(t)

	eventCh := make(chan model.FlowEvent, 100)
	store := proxy.NewFlowStore(1000, eventCh)

	sess, err := New(store)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Add a flow.
	store.Add(&model.Flow{
		StartTime: time.Now(),
		Request: &model.Request{
			Method: "GET",
			URL:    "https://example.com/test",
			Host:   "example.com",
			Path:   "/test",
			Proto:  "HTTP/1.1",
		},
		Response: &model.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Proto:      "HTTP/1.1",
		},
	})

	// Stop triggers final flush.
	sess.Stop()

	// Verify session file was created.
	dir, _ := SessionDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var harFiles []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".har" {
			harFiles = append(harFiles, e.Name())
		}
	}
	if len(harFiles) != 1 {
		t.Fatalf("expected 1 HAR file, got %d", len(harFiles))
	}

	// Load it back.
	flows, path, err := LoadLatestSession()
	if err != nil {
		t.Fatalf("LoadLatestSession: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(flows))
	}
	if flows[0].Request.URL != "https://example.com/test" {
		t.Errorf("unexpected URL: %s", flows[0].Request.URL)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}

func TestLoadLatestSession_Empty(t *testing.T) {
	setupTestConfig(t)

	flows, path, err := LoadLatestSession()
	if err != nil {
		t.Fatalf("LoadLatestSession: %v", err)
	}
	if flows != nil {
		t.Errorf("expected nil flows, got %d", len(flows))
	}
	if path != "" {
		t.Errorf("expected empty path, got %s", path)
	}
}
