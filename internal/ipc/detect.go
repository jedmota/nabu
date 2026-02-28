package ipc

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// socketDirOverride, when non-empty, is used by SocketPath instead of
// the default ~/.nabu. Tests in this package set it directly.
var socketDirOverride string

// SocketPath returns the Unix domain socket path for a given proxy port.
func SocketPath(port int) string {
	var dir string
	if socketDirOverride != "" {
		dir = socketDirOverride
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "/tmp"
		}
		dir = filepath.Join(home, ".nabu")
	}
	os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, fmt.Sprintf("proxy-%d.sock", port))
}

// IsInstanceRunning checks whether another nabu instance is already
// serving on the given port by probing the Unix domain socket.
// If a stale socket exists (no one listening), it is cleaned up.
func IsInstanceRunning(port int) bool {
	sockPath := SocketPath(port)

	// Check if socket file exists at all
	if _, err := os.Stat(sockPath); os.IsNotExist(err) {
		return false
	}

	// Try to connect
	conn, err := net.DialTimeout("unix", sockPath, 2*time.Second)
	if err != nil {
		// Stale socket — clean it up
		os.Remove(sockPath)
		return false
	}
	conn.Close()
	return true
}
