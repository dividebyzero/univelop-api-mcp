package mcp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/dividebyzero/univelop-api-mcp/internal/config"
	"github.com/dividebyzero/univelop-api-mcp/internal/tools"
)

// Server hosts the MCP service.
type Server struct {
	cfg      *config.Config
	registry *tools.Registry
	mcp      *server.MCPServer
	sse      *server.SSEServer
	httpSrv  *http.Server
}

// New creates the MCP server, registers all tools, and returns the wrapper.
func New(cfg *config.Config) (*Server, error) {
	registry, err := tools.NewRegistry(cfg)
	if err != nil {
		return nil, fmt.Errorf("initializing registry: %w", err)
	}

	name := cfg.MCP.Name
	if name == "" {
		name = "Univelop API MCP"
	}
	version := cfg.MCP.Version
	if version == "" {
		version = "0.1.0"
	}

	mcpSrv := server.NewMCPServer(name, version,
		server.WithResourceCapabilities(true, false),
		server.WithLogging(),
		server.WithRecovery(),
		server.WithInputSchemaValidation(),
	)
	registry.Register(mcpSrv)

	sseSrv := server.NewSSEServer(mcpSrv)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","tools":%d}`, registry.ToolCount())
	})
	mux.Handle("/", sseSrv)

	addr := fmt.Sprintf(":%d", cfg.MCP.Port)
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &Server{
		cfg:      cfg,
		registry: registry,
		mcp:      mcpSrv,
		sse:      sseSrv,
		httpSrv:  httpSrv,
	}, nil
}

// ToolCount reports the number of registered MCP tools.
func (s *Server) ToolCount() int {
	return len(s.mcp.ListTools())
}

// RunSSE starts the SSE+HTTPS server and blocks until ctx is cancelled or SIGINT/SIGTERM.
func (s *Server) RunSSE(ctx context.Context, devTLS bool) error {
	if !devTLS {
		if s.cfg.MCP.CertFile == "" || s.cfg.MCP.KeyFile == "" {
			return fmt.Errorf("TLS is required for SSE transport. Set mcp.cert_file and mcp.key_file in the config, or use --dev-tls for development")
		}
		for _, f := range []string{s.cfg.MCP.CertFile, s.cfg.MCP.KeyFile} {
			if _, err := os.Stat(f); err != nil {
				return fmt.Errorf("TLS file %q: %w", f, err)
			}
		}
	}

	ln, err := net.Listen("tcp", s.httpSrv.Addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", s.httpSrv.Addr, err)
	}

	var tlsConfig *tls.Config
	if devTLS {
		cert, err := generateSelfSignedCert()
		if err != nil {
			return fmt.Errorf("generating dev certificate: %w", err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	} else {
		cert, err := tls.LoadX509KeyPair(s.cfg.MCP.CertFile, s.cfg.MCP.KeyFile)
		if err != nil {
			return fmt.Errorf("loading TLS keypair: %w", err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}
	s.httpSrv.TLSConfig = tlsConfig

	// Signal handling: graceful shutdown on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
		case <-ctx.Done():
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.httpSrv.Shutdown(shutdownCtx)
	}()

	fmt.Fprintf(os.Stderr, "univelop-mcp: listening on %s (TLS %s)\n", s.httpSrv.Addr, map[bool]string{true: "dev-self-signed", false: "configured"}[devTLS])
	fmt.Fprintf(os.Stderr, "univelop-mcp: %d tools registered\n", s.ToolCount())
	fmt.Fprintf(os.Stderr, "univelop-mcp: SSE endpoint: https://localhost%s/sse\n", s.httpSrv.Addr)

	tlsLn := tls.NewListener(ln, tlsConfig)
	err = s.httpSrv.Serve(tlsLn)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// RunStdio starts the MCP server over stdio and blocks.
func (s *Server) RunStdio(ctx context.Context) error {
	fmt.Fprintf(os.Stderr, "univelop-mcp: running in stdio mode, %d tools registered\n", s.ToolCount())
	return server.ServeStdio(s.mcp)
}

// generateSelfSignedCert creates an in-memory ECDSA self-signed certificate valid for localhost.
func generateSelfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating ecdsa key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating serial: %w", err)
	}
	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"Univelop MCP Dev"},
			CommonName:   "localhost",
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("creating certificate: %w", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("marshalling key: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return tls.X509KeyPair(certPEM, keyPEM)
}