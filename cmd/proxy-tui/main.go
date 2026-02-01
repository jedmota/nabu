package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"proxy-tui/internal/proxy"
	"proxy-tui/internal/ui"
	"proxy-tui/internal/viewmodel"
	"proxy-tui/pkg/ca"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "Proxy port")
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

	// Create proxy configuration
	config := &proxy.Config{
		Port:        *port,
		BindAddress: *bind,
		CA:          certificate,
		MaxFlows:    10000,
		Verbose:     *verbose,
	}

	// Create proxy
	p, err := proxy.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create proxy: %v\n", err)
		os.Exit(1)
	}

	// Create ViewModel
	vm := viewmodel.New(p)
	vm.StartEventLoop()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Headless mode - run without TUI
	if *headless {
		fmt.Printf("Proxy running in headless mode on %s:%d\n", *bind, *port)
		fmt.Printf("CA Certificate: %s\n", certificate.CertPath())

		// Start proxy
		if err := p.StartAsync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start proxy: %v\n", err)
			os.Exit(1)
		}

		// Wait for shutdown signal
		<-sigChan
		fmt.Println("\nShutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := p.Stop(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
		}
		fmt.Println("Goodbye!")
		return
	}

	// Create TUI application
	app := ui.NewApp(vm)

	// Start proxy in background
	if err := p.StartAsync(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start proxy: %v\n", err)
		os.Exit(1)
	}

	// Run proxy info goroutine
	go func() {
		// Print startup message (will be overwritten by TUI)
		fmt.Printf("Proxy TUI starting on %s:%d...\n", *bind, *port)
		fmt.Printf("CA Certificate: %s\n", certificate.CertPath())
	}()

	// Handle shutdown
	go func() {
		<-sigChan
		app.Stop()
	}()

	// Run TUI (blocks until exit)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
	}

	// Graceful shutdown
	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Stop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
	}

	fmt.Println("Goodbye!")
}
