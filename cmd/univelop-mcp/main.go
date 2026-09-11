// Command univelop-mcp serves the Univelop API v2 as a Model Context
// Protocol (MCP) server over SSE/HTTPS (default) or stdio.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dividebyzero/univelop-api-mcp/internal/config"
	"github.com/dividebyzero/univelop-api-mcp/internal/mcp"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	// Subcommand: validate
	if len(os.Args) > 1 && os.Args[1] == "validate" {
		exitCode, err := runValidate(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(exitCode)
		}
		os.Exit(exitCode)
	}

	fs := flag.NewFlagSet("univelop-mcp", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "path to the YAML configuration file")
	stdioMode := fs.Bool("stdio", false, "run over stdio (no TLS required; use for local MCP hosts like Claude Desktop)")
	devTLS := fs.Bool("dev-tls", false, "generate an in-memory self-signed certificate for development (SSE mode only)")
	port := fs.Int("port", 0, "override the configured TCP port (0 = use config)")
	showVersion := fs.Bool("version", false, "print version and exit")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: univelop-mcp [validate] [options]\n\n")
		fmt.Fprintf(fs.Output(), "MCP server exposing the Univelop API v2. Runs SSE over HTTPS by default.\n\nOptions:\n")
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\nSubcommands:\n  validate   load and validate the config file, then exit\n")
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}

	if *showVersion {
		fmt.Printf("univelop-mcp %s (commit %s)\n", version, commit)
		return
	}

	if err := run(*configPath, *stdioMode, *devTLS, *port); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

// run is the main entry point; split out for testability.
func run(configPath string, stdioMode, devTLS bool, port int) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if port != 0 {
		cfg.MCP.Port = port
	}

	srv, err := mcp.New(cfg)
	if err != nil {
		return err
	}

	if stdioMode {
		return srv.RunStdio(context.Background())
	}
	return srv.RunSSE(context.Background(), devTLS)
}

// runValidate loads and validates the config, printing a summary. Returns exit code.
func runValidate(args []string) (int, error) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "path to the YAML configuration file")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		return 1, err
	}
	fmt.Printf("config %q is valid\n", *configPath)
	fmt.Printf("  workspaces: %d\n", len(cfg.Workspaces))
	for alias, w := range cfg.Workspaces {
		label := w.Label
		if label == "" {
			label = alias
		}
		fmt.Printf("    - %s (label: %q, base_url: %s, api_key: %s)\n",
			alias, label, w.BaseURL, mask(w.APIKey))
	}
	fmt.Printf("  mcp: port=%d tls=%s server=%q v=%q\n",
		cfg.MCP.Port, tlsState(cfg), cfg.MCP.Name, cfg.MCP.Version)
	return 0, nil
}

func mask(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

func tlsState(cfg *config.Config) string {
	if cfg.MCP.CertFile != "" && cfg.MCP.KeyFile != "" {
		return "configured"
	}
	return "MISSING (use --dev-tls or set cert_file/key_file)"
}