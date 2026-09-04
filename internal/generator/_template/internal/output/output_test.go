package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy error: %v", err)
	}
	return buf.String()
}

func TestUseJSON_ForceJSON(t *testing.T) {
	ForceJSON = true
	QuietMode = false
	defer func() { ForceJSON = false }()

	if !UseJSON() {
		t.Error("UseJSON() = false, want true when ForceJSON is set")
	}
}

func TestUseJSON_QuietMode(t *testing.T) {
	ForceJSON = false
	QuietMode = true
	defer func() { QuietMode = false }()

	if !UseJSON() {
		t.Error("UseJSON() = false, want true when QuietMode is set")
	}
}

func TestPrintMessage_JSON(t *testing.T) {
	ForceJSON = true
	QuietMode = false
	defer func() { ForceJSON = false }()

	out := captureStdout(t, func() {
		PrintMessage("hello world")
	})

	var parsed map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["message"] != "hello world" {
		t.Errorf("message = %q, want %q", parsed["message"], "hello world")
	}
}

func TestPrintMessage_Plain(t *testing.T) {
	ForceJSON = false
	QuietMode = false

	out := captureStdout(t, func() {
		PrintMessage("hello world")
	})

	trimmed := strings.TrimSpace(out)
	if !strings.Contains(trimmed, "hello world") {
		t.Errorf("output = %q, want it to contain %q", trimmed, "hello world")
	}
}

func TestPrint_QuietMode(t *testing.T) {
	ForceJSON = false
	QuietMode = true
	defer func() { QuietMode = false }()

	data := json.RawMessage(`[{"id":1}]`)
	out := captureStdout(t, func() {
		Print(data, nil)
	})

	trimmed := strings.TrimSpace(out)
	if trimmed != `[{"id":1}]` {
		t.Errorf("quiet output = %q, want raw data", trimmed)
	}
}

func TestPrint_JSONEnvelope(t *testing.T) {
	ForceJSON = true
	QuietMode = false
	defer func() { ForceJSON = false }()

	data := json.RawMessage(`{"name":"test"}`)
	breadcrumbs := []Breadcrumb{{Label: "details", Command: "app show 1"}}

	out := captureStdout(t, func() {
		Print(data, breadcrumbs)
	})

	var envelope Envelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	var parsed map[string]string
	if err := json.Unmarshal(envelope.Data, &parsed); err != nil {
		t.Fatalf("envelope data is not valid JSON: %v", err)
	}
	if parsed["name"] != "test" {
		t.Errorf("data.name = %q, want %q", parsed["name"], "test")
	}
	if len(envelope.Breadcrumbs) != 1 || envelope.Breadcrumbs[0].Label != "details" {
		t.Errorf("breadcrumbs = %v, want [{details app show 1}]", envelope.Breadcrumbs)
	}
}

func TestPrintWithPagination_QuietMode(t *testing.T) {
	ForceJSON = false
	QuietMode = true
	defer func() { QuietMode = false }()

	data := json.RawMessage(`[{"id":1}]`)
	pagination := json.RawMessage(`{"current_page_number":1,"total_pages":3,"total_records":25}`)

	out := captureStdout(t, func() {
		PrintWithPagination(data, pagination, nil)
	})

	trimmed := strings.TrimSpace(out)
	if trimmed != `[{"id":1}]` {
		t.Errorf("quiet output = %q, want raw data without pagination", trimmed)
	}
}

func TestPrintWithPagination_JSONEnvelope(t *testing.T) {
	ForceJSON = true
	QuietMode = false
	defer func() { ForceJSON = false }()

	data := json.RawMessage(`[{"id":1}]`)
	pagination := json.RawMessage(`{"current_page_number":1,"total_pages":3}`)

	out := captureStdout(t, func() {
		PrintWithPagination(data, pagination, nil)
	})

	var envelope Envelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if envelope.Pagination == nil {
		t.Error("pagination should be present in envelope")
	}
}

func TestPrint_ToonEndsWithNewline(t *testing.T) {
	ToonMode = true
	defer func() { ToonMode = false }()

	out := captureStdout(t, func() {
		Print(json.RawMessage(`{"meeting":{"id":"m1","name":"Demo"}}`), nil)
	})

	want := "data:\n  meeting:\n    id: m1\n    name: Demo\n"
	if out != want {
		t.Errorf("Print toon = %q, want %q", out, want)
	}
}

func TestPrintWithPagination_ToonEmptyListKeepsPaginationOnItsOwnLine(t *testing.T) {
	ToonMode = true
	defer func() { ToonMode = false }()

	out := captureStdout(t, func() {
		PrintWithPagination(
			json.RawMessage(`[]`),
			json.RawMessage(`{"current_page_number":1,"total_pages":1,"total_records":0}`),
			nil,
		)
	})

	want := "data[0]:\npagination:\n  current_page_number: 1\n  total_pages: 1\n  total_records: 0\n"
	if out != want {
		t.Errorf("PrintWithPagination toon = %q, want %q", out, want)
	}
}

func TestPrintWithPagination_ToonRowsDoNotRunIntoPagination(t *testing.T) {
	ToonMode = true
	defer func() { ToonMode = false }()

	out := captureStdout(t, func() {
		PrintWithPagination(
			json.RawMessage(`[{"id":"m1","name":"Demo"},{"id":"m2","name":"Intro"}]`),
			json.RawMessage(`{"current_page_number":1,"total_pages":1,"total_records":2}`),
			nil,
		)
	})

	want := "data[2]{id,name}:\n  m1,Demo\n  m2,Intro\npagination:\n  current_page_number: 1\n  total_pages: 1\n  total_records: 2\n"
	if out != want {
		t.Errorf("PrintWithPagination toon = %q, want %q", out, want)
	}
}

func TestPrint_ToonFallsBackToRawBodyWhenNotJSON(t *testing.T) {
	ToonMode = true
	defer func() { ToonMode = false }()

	out := captureStdout(t, func() {
		Print(json.RawMessage(`<html>oops</html>`), nil)
	})

	if out != "<html>oops</html>\n" {
		t.Errorf("Print toon fallback = %q, want the raw body on its own line", out)
	}
}
