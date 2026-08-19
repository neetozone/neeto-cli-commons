package output

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

const longURL = "https://spinkart.neetocal.com/meeting-with-oliver-smith?one_off=eyJhbGciOiJIUzI1NiJ9.eyJtZWV0aW5nX2lkIjoxMjM0NTZ9.abcdefghijklmnop"

func resetModes() {
	ForceJSON, QuietMode, ToonMode = false, false, false
}

func TestTable_URLIsNeverTruncated(t *testing.T) {
	resetModes()

	data := json.RawMessage(`[{"id":"1","name":"A meeting with a deliberately long name value","url":"` + longURL + `"}]`)
	out := captureStdout(t, func() { printPretty(data) })

	if !strings.Contains(out, longURL) {
		t.Errorf("table output = %q, want it to contain the full URL %q", out, longURL)
	}
}

func TestTable_URLColumnSurvivesColumnCap(t *testing.T) {
	resetModes()

	sample := map[string]interface{}{
		"sid": "a1", "id": "x", "name": "Intro", "title": "t",
		"email": "e@example.com", "status": "active", "kind": "one_on_one",
		"url": "https://spinkart.neetocal.com/intro",
	}

	cols := pickColumns([]map[string]interface{}{sample})

	if !slices.Contains(cols, "url") {
		t.Errorf("pickColumns = %v, want url to be included", cols)
	}
}

func TestTable_URLColumnKeepsFullWidthWhenTableOverflows(t *testing.T) {
	resetModes()

	data := json.RawMessage(`[{"name":"` + strings.Repeat("x", 200) + `","url":"` + longURL + `"}]`)
	out := captureStdout(t, func() { printPretty(data) })

	if !strings.Contains(out, longURL) {
		t.Errorf("table output = %q, want the URL column to keep its full width", out)
	}
	if !strings.Contains(out, "...") {
		t.Error("table output should still truncate the non-URL column")
	}
}

func TestTruncate_DoesNotSplitMultibyteRunes(t *testing.T) {
	got := truncate("Réunion avec Zoë — planification", 20)

	if !utf8Valid(got) {
		t.Errorf("truncate = %q, want valid UTF-8", got)
	}
	if displayWidth(got) != 20 {
		t.Errorf("displayWidth(%q) = %d, want 20", got, displayWidth(got))
	}
}

func TestTruncate_LeavesURLsIntact(t *testing.T) {
	if got := truncate(longURL, 20); got != longURL {
		t.Errorf("truncate = %q, want the URL unchanged", got)
	}
}

func TestKeyValue_NestedURLIsNotHiddenBehindFieldCount(t *testing.T) {
	resetModes()

	data := json.RawMessage(`{"booking":{"id":"42","status":"scheduled","meeting":{"sid":"abc","name":"Intro call","slug":"intro-call","duration":30,"url":"` + longURL + `"}}}`)
	out := captureStdout(t, func() { printPretty(data) })

	if !strings.Contains(out, longURL) {
		t.Errorf("key-value output = %q, want it to contain the nested URL", out)
	}
}

func utf8Valid(s string) bool {
	for _, r := range s {
		if r == '�' {
			return false
		}
	}
	return true
}

func TestTruncate_KeepsAtLeastMinContentWidth(t *testing.T) {
	resetModes()

	data := json.RawMessage(`[{"name":"Weekly planning session with the design team","slug":"weekly-planning-session-design","summary":"` + strings.Repeat("y", 300) + `","url":"` + longURL + `"}]`)
	out := captureStdout(t, func() { printPretty(data) })

	for _, line := range strings.Split(out, "\n") {
		for _, cell := range strings.Split(line, strings.Repeat(" ", colPadding)) {
			cell = strings.TrimSpace(cell)
			if !strings.HasSuffix(cell, ellipsis) {
				continue
			}
			kept := displayWidth(strings.TrimSuffix(cell, ellipsis))
			if kept < minContentWidth {
				t.Errorf("cell %q keeps %d characters, want at least %d", cell, kept, minContentWidth)
			}
		}
	}
}
