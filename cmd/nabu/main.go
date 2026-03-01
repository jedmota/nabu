package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"nabu/internal/config"
	"nabu/internal/debug"
	"nabu/internal/ipc"
	"nabu/internal/proxy"
	"nabu/internal/session"
	"nabu/internal/ui"
	"nabu/internal/viewmodel"
	"nabu/pkg/ca"
)

// version is set at build time via -ldflags "-X main.version=x.y.z"
var version = "dev"

type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(val string) error {
	*s = append(*s, val)
	return nil
}

// fatal logs an error message and exits.
func fatal(format string, args ...any) {
	debug.Log("FATAL: "+format, args...)
	os.Exit(1)
}

// warn logs a warning message.
func warn(format string, args ...any) {
	debug.Log("WARN: "+format, args...)
}

// notifyRunningInstance sends a config_reload command to a running instance
// on the given port, if one exists. Errors are silently ignored.
func notifyRunningInstance(port int) {
	if !ipc.IsInstanceRunning(port) {
		return
	}
	client, err := ipc.NewClient(port)
	if err != nil {
		return
	}
	client.SendConfigReload()
	client.Close()
}

func main() {
	ui.SetVersion(version)
	viewmodel.SetAppVersion(version)

	// Parse command line flags
	port := flag.Int("port", 9090, "Proxy port")
	bind := flag.String("bind", "0.0.0.0", "Bind address (0.0.0.0 for all interfaces, 127.0.0.1 for localhost only)")
	verbose := flag.Bool("verbose", false, "Verbose logging")
	showCA := flag.Bool("show-ca", false, "Show CA certificate path and exit")
	headless := flag.Bool("headless", false, "Run without TUI (for testing)")

	// Add (repeatable, proxy continues to start)
	var whitelistFlags, mapLocalFlags, mapRemoteFlags stringSlice
	flag.Var(&whitelistFlags, "whitelist", "Add whitelist pattern (repeatable)")
	flag.Var(&mapLocalFlags, "map-local", "Add map-local rule as pattern=>localpath (repeatable)")
	flag.Var(&mapRemoteFlags, "map-remote", "Add map-remote rule as pattern=>remoteurl (repeatable)")

	// List (print with IDs, then exit)
	listWhitelist := flag.Bool("list-whitelist", false, "List whitelist patterns and exit")
	listMapLocal := flag.Bool("list-map-local", false, "List map-local rules and exit")
	listMapRemote := flag.Bool("list-map-remote", false, "List map-remote rules and exit")

	// Remove by 1-based ID (remove, then exit)
	rmWhitelist := flag.Int("rm-whitelist", 0, "Remove whitelist pattern by ID and exit")
	rmMapLocal := flag.Int("rm-map-local", 0, "Remove map-local rule by ID and exit")
	rmMapRemote := flag.Int("rm-map-remote", 0, "Remove map-remote rule by ID and exit")

	flag.Parse()

	// Start debug logger (best-effort; failure is non-fatal)
	_ = debug.Init()
	defer debug.Close()

	// Handle --list-* flags (print and exit)
	if *listWhitelist {
		listConfig("whitelist patterns", config.LoadWhitelist,
			func(p config.WhitelistPattern) (string, bool) { return p.Pattern, p.Enabled })
		return
	}
	if *listMapLocal {
		listConfig("map-local rules", config.LoadMapLocal,
			func(e config.MapLocalEntry) (string, bool) { return e.Pattern + " => " + e.LocalPath, e.Enabled })
		return
	}
	if *listMapRemote {
		listConfig("map-remote rules", config.LoadMapRemote,
			func(e config.MapRemoteEntry) (string, bool) { return e.Pattern + " => " + e.RemoteURL, e.Enabled })
		return
	}

	// Handle --rm-* flags (remove and exit)
	if *rmWhitelist > 0 {
		removeConfig("whitelist pattern", *rmWhitelist, *port,
			config.LoadWhitelist, config.SaveWhitelist,
			func(p config.WhitelistPattern) string { return p.Pattern })
		return
	}
	if *rmMapLocal > 0 {
		removeConfig("map-local rule", *rmMapLocal, *port,
			config.LoadMapLocal, config.SaveMapLocal,
			func(e config.MapLocalEntry) string { return e.Pattern + " => " + e.LocalPath })
		return
	}
	if *rmMapRemote > 0 {
		removeConfig("map-remote rule", *rmMapRemote, *port,
			config.LoadMapRemote, config.SaveMapRemote,
			func(e config.MapRemoteEntry) string { return e.Pattern + " => " + e.RemoteURL })
		return
	}

	// Handle --whitelist / --map-local / --map-remote (add, then continue to start proxy)
	for _, pattern := range whitelistFlags {
		if err := config.AddToWhitelist(pattern); err != nil {
			fatal("Failed to add whitelist pattern %q: %v", pattern, err)
		}
		debug.Log("Added whitelist pattern: %s", pattern)
	}
	for _, rule := range mapLocalFlags {
		parts := strings.SplitN(rule, "=>", 2)
		if len(parts) != 2 {
			fatal("Invalid map-local rule %q (expected pattern=>localpath)", rule)
		}
		entry := config.MapLocalEntry{
			Pattern:   strings.TrimSpace(parts[0]),
			LocalPath: strings.TrimSpace(parts[1]),
			Enabled:   true,
		}
		if err := config.AddMapLocalEntry(entry); err != nil {
			fatal("Failed to add map-local rule %q: %v", rule, err)
		}
		debug.Log("Added map-local rule: %s => %s", entry.Pattern, entry.LocalPath)
	}
	for _, rule := range mapRemoteFlags {
		parts := strings.SplitN(rule, "=>", 2)
		if len(parts) != 2 {
			fatal("Invalid map-remote rule %q (expected pattern=>remoteurl)", rule)
		}
		entry := config.MapRemoteEntry{
			Pattern:   strings.TrimSpace(parts[0]),
			RemoteURL: strings.TrimSpace(parts[1]),
			Enabled:   true,
		}
		if err := config.AddMapRemoteEntry(entry); err != nil {
			fatal("Failed to add map-remote rule %q: %v", rule, err)
		}
		debug.Log("Added map-remote rule: %s => %s", entry.Pattern, entry.RemoteURL)
	}
	if len(whitelistFlags) > 0 || len(mapLocalFlags) > 0 || len(mapRemoteFlags) > 0 {
		notifyRunningInstance(*port)
	}

	// Load or generate CA
	certificate, err := ca.Load()
	if err != nil {
		fatal("Failed to load CA: %v", err)
	}

	// If --show-ca, just print CA info and exit
	if *showCA {
		fmt.Println("CA Certificate Path:", certificate.CertPath())
		fmt.Println("CA Fingerprint:", certificate.Fingerprint())
		fmt.Println("\nTo install the CA certificate, run:")
		fmt.Println("  ./scripts/install-ca.sh")
		return
	}

	// Check if another instance is already running on this port
	if ipc.IsInstanceRunning(*port) {
		runSecondary(*port)
		return
	}

	// Primary mode
	runPrimary(*port, *bind, *verbose, *headless, certificate)
}

