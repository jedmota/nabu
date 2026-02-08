package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"proxy-tui/internal/ipc"
	"proxy-tui/internal/proxy"
	"proxy-tui/internal/ui"
	"proxy-tui/internal/viewmodel"
	"proxy-tui/pkg/ca"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 9090, "Proxy port")
	bind := flag.String("bind", "0.0.0.0", "Bind address (0.0.0.0 for all interfaces, 127.0.0.1 for localhost only)")
	verbose := flag.Bool("verbose", false, "Verbose logging")
	showCA := flag.Bool("show-ca", false, "Show CA certificate path and exit")
	headless := flag.Bool("headless", false, "Run without TUI (for testing)")
	flag.Parse()

	// Load or generate CA
	certificate, err := ca.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load CA: %v\n", err)
		os.Exit(1)
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
	config := &proxy.Config{
		Port:        port,
		BindAddress: bind,
		CA:          certificate,
		MaxFlows:    10000,
		Verbose:     verbose,
	}

	p, err := proxy.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create proxy: %v\n", err)
		os.Exit(1)
	}

	// Start IPC server
	ipcServer, err := ipc.NewServer(p, port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to start IPC server: %v\n", err)
		// Non-fatal — continue without IPC support
	}

	vm := viewmodel.New(p)
	vm.StartEventLoop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if headless {
		fmt.Printf("Proxy running in headless mode on %s:%d\n", bind, port)
		fmt.Printf("CA Certificate: %s\n", certificate.CertPath())
		if ipcServer != nil {
			fmt.Printf("IPC socket: %s\n", ipc.SocketPath(port))
		}

		if err := p.StartAsync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start proxy: %v\n", err)
			os.Exit(1)
		}

		<-sigChan
		fmt.Println("\nShutting down...")
		if ipcServer != nil {
			ipcServer.Stop()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := p.Stop(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
		}
		fmt.Println("Goodbye!")
		return
	}

	// TUI mode
	app := ui.NewApp(vm)

	if err := p.StartAsync(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start proxy: %v\n", err)
		os.Exit(1)
	}

	go func() {
		fmt.Printf("Proxy TUI starting on %s:%d...\n", bind, port)
		fmt.Printf("CA Certificate: %s\n", certificate.CertPath())
	}()

	go func() {
		<-sigChan
		app.Stop()
	}()

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
	}

	fmt.Println("\nShutting down...")
	if ipcServer != nil {
		ipcServer.Stop()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Stop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
	}
	fmt.Println("Goodbye!")
}

// runSecondary connects to an existing primary instance via IPC and runs TUI only.
func runSecondary(port int) {
	fmt.Printf("Connecting to existing instance on port %d...\n", port)

	client, err := ipc.NewClient(port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to primary instance: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	adapter := ipc.NewAdapter(client)

	vm := viewmodel.New(adapter)
	vm.SetSecondary(true)
	vm.StartEventLoop()

	app := ui.NewApp(vm)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Watch for primary disconnection
	go func() {
		select {
		case <-adapter.Disconnected():
			app.ShowMessage("[red]Primary instance disconnected[-]")
		case <-sigChan:
			app.Stop()
		}
	}()

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
	}

	fmt.Println("Goodbye!")
}
