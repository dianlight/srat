package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpenAPIGenerationCreatesFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping openapi generation test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	outDir := t.TempDir()

	probePath := filepath.Join(outDir, "preflight.txt")
	if err := os.WriteFile(probePath, []byte("ok"), 0o600); err != nil {
		t.Fatalf("failed to write probe file to temp dir: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	cmd := exec.CommandContext(ctx, "go", "run", ".", "-out", outDir, "-mock", "-loglevel", "error")
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "SRAT_MOCK=true")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("srat-openapi timed out: %s", string(output))
		}
		t.Fatalf("srat-openapi execution failed: %v\nOutput:\n%s", err, string(output))
	}

	t.Logf("srat-openapi output:\n%s", string(output))

	yamlPath := filepath.Join(outDir, "openapi.yaml")
	jsonPath := filepath.Join(outDir, "openapi.json")

	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("failed to read generated openapi.yaml: %v", err)
	}
	if len(bytes.TrimSpace(yamlData)) == 0 {
		t.Fatalf("generated openapi.yaml is empty")
	}
	if !bytes.Contains(yamlData, []byte("openapi:")) {
		t.Fatalf("generated openapi.yaml missing openapi version field")
	}

	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read generated openapi.json: %v", err)
	}
	if len(bytes.TrimSpace(jsonData)) == 0 {
		t.Fatalf("generated openapi.json is empty")
	}

	var payload map[string]any
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		t.Fatalf("generated openapi.json is not valid JSON: %v", err)
	}

	version, ok := payload["openapi"].(string)
	if !ok || version == "" {
		t.Fatalf("generated openapi.json missing openapi version field: %v", payload["openapi"])
	}
}

func TestOpenAPIFilenames(t *testing.T) {
	yamlPath, jsonPath := openAPIFilenames("./docs/")
	if !strings.HasSuffix(yamlPath, "docs/openapi.yaml") {
		t.Fatalf("unexpected yaml path: %s", yamlPath)
	}
	if !strings.HasSuffix(jsonPath, "docs/openapi.json") {
		t.Fatalf("unexpected json path: %s", jsonPath)
	}

	emptyYaml, emptyJSON := openAPIFilenames("")
	if emptyYaml != "/openapi.yaml" || emptyJSON != "/openapi.json" {
		t.Fatalf("unexpected paths for empty dir: %s %s", emptyYaml, emptyJSON)
	}
}

func TestApplyMockEnv(t *testing.T) {
	t.Setenv("SRAT_MOCK", "")
	applyMockEnv(false)
	if value := os.Getenv("SRAT_MOCK"); value != "" {
		t.Fatalf("expected SRAT_MOCK to be unset, got %q", value)
	}

	applyMockEnv(true)
	if value := os.Getenv("SRAT_MOCK"); value != "true" {
		t.Fatalf("expected SRAT_MOCK to be true, got %q", value)
	}
}
