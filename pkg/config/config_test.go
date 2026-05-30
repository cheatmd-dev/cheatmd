package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

func TestEmbeddedTemplateIsValidYAML(t *testing.T) {
	if len(defaultConfigTemplate) == 0 {
		t.Fatal("embedded default config template is empty")
	}
	var m map[string]any
	if err := yaml.Unmarshal(defaultConfigTemplate, &m); err != nil {
		t.Fatalf("embedded template is not valid YAML: %v", err)
	}
	if _, ok := m["registry_url"]; !ok {
		t.Error("embedded template missing registry_url key")
	}
	if _, ok := m["path"]; !ok {
		t.Error("embedded template missing path key")
	}
}

func TestWriteDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "cheatmd.yaml")

	if err := WriteDefaultConfig(path); err != nil {
		t.Fatalf("WriteDefaultConfig: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}
	// The path line must be rewritten to the resolved cheats dir.
	want := "path: " + DefaultCheatsDir()
	if !strings.Contains(string(data), want) {
		t.Errorf("written config missing %q", want)
	}

	// Refuses to overwrite an existing config.
	if err := WriteDefaultConfig(path); err == nil {
		t.Error("expected error writing over existing config, got nil")
	}
}
