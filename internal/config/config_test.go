package config

import (
	"os"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	t.Setenv("TEST_KEY", "secret-value")
	result, err := ExpandEnv("hello ${TEST_KEY}")
	if err != nil {
		t.Fatal(err)
	}
	if result != "hello secret-value" {
		t.Fatalf("expected 'hello secret-value', got %q", result)
	}

	// missing var
	_, err = ExpandEnv("${MISSING_VAR_XXXX}")
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
}

func TestExpandEnvMultiple(t *testing.T) {
	t.Setenv("A", "x")
	t.Setenv("B", "y")
	result, err := ExpandEnv("${A}-${B}")
	if err != nil {
		t.Fatal(err)
	}
	if result != "x-y" {
		t.Fatalf("expected 'x-y', got %q", result)
	}
}

func TestExpandEnvNoVars(t *testing.T) {
	result, err := ExpandEnv("plain string with no vars")
	if err != nil {
		t.Fatal(err)
	}
	if result != "plain string with no vars" {
		t.Fatalf("expected input unchanged, got %q", result)
	}
}

func TestLoadSkipsComments(t *testing.T) {
	input := `# ${MISSING_VAR} should be ignored in comments
mcp:
  port: 8443
univelop:
  default_base_url: "https://app.univelop.de/"
  timeout: "30s"
workspaces:
  test:
    workspace_id: "ws-123"
    api_key: "${REAL_VAR}"
    label: "Test"
`
	tmpfile, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.WriteString(input); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	t.Setenv("REAL_VAR", "my-secret")
	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(cfg.Workspaces))
	}
	if cfg.Workspaces["test"].APIKey != "my-secret" {
		t.Fatalf("expected api_key 'my-secret', got %q", cfg.Workspaces["test"].APIKey)
	}
}

func TestStripYAMLComments(t *testing.T) {
	input := `# full line comment
key: value  # inline comment
# another comment
`
	expected := "key: value  # inline comment\n"
	result := stripYAMLComments(input)
	if result != expected {
		t.Fatalf("got %q, want %q", result, expected)
	}
}