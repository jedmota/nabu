package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const appName = "nabu"

var (
	mu   sync.Mutex
	file *os.File
)

// Init opens the debug log file in the OS cache directory.
// Safe to call multiple times; subsequent calls are no-ops.
func Init() error {
	mu.Lock()
	defer mu.Unlock()

	if file != nil {
		return nil
	}

	dir, err := logDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(dir, "debug.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	file = f
	return nil
}

// Close flushes and closes the log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()

	if file != nil {
		file.Close()
		file = nil
	}
}

// Log writes a timestamped message to the debug log.
// No-op if Init has not been called.
func Log(format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()

	if file == nil {
		return
	}

	ts := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(file, "%s  %s\n", ts, msg)
}

// logDir returns the platform-appropriate directory for debug logs.
func logDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appName), nil
}
