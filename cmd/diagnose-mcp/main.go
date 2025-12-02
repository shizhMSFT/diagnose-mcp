// Package main provides the diagnose-mcp CLI entry point
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/shizhMSFT/diagnose-mcp/internal/config"
	"github.com/shizhMSFT/diagnose-mcp/internal/proxy"
)

func main() {
	// Parse command-line flags
	cfg, err := config.ParseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Handle --help flag
	if cfg.ShowHelp {
		fmt.Println(config.Usage())
		os.Exit(0)
	}

	// Handle --version flag
	if cfg.ShowVersion {
		fmt.Println(config.VersionString())
		os.Exit(0)
	}

	// Create context with signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Fprintf(os.Stderr, "\nReceived signal: %v\n", sig)
		cancel()
	}()

	// Create and run proxy
	p := proxy.NewProxy(cfg)
	if err := p.Run(ctx); err != nil {
		if err != context.Canceled {
			fmt.Fprintf(os.Stderr, "Proxy error: %v\n", err)
			os.Exit(1)
		}
	}

	os.Exit(0)
}
