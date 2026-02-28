package session

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"nabu/internal/config"
	"nabu/internal/debug"
	"nabu/internal/model"
	"nabu/internal/proxy"
	"nabu/internal/viewmodel"
)

const (
	sessionsDir   = "sessions"
	debounceDelay = 2 * time.Second
)

// SessionDir returns the sessions directory path, creating it if needed.
func SessionDir() (string, error) {
	dir := filepath.Join(config.GetConfigDir(), sessionsDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// Session manages automatic persistence of flows to a HAR file.
type Session struct {
	flowStore *proxy.FlowStore
	path      string
	notify    chan struct{}
	done      chan struct{}
}

// New creates a new session file and starts background auto-save.
func New(flowStore *proxy.FlowStore) (*Session, error) {
	dir, err := SessionDir()
	if err != nil {
		return nil, err
	}

	name := time.Now().Format("2006-01-02T15-04-05") + ".har"
	path := filepath.Join(dir, name)

	s := &Session{
		flowStore: flowStore,
		path:      path,
		notify:    make(chan struct{}, 1),
		done:      make(chan struct{}),
	}

	flowStore.OnChange(func() {
		select {
		case s.notify <- struct{}{}:
		default:
		}
	})

	go s.run()
	return s, nil
}

func (s *Session) run() {
	defer close(s.done)

	var timer *time.Timer
	var timerCh <-chan time.Time
	for {
		select {
		case _, ok := <-s.notify:
			if !ok {
				// Channel closed — do final flush and exit.
				s.flush()
				return
			}
			// Debounce: reset or start a timer.
			if timer == nil {
				timer = time.NewTimer(debounceDelay)
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(debounceDelay)
			}
			timerCh = timer.C
		case <-timerCh:
			s.flush()
			timer = nil
			timerCh = nil
		}
	}
}

func (s *Session) flush() {
	flows := s.flowStore.All()
	if len(flows) == 0 {
		return
	}

	data, err := viewmodel.FormatHAR(flows)
	if err != nil {
		debug.Log("session: failed to format HAR: %v", err)
		return
	}

	// Atomic write: write to tmp file, then rename.
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		debug.Log("session: failed to write tmp file: %v", err)
		return
	}
	if err := os.Rename(tmp, s.path); err != nil {
		debug.Log("session: failed to rename tmp file: %v", err)
		return
	}
	debug.Log("session: flushed %d flows to %s", len(flows), filepath.Base(s.path))
}

// Stop performs a final flush and stops the background goroutine.
func (s *Session) Stop() {
	close(s.notify)
	<-s.done
}

// LoadLatestSession finds and parses the most recent session HAR file.
// Returns the parsed flows and the file path. Returns nil, "", nil if no session files exist.
func LoadLatestSession() ([]*model.Flow, string, error) {
	dir, err := SessionDir()
	if err != nil {
		return nil, "", err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", err
	}

	// Filter to .har files only.
	var harFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".har") {
			harFiles = append(harFiles, e.Name())
		}
	}
	if len(harFiles) == 0 {
		return nil, "", nil
	}

	sort.Strings(harFiles)
	latest := filepath.Join(dir, harFiles[len(harFiles)-1])

	data, err := os.ReadFile(latest)
	if err != nil {
		return nil, "", err
	}

	flows, err := viewmodel.ParseHAR(data)
	if err != nil {
		return nil, "", err
	}

	debug.Log("session: loaded %d flows from %s", len(flows), filepath.Base(latest))
	return flows, latest, nil
}