// runPrimary starts the proxy, IPC server, and TUI (or headless loop).
func runPrimary(port int, bind string, verbose, headless bool, certificate *ca.CA) {
	proxyCfg := &proxy.Config{
		Port:        port,
		BindAddress: bind,
		CA:          certificate,
		MaxFlows:    10000,
		Verbose:     verbose,
	}

	p, err := proxy.New(proxyCfg)
	if err != nil {
		fatal("Failed to create proxy: %v", err)
	}

	// Start IPC server
	ipcServer, err := ipc.NewServer(p, port)
	if err != nil {
		warn("Failed to start IPC server: %v", err)
		// Non-fatal — continue without IPC support
	}

	vm := viewmodel.New(p, config.DefaultPersistence{})

	// Start auto-saving flows to a session file.
	sess, err := session.New(p.FlowStore())
	if err != nil {
		warn("Failed to start session: %v", err)
	}

	vm.StartEventLoop()

	// Listen for config reload requests from secondary instances
	if ipcServer != nil {
		go func() {
			for range ipcServer.ConfigReloads() {
				vm.ReloadConfig()
			}
		}()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if headless {
		fmt.Printf("Proxy running in headless mode on %s:%d\n", bind, port)
		fmt.Printf("CA Certificate: %s\n", certificate.CertPath())
		if ipcServer != nil {
			fmt.Printf("IPC socket: %s\n", ipc.SocketPath(port))
		}

		if err := p.StartAsync(); err != nil {
			fatal("Failed to start proxy: %v", err)
		}

		<-sigChan
		debug.Log("Shutting down (headless)...")
		if sess != nil {
			sess.Stop()
		}
		if ipcServer != nil {
			ipcServer.Stop()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := p.Stop(ctx); err != nil {
			warn("Error during shutdown: %v", err)
		}
		debug.Log("Shutdown complete")
		return
	}

	// TUI mode
	if err := p.StartAsync(); err != nil {
		fatal("Failed to start proxy: %v", err)
	}

	m := ui.NewModel(vm)
	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	go func() {
		<-sigChan
		prog.Quit()
	}()

	if _, err := prog.Run(); err != nil {
		warn("TUI error: %v", err)
	}

	debug.Log("Shutting down...")
	if sess != nil {
		sess.Stop()
	}
	if ipcServer != nil {
		ipcServer.Stop()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Stop(ctx); err != nil {
		warn("Error during shutdown: %v", err)
	}
	debug.Log("Shutdown complete")
}

// runSecondary connects to an existing primary instance via IPC and runs TUI only.
func runSecondary(port int) {
	debug.Log("Connecting to existing instance on port %d...", port)

	client, err := ipc.NewClient(port)
	if err != nil {
		fatal("Failed to connect to primary instance: %v", err)
	}
	defer client.Close()

	adapter := ipc.NewAdapter(client)

	vm := viewmodel.New(adapter, config.DefaultPersistence{})
	vm.SetSecondary(true)
	vm.StartEventLoop()

	m := ui.NewModel(vm)
	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Watch for primary disconnection
	go func() {
		select {
		case <-adapter.Disconnected():
			prog.Send(ui.StatusMsg{Text: "Primary instance disconnected"})
		case <-sigChan:
			prog.Quit()
		}
	}()

	if _, err := prog.Run(); err != nil {
		warn("TUI error: %v", err)
	}

	debug.Log("Secondary instance shutdown complete")
}

// listConfig loads entries and prints them with 1-based IDs.
func listConfig[E any](name string, load func() ([]E, error), format func(E) (string, bool)) {
	entries, err := load()
	if err != nil {
		fatal("Failed to load %s: %v", name, err)
	}
	if len(entries) == 0 {
		fmt.Printf("No %s configured.\n", name)
		return
	}
	for i, e := range entries {
		display, enabled := format(e)
		status := "enabled"
		if !enabled {
			status = "disabled"
		}
		fmt.Printf("%d: %s (%s)\n", i+1, display, status)
	}
}

// removeConfig removes the entry at 1-based id, saves, and notifies any running instance.
func removeConfig[E any](name string, id, port int, load func() ([]E, error), save func([]E) error, display func(E) string) {
	entries, err := load()
	if err != nil {
		fatal("Failed to load %s: %v", name, err)
	}
	idx := id - 1
	if idx < 0 || idx >= len(entries) {
		fatal("Invalid %s ID %d (have %d entries)", name, id, len(entries))
	}
	removed := display(entries[idx])
	entries = append(entries[:idx], entries[idx+1:]...)
	if err := save(entries); err != nil {
		fatal("Failed to save %s: %v", name, err)
	}
	fmt.Printf("Removed %s: %s\n", name, removed)
	notifyRunningInstance(port)
}
