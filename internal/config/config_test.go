package config

import "testing"

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