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

	if !strings.Contains(stripLayout(out), longURL) {
		t.Errorf("table output = %q, want it to reassemble to the full URL %q", out, longURL)
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

func TestTable_URLWrapsInsideItsColumnWithoutBreakingLayout(t *testing.T) {
	resetModes()

	data := json.RawMessage(`[{"name":"` + strings.Repeat("x", 60) + `","url":"` + longURL + `"}]`)
	out := captureStdout(t, func() { printPretty(data) })

	if strings.Contains(out, "https://spinkart.neetocal.com/meeting-with-oliver-smith?one_off=eyJhbGciOiJIUzI1NiJ9...") {
		t.Error("URL was truncated instead of wrapped")
	}
	if joined := stripLayout(out); !strings.Contains(joined, longURL) {
		t.Errorf("wrapped URL does not reassemble to the full value:\n%s", out)
	}
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if w := displayWidth(line); w > getTerminalWidth() {
			t.Errorf("line is %d wide, wider than the %d-column terminal: %q", w, getTerminalWidth(), line)
		}
	}
}

func TestTable_WrappedRowsKeepColumnsAligned(t *testing.T) {
	resetModes()

	data := json.RawMessage(`[{"sid":"a1","name":"Intro","url":"` + longURL + `"},{"sid":"b2","name":"Short","url":"https://example.com/x"}]`)
	out := captureStdout(t, func() { printPretty(data) })

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	start := strings.Index(lines[0], "URL")
	if start <= 0 {
		t.Fatalf("could not locate the URL column in:\n%s", out)
	}
	for _, line := range lines[2:] {
		if displayWidth(line) <= start {
			continue
		}
		if r := []rune(line)[start-1]; r != ' ' {
			t.Errorf("column boundary at %d is not padding in %q", start, line)
		}
	}
}

func stripLayout(out string) string {
	var b strings.Builder
	for _, line := range strings.Split(out, "\n") {
		b.WriteString(strings.TrimSpace(line))
	}
	return b.String()
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

func TestIsURL_SchemeIsCaseInsensitive(t *testing.T) {
	for _, s := range []string{"https://example.com", "HTTPS://EXAMPLE.COM", "Http://Example.com"} {
		if !isURL(s) {
			t.Errorf("isURL(%q) = false, want true", s)
		}
	}
	if isURL("nothttp://example.com") {
		t.Error("isURL matched a string that does not start with a scheme")
	}
}

func TestTable_URLColumnFoundWhenAbsentFromFirstRow(t *testing.T) {
	resetModes()

	data := json.RawMessage(`[{"id":"1","name":"no link"},{"id":"2","name":"has link","url":"` + longURL + `"}]`)
	out := captureStdout(t, func() { printPretty(data) })

	if !strings.Contains(stripLayout(out), longURL) {
		t.Errorf("table output = %q, want the URL discovered on a later row", out)
	}
}

func TestKeyValue_NestedArrayURLIsNotHidden(t *testing.T) {
	resetModes()

	data := json.RawMessage(`{"booking":{"id":"42","attachments":[{"name":"` + strings.Repeat("a", 90) + `","url":"` + longURL + `"}]}}`)
	out := captureStdout(t, func() { printPretty(data) })

	if !strings.Contains(out, longURL) {
		t.Errorf("key-value output = %q, want the URL inside the nested array", out)
	}
}
func TestCalculateWidths_NeverOverflowsTheTerminal(t *testing.T) {
	shapes := [][]int{
		{2, 20, 130}, {3, 5, 40, 200}, {2, 2, 2, 150}, {4, 9, 9, 9, 120},
		{200, 200, 200}, {1, 1}, {60, 60}, {5, 300}, {13, 13, 13, 13, 90},
	}

	for _, natural := range shapes {
		headers := make([]string, len(natural))
		row := make([]string, len(natural))
		for i, n := range natural {
			headers[i] = "H"
			row[i] = strings.Repeat("x", n)
		}

		widths := calculateWidths(headers, [][]string{row})

		line := (len(widths) - 1) * colPadding
		floored := true
		for i, w := range widths {
			line += w
			if w > minColWidth && w < natural[i] {
				floored = false
			}
		}
		if line > getTerminalWidth() && !floored {
			t.Errorf("shape %v produced widths %v totalling %d, wider than the %d-column terminal",
				natural, widths, line, getTerminalWidth())
		}
	}
}
