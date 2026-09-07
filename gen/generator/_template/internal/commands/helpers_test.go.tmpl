package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadJSONFile_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := os.WriteFile(path, []byte(`{"name":"test","count":42}`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := readJSONFile(path)
	if err != nil {
		t.Fatalf("readJSONFile() error = %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("name = %v, want test", result["name"])
	}
	if result["count"] != float64(42) {
		t.Errorf("count = %v, want 42", result["count"])
	}
}

func TestReadJSONFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{not valid`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := readJSONFile(path)
	if err == nil {
		t.Error("readJSONFile() expected error for invalid JSON")
	}
}

func TestReadJSONFile_NotFound(t *testing.T) {
	_, err := readJSONFile("/nonexistent/file.json")
	if err == nil {
		t.Error("readJSONFile() expected error for missing file")
	}
}
