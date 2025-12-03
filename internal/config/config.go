// Package config provides configuration types and CLI flag parsing for diagnose-mcp
package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// ConnectionType represents the type of MCP server connection
type ConnectionType string

const (
	// ConnectionTypeLocal indicates a local stdio-based MCP server
	ConnectionTypeLocal ConnectionType = "local"
	// ConnectionTypeRemote indicates a remote HTTP/WebSocket MCP server
	ConnectionTypeRemote ConnectionType = "remote"
)

// OutputFormat represents the log output format
type OutputFormat string

const (
	// OutputText indicates human-readable text format (default)
	OutputText OutputFormat = "text"
	// OutputJSON indicates machine-parsable JSON format
	OutputJSON OutputFormat = "json"
)

// Config represents the complete configuration for diagnose-mcp
type Config struct {
	// ConnectionType is local or remote
	ConnectionType ConnectionType

	// ServerBinary is the path to the local MCP server binary (local mode)
	ServerBinary string
	// ServerArgs are arguments to pass to the local server (local mode)
	ServerArgs []string

	// RemoteURL is the URL of the remote MCP server (remote mode)
	RemoteURL string

	// WatchedFiles are file paths to monitor for changes
	WatchedFiles []string

	// LogFile is the path to write logs to (default: stderr)
	LogFile string

	// Verbose enables detailed logging (full message payloads)
	Verbose bool

	// OutputFormat is text or json
	OutputFormat OutputFormat

	// ShowHelp indicates --help flag was provided
	ShowHelp bool
	// ShowVersion indicates --version flag was provided
	ShowVersion bool
}

// stringSliceFlag implements flag.Value for multiple string flags
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// ParseFlags parses command-line flags and returns a Config
func ParseFlags(args []string) (*Config, error) {
	cfg := &Config{}

	fs := flag.NewFlagSet("diagnose-mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var watchFiles stringSliceFlag

	// Define flags
	fs.StringVar(&cfg.RemoteURL, "remote", "", "URL of remote MCP server (http://, https://, ws://, wss://)")
	fs.StringVar(&cfg.RemoteURL, "r", "", "URL of remote MCP server (shorthand)")
	fs.Var(&watchFiles, "watch", "File path to monitor (can be repeated)")
	fs.Var(&watchFiles, "w", "File path to monitor (shorthand)")
	fs.StringVar(&cfg.LogFile, "log-file", "", "Path to write logs (supports {session}, {pid} patterns, default: stderr)")
	fs.StringVar(&cfg.LogFile, "l", "", "Path to write logs (shorthand)")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "Enable detailed logging with full message payloads")
	fs.BoolVar(&cfg.Verbose, "v", false, "Enable detailed logging (shorthand)")

	jsonFlag := false
	fs.BoolVar(&jsonFlag, "json", false, "Output logs in JSON format")
	fs.BoolVar(&jsonFlag, "j", false, "Output logs in JSON format (shorthand)")

	fs.BoolVar(&cfg.ShowHelp, "help", false, "Display help message")
	fs.BoolVar(&cfg.ShowHelp, "h", false, "Display help message (shorthand)")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "Display version")
	fs.BoolVar(&cfg.ShowVersion, "V", false, "Display version (shorthand)")

	// Parse flags
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Handle help and version early
	if cfg.ShowHelp || cfg.ShowVersion {
		return cfg, nil
	}

	// Set output format
	if jsonFlag {
		cfg.OutputFormat = OutputJSON
	} else {
		cfg.OutputFormat = OutputText
	}

	// Set watched files
	cfg.WatchedFiles = watchFiles

	// Determine connection type and validate
	positionalArgs := fs.Args()

	if cfg.RemoteURL != "" {
		// Remote mode
		cfg.ConnectionType = ConnectionTypeRemote

		// Validate remote URL
		if err := validateRemoteURL(cfg.RemoteURL); err != nil {
			return nil, err
		}

		// Remote mode should not have server binary
		if len(positionalArgs) > 0 {
			return nil, fmt.Errorf("conflicting modes: --remote flag cannot be used with local server binary")
		}
	} else {
		// Local mode
		cfg.ConnectionType = ConnectionTypeLocal

		// Local mode requires server binary
		if len(positionalArgs) == 0 {
			return nil, fmt.Errorf("missing required argument: <server-binary> (or use --remote for remote mode)")
		}

		cfg.ServerBinary = positionalArgs[0]
		if len(positionalArgs) > 1 {
			cfg.ServerArgs = positionalArgs[1:]
		}
	}

	return cfg, nil
}

// validateRemoteURL validates that the remote URL has a valid scheme
func validateRemoteURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("remote URL cannot be empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid remote URL: %w", err)
	}

	validSchemes := map[string]bool{
		"http":  true,
		"https": true,
		"ws":    true,
		"wss":   true,
	}

	if !validSchemes[u.Scheme] {
		return fmt.Errorf("invalid remote URL scheme: %s (must be http://, https://, ws://, or wss://)", u.Scheme)
	}

	return nil
}

// Usage returns the help message
func Usage() string {
	return `diagnose-mcp - MCP Protocol Proxy Server

USAGE:
  diagnose-mcp [options] <server-binary> [server-args...]   # Local mode
  diagnose-mcp --remote <url> [options]                      # Remote mode

OPTIONS:
  --remote, -r <url>    URL of remote MCP server (http://, https://, ws://, wss://)
  --watch, -w <path>    Monitor file for changes (can be repeated)
  --log-file, -l <path> Write logs to file (supports {session}, {pid} patterns)
  --verbose, -v         Enable detailed logging with full message payloads
  --json, -j            Output logs in JSON format
  --help, -h            Display this help message
  --version, -V         Display version information

EXAMPLES:
  diagnose-mcp ./my-mcp-server --port 8080
  diagnose-mcp --verbose ./my-server
  diagnose-mcp --remote ws://localhost:8080/mcp
  diagnose-mcp --watch /tmp/server.log ./my-server
  diagnose-mcp --json ./my-server | jq .
  diagnose-mcp --log-file "logs/{session}.log" ./my-server

For more information, see the documentation.
`
}

// Version returns the version string
const Version = "1.0.0"

func VersionString() string {
	return fmt.Sprintf("diagnose-mcp version %s", Version)
}
