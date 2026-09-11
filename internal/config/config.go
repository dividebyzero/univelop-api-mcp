package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Defaults
const (
	DefaultPort       = 8443
	DefaultBaseURL    = "https://app.univelop.de/"
	DefaultTimeout    = 30 * time.Second
	DefaultMaxRetries = 3
)

// Config is the top-level configuration for the MCP server.
type Config struct {
	MCP        MCPConfig            `yaml:"mcp"`
	Univelop   UnivelopConfig       `yaml:"univelop"`
	Workspaces map[string]Workspace `yaml:"workspaces"`
}

// MCPConfig holds MCP server transport settings.
type MCPConfig struct {
	Port     int    `yaml:"port"`
	CertFile string `yaml:"cert_file,omitempty"`
	KeyFile  string `yaml:"key_file,omitempty"`
	Name     string `yaml:"name,omitempty"`
	Version  string `yaml:"version,omitempty"`
}

// UnivelopConfig holds global Univelop API client settings.
type UnivelopConfig struct {
	DefaultBaseURL     string   `yaml:"default_base_url,omitempty"`
	Timeout            Duration `yaml:"timeout,omitempty"` // e.g. "30s" or bare seconds
	MaxRetries         int      `yaml:"max_retries,omitempty"`
	RateLimitPerSecond float64  `yaml:"rate_limit_per_second,omitempty"`
}

// Workspace defines credentials and endpoint for a single workspace.
// The config map key is an alias for humans (e.g. "production"); workspace_id
// is the id the API expects in the URL path. If workspace_id is omitted the
// alias is used directly.
type Workspace struct {
	ID      string `yaml:"workspace_id,omitempty"`
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url,omitempty"`
	Label   string `yaml:"label,omitempty"`
}

// Duration is a time.Duration that unmarshals from either a Go duration
// string ("30s", "1m") or a bare integer number of seconds.
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err == nil {
		parsed, err := time.ParseDuration(s)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", s, err)
		}
		d.Duration = parsed
		return nil
	}
	var secs int
	if err := value.Decode(&secs); err == nil {
		d.Duration = time.Duration(secs) * time.Second
		return nil
	}
	return fmt.Errorf("invalid duration value %q", value.Value)
}

// ResolvedBaseURL returns the effective base URL for this workspace.
func (w *Workspace) ResolvedBaseURL(defaultBase string) string {
	if w.BaseURL != "" {
		return w.BaseURL
	}
	return defaultBase
}

var envVarRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// ExpandEnv replaces ${VAR} references in s with the environment variable value.
// Returns an error if a referenced variable is not set.
func ExpandEnv(s string) (string, error) {
	var firstErr error
	out := envVarRe.ReplaceAllStringFunc(s, func(m string) string {
		name := m[2 : len(m)-1]
		val, ok := os.LookupEnv(name)
		if !ok {
			if firstErr == nil {
				firstErr = fmt.Errorf("environment variable %q is not set (referenced by %s)", name, m)
			}
			return m
		}
		return val
	})
	return out, firstErr
}

// Parse decodes YAML data directly, filling defaults and validating.
// It does NOT perform environment-variable substitution. For test use.
func Parse(yamlData []byte) (*Config, error) {
	cfg := &Config{}
	if err := yaml.Unmarshal(yamlData, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	cfg.fillDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// stripYAMLComments removes YAML comments (# to end of line) so that
// example ${VAR} references in doc comments don't trip env expansion.
func stripYAMLComments(raw string) string {
	lines := strings.Split(raw, "\n")
	var out []string
	for _, line := range lines {
		// A YAML comment is everyhing from the first unquoted '#' to EOL,
		// but for env expansion we only need to strip lines where a '#' appears
		// before any unquoted ${...}.  The safest conservative approach: drop
		// lines whose first non-space character is '#'.  Inline comments that
		// follow a value are rare in this config and harmless when they contain
		// ${...} because yaml parsing skips them anyway — we skip them too.
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// Load reads, expands environment variables, and parses a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cleaned := stripYAMLComments(string(data))
	expanded, err := ExpandEnv(cleaned)
	if err != nil {
		return nil, fmt.Errorf("expanding env vars: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.fillDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) fillDefaults() {
	if c.MCP.Port == 0 {
		c.MCP.Port = DefaultPort
	}
	if c.MCP.Name == "" {
		c.MCP.Name = "Univelop API MCP"
	}
	if c.MCP.Version == "" {
		c.MCP.Version = "0.1.0"
	}
	if c.Univelop.DefaultBaseURL == "" {
		c.Univelop.DefaultBaseURL = DefaultBaseURL
	}
	if c.Univelop.Timeout.Duration <= 0 {
		c.Univelop.Timeout.Duration = DefaultTimeout
	}
	if c.Univelop.MaxRetries == 0 {
		c.Univelop.MaxRetries = DefaultMaxRetries
	}
	if c.Univelop.MaxRetries < 0 {
		c.Univelop.MaxRetries = 0
	}
}

// Validate checks the configuration for correctness.
func (c *Config) Validate() error {
	if c.MCP.Port < 1 || c.MCP.Port > 65535 {
		return fmt.Errorf("mcp.port must be between 1 and 65535, got %d", c.MCP.Port)
	}
	if len(c.Workspaces) == 0 {
		return fmt.Errorf("at least one workspace must be configured")
	}
	hasCert := c.MCP.CertFile != ""
	hasKey := c.MCP.KeyFile != ""
	if hasCert != hasKey {
		return fmt.Errorf("both mcp.cert_file and mcp.key_file must be provided together")
	}

	for alias, ws := range c.Workspaces {
		if alias == "" {
			return fmt.Errorf("workspace map keys must be non-empty")
		}
		if ws.APIKey == "" {
			return fmt.Errorf("workspace %q: api_key is required", alias)
		}
		baseURL := ws.ResolvedBaseURL(c.Univelop.DefaultBaseURL)
		u, err := url.Parse(baseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("workspace %q: invalid base_url %q", alias, baseURL)
		}
	}
	return nil
}

// ConfigFileDefault returns the default config file path.
const ConfigFileDefault = "config.yaml"

func (c *Config) String() string {
	return fmt.Sprintf("MCP server %q on :%d, %d workspace(s)", c.MCP.Name, c.MCP.Port, len(c.Workspaces))
}

// Timeout returns the configured HTTP client timeout.
func (c *Config) Timeout() time.Duration {
	if c.Univelop.Timeout.Duration <= 0 {
		return DefaultTimeout
	}
	return c.Univelop.Timeout.Duration
}

// HasTLS returns true if TLS cert and key file paths are configured.
func (c *Config) HasTLS() bool {
	return c.MCP.CertFile != "" && c.MCP.KeyFile != ""
}

// Summary returns a printable config summary.
func (c *Config) Summary() string {
	s := fmt.Sprintf("Port: %d\n", c.MCP.Port)
	if c.HasTLS() {
		s += fmt.Sprintf("TLS: %s / %s\n", c.MCP.CertFile, c.MCP.KeyFile)
	} else {
		s += "TLS: (none — use --dev-tls or --stdio)\n"
	}
	s += fmt.Sprintf("Workspaces: %d\n", len(c.Workspaces))
	for alias, ws := range c.Workspaces {
		label := ws.Label
		if label == "" {
			label = alias
		}
		base := ws.ResolvedBaseURL(c.Univelop.DefaultBaseURL)
		s += fmt.Sprintf("  %s (%s) → %s\n", label, alias, base)
	}
	return s
}