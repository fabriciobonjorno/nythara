package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRequiredSecretFromEnvironment(t *testing.T) {
	t.Setenv("TEST_SECRET", " value ")
	value, err := RequiredSecret("TEST_SECRET")
	if err != nil || value != "value" {
		t.Fatalf("value=%q err=%v", value, err)
	}
}

func TestRequiredSecretFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte("file-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEST_SECRET_FILE", path)
	value, err := RequiredSecret("TEST_SECRET")
	if err != nil || value != "file-value" {
		t.Fatalf("value=%q err=%v", value, err)
	}
}

func TestRequiredSecretFailsClosed(t *testing.T) {
	if _, err := RequiredSecret("TEST_SECRET"); err == nil {
		t.Fatal("segredo ausente deveria falhar")
	}
	t.Setenv("TEST_SECRET", "direct")
	t.Setenv("TEST_SECRET_FILE", filepath.Join(t.TempDir(), "secret"))
	if _, err := RequiredSecret("TEST_SECRET"); err == nil {
		t.Fatal("configuração ambígua deveria falhar")
	}
}
