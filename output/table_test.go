package output

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const longURL = "https://spinkart.neetocal.com/meeting-with-oliver-smith?one_off=eyJhbGciOiJIUzI1NiJ9.eyJtZWV0aW5nX2lkIjoxMjM0NTZ9.abcdefghijklmnop"

func columns(t *testing.T, raw string) []string {
	t.Helper()
	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		t.Fatalf("bad sample json: %v", err)
	}
	pr, _ := newTestPrinter()
	return pr.pickColumns(rows)
}

func TestPickColumns_PriorityFieldsLeadThenAlphabetical(t *testing.T) {
	got := columns(t, `[{"title":"t","slug":"s","email":"e","zebra":"z","apple":"a","nested":{"x":1}}]`)
	want := []string{"title", "email", "slug", "apple", "zebra"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("pickColumns() = %v, want %v", got, want)
	}
}

func TestPickColumns_IDFollowsSID(t *testing.T) {
	got := columns(t, `[{"sid":"short-id","id":"a1b2c3","name":"Oliver Smith"}]`)

	sidIdx, idIdx := slices.Index(got, "sid"), slices.Index(got, "id")
	if sidIdx == -1 || idIdx == -1 {
		t.Fatalf("pickColumns = %v, want both sid and id present", got)
	}
	if sidIdx > idIdx {
		t.Errorf("pickColumns = %v, want sid before id", got)
	}
}

func TestPickColumns_IncludesIDAlongsideCountsAndTimestamps(t *testing.T) {
	got := columns(t, `[{"id":"a1b2c3","name":"Oliver Smith","email":"oliver@example.com","status":"confirmed",
		"duration":0,"bookings_remaining":5,"created_at":"2026-06-25T02:34:46.708Z","duration_remaining":null}]`)

	if !slices.Contains(got, "id") {
		t.Errorf("pickColumns = %v, want id to be included", got)
	}
}

func TestPickColumns_NoPriorityMatchFallsBackToSortedScalars(t *testing.T) {
	got := columns(t, `[{"zebra":1,"apple":2,"mango":3}]`)
	want := []string{"apple", "mango", "zebra"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("pickColumns() = %v, want %v", got, want)
	}
}

func TestPickColumns_EmptyPayloadHasNoColumns(t *testing.T) {
	if got := columns(t, `[{}]`); len(got) != 0 {
		t.Errorf("pickColumns(empty) = %v, want no columns", got)
	}
}

func TestPickColumns_NeverExceedsTheColumnBudget(t *testing.T) {
	got := columns(t, `[{"a":1,"b":2,"c":3,"d":4,"e":5,"f":6,"g":7,"h":8,"i":9,"j":10}]`)

	if len(got) > maxTableColumns {
		t.Errorf("pickColumns returned %d columns, want at most %d", len(got), maxTableColumns)
	}
}

func TestPickColumns_KeepsScalarSlicesAndDropsObjectSlices(t *testing.T) {
	got := columns(t, `[{"name":"Acme","tags":["a","b"],"empty":[],"client":{"id":"1"},"nested":[{"k":"v"}]}]`)

	if !slices.Contains(got, "tags") {
		t.Errorf("pickColumns = %v, want the scalar slice tags to be a column", got)
	}
	if slices.Contains(got, "empty") {
		t.Errorf("pickColumns = %v, want a slice that is empty in every row to be dropped as noise", got)
	}
	for _, col := range got {
		if col == "client" || col == "nested" {
			t.Errorf("pickColumns = %v, want no column for the non-scalar %q", got, col)
		}
	}
}

func TestPickColumns_URLColumnsSortLastAndSurviveTheBudget(t *testing.T) {
	got := columns(t, `[{"sid":"a1","id":"x","name":"Intro","title":"t","email":"e@example.com",
		"status":"active","kind":"one_on_one","url":"https://spinkart.neetocal.com/intro",
		"avatar_url":"https://cdn.example.com/a.png"}]`)

	if len(got) < 2 {
		t.Fatalf("pickColumns = %v, want columns", got)
	}
	if tail := got[len(got)-2:]; !reflect.DeepEqual(tail, []string{"avatar_url", "url"}) {
		t.Errorf("pickColumns = %v, want the URL columns sorted last", got)
	}
}

func TestPickColumns_DetectsURLColumnsInLaterRows(t *testing.T) {
	got := columns(t, `[{"id":"1","title":"Customer feedback","attempt_url":null},
		{"id":"2","title":"Onboarding","attempt_url":"https://acme.neetoform.com/0c1d2e"}]`)

	if joined := strings.Join(got, " "); joined != "id title attempt_url" {
		t.Errorf("pickColumns() = %q, want %q", joined, "id title attempt_url")
	}
}

func TestPickColumns_RealResourcePayloads(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    []string
	}{
		{
			name: "articles list",
			payload: `[{"id":"a1","slug":"getting-started","title":"Getting Started","state":"published",
				"unique_views_count":42,"category":{"id":"c1","name":"Guides"}}]`,
			want: []string{"id", "title", "state", "slug", "unique_views_count"},
		},
		{
			name: "team members list",
			payload: `[{"id":"t1","email":"oliver@example.com","first_name":"Oliver","last_name":"Smith",
				"time_zone":"Asia/Kolkata","profile_image_url":null,"active":true,"organization_role":"Admin"}]`,
			want: []string{"id", "email", "first_name", "last_name", "active", "organization_role", "profile_image_url"},
		},
		{
			name:    "submissions list",
			payload: `[{"id":"f1c0d5e2","created_at":"2026-02-14T11:05:31Z"}]`,
			want:    []string{"id", "created_at"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := columns(t, tc.payload); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("pickColumns() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTableUninformative_OnlyWhenColumnsHideNestedData(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    bool
	}{
		{"two columns hiding an object", `[{"id":1,"title":"t","owner":{"name":"Oliver Smith"}}]`, true},
		{"three columns", `[{"id":1,"name":"n","note":"x","owner":{"name":"Oliver Smith"}}]`, false},
		{"two columns, nothing hidden", `[{"id":1,"name":"Only two"}]`, false},
		{"two columns, empty containers only", `[{"id":1,"name":"n","extras":{}}]`, false},
		{"scalar slice is not nested data", `[{"id":1,"tags":["a","b"]}]`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rows []map[string]interface{}
			if err := json.Unmarshal([]byte(tc.payload), &rows); err != nil {
				t.Fatalf("bad sample json: %v", err)
			}
			pr, _ := newTestPrinter()
			if got := pr.tableUninformative(rows); got != tc.want {
				t.Errorf("tableUninformative() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPrintTable_JoinsScalarSlicesIntoOneColumn(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`[{"name":"Desk","roles":["Admin","Agent"]},{"name":"KB","roles":["Admin","Collaborator","Editor"]}]`))

	out := buf.String()
	for _, want := range []string{"───", "ROLES", "Admin, Agent", "Admin, Collaborator, Editor"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintTable_EmptySliceRendersAsDash(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`[{"name":"run-a","tags":["smoke"]},{"name":"run-b","tags":[]}]`))

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	last := lines[len(lines)-1]
	if !strings.HasSuffix(strings.TrimRight(last, " "), "-") {
		t.Errorf("row with an empty slice = %q, want it to end in %q", last, "-")
	}
}

func TestPrintTable_IndentedTableStaysInsideTheTerminal(t *testing.T) {
	pr, buf := newTestPrinter()

	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(`[{"name":"`+strings.Repeat("x", 80)+`","note":"`+strings.Repeat("y", 80)+`"}]`), &rows); err != nil {
		t.Fatalf("bad sample json: %v", err)
	}
	pr.printTable(rows, 2)

	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if !strings.HasPrefix(line, "    ") {
			t.Errorf("indented table line is missing its prefix: %q", line)
		}
		if w := displayWidth(line); w > pr.terminalWidth() {
			t.Errorf("line is %d wide, wider than the %d-column terminal: %q", w, pr.terminalWidth(), line)
		}
	}
}

func TestBuildGrid(t *testing.T) {
	got := buildGrid([][]interface{}{
		{"closed", float64(8), nil},
		{"open", float64(3), float64(2)},
	})
	want := [][]string{
		{"closed", "8", "-"},
		{"open", "3", "2"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildGrid() = %#v, want %#v", got, want)
	}
}

func TestRenderGrid(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.renderGrid([]string{"NAME", "PRESENT"}, [][]string{{"closed", "8"}}, 0)

	for _, want := range []string{"NAME", "PRESENT", "closed", "8", "─"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("renderGrid() output %q missing %q", buf.String(), want)
		}
	}
}

func TestCalculateWidths_NeverOverflowsTheTerminal(t *testing.T) {
	pr, _ := newTestPrinter()

	shapes := [][]int{
		{2, 20, 130}, {3, 5, 40, 200}, {2, 2, 2, 150}, {4, 9, 9, 9, 120},
		{200, 200, 200}, {1, 1}, {60, 60}, {5, 300}, {13, 13, 13, 13, 90},
	}

	for _, indent := range []int{0, 2} {
		for _, natural := range shapes {
			headers := make([]string, len(natural))
			row := make([]string, len(natural))
			for i, n := range natural {
				headers[i] = "H"
				row[i] = strings.Repeat("x", n)
			}

			widths := pr.calculateWidths(headers, [][]string{row}, indent)

			line := (len(widths)-1)*colPadding + indent*2
			floored := true
			for i, w := range widths {
				line += w
				if w > minColWidth && w < natural[i] {
					floored = false
				}
			}
			if line > pr.terminalWidth() && !floored {
				t.Errorf("shape %v at indent %d produced widths %v totalling %d, wider than the %d-column terminal",
					natural, indent, widths, line, pr.terminalWidth())
			}
		}
	}
}

func TestTable_URLIsNeverTruncated(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`[{"id":"1","name":"A meeting with a deliberately long name value","url":"` + longURL + `"}]`))

	if !strings.Contains(stripLayout(buf.String()), longURL) {
		t.Errorf("table output = %q, want it to reassemble to the full URL %q", buf.String(), longURL)
	}
}

func TestTable_URLWrapsInsideItsColumnWithoutBreakingLayout(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`[{"name":"` + strings.Repeat("x", 60) + `","url":"` + longURL + `"}]`))

	out := buf.String()
	if strings.Contains(out, "https://spinkart.neetocal.com/meeting-with-oliver-smith?one_off=eyJhbGciOiJIUzI1NiJ9...") {
		t.Error("URL was truncated instead of wrapped")
	}
	if joined := stripLayout(out); !strings.Contains(joined, longURL) {
		t.Errorf("wrapped URL does not reassemble to the full value:\n%s", out)
	}
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if w := displayWidth(line); w > pr.terminalWidth() {
			t.Errorf("line is %d wide, wider than the %d-column terminal: %q", w, pr.terminalWidth(), line)
		}
	}
}

func TestTable_WrappedRowsKeepColumnsAligned(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`[{"sid":"a1","name":"Intro","url":"` + longURL + `"},{"sid":"b2","name":"Short","url":"https://example.com/x"}]`))

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	start := strings.Index(lines[0], "URL")
	if start <= 0 {
		t.Fatalf("could not locate the URL column in:\n%s", buf.String())
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

func TestTable_KeepsAtLeastMinContentWidthWhenTruncating(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`[{"name":"Weekly planning session with the design team","slug":"weekly-planning-session-design","summary":"` + strings.Repeat("y", 300) + `","url":"` + longURL + `"}]`))

	for _, line := range strings.Split(buf.String(), "\n") {
		for _, cell := range strings.Split(line, strings.Repeat(" ", colPadding)) {
			cell = strings.TrimSpace(cell)
			if !strings.HasSuffix(cell, ellipsis) {
				continue
			}
			if kept := displayWidth(strings.TrimSuffix(cell, ellipsis)); kept < minContentWidth {
				t.Errorf("cell %q keeps %d characters, want at least %d", cell, kept, minContentWidth)
			}
		}
	}
}

func TestTruncate(t *testing.T) {
	got := truncate("Réunion avec Zoë — planification", 20)

	for _, r := range got {
		if r == '\uFFFD' {
			t.Errorf("truncate = %q, want valid UTF-8", got)
		}
	}
	if displayWidth(got) != 20 {
		t.Errorf("displayWidth(%q) = %d, want 20", got, displayWidth(got))
	}
	if got := truncate(longURL, 20); got != longURL {
		t.Errorf("truncate = %q, want the URL unchanged", got)
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

func stripLayout(out string) string {
	var b strings.Builder
	for _, line := range strings.Split(out, "\n") {
		b.WriteString(strings.TrimSpace(line))
	}
	return b.String()
}

func TestPickColumns_KeepsAColumnThatIsOnlyEmptyInTheFirstRow(t *testing.T) {
	got := columns(t, `[{"id":1,"tags":[]},{"id":2,"tags":["smoke","fast"]}]`)

	if !slices.Contains(got, "tags") {
		t.Errorf("pickColumns = %v, want tags kept so the second row's values are not dropped", got)
	}
}

func TestPrintTable_FallsBackToJSONWhenNoColumnSurvives(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printTable([]map[string]interface{}{{"owner": map[string]interface{}{"name": "Oliver Smith"}}}, 0)

	if !strings.Contains(buf.String(), "Oliver Smith") {
		t.Errorf("printTable with no scalar column should fall back to JSON:\n%s", buf.String())
	}
}

func TestPickColumns_SliceColumnEarnsItsPlace(t *testing.T) {
	pr := &Printer{PriorityFields: []string{"id", "name"}}

	t.Run("kept when a later row has values", func(t *testing.T) {
		rows := []map[string]interface{}{
			{"id": 1.0, "name": "a", "tags": []interface{}{}},
			{"id": 2.0, "name": "b", "tags": []interface{}{"x"}},
		}
		if !contains(pr.pickColumns(rows), "tags") {
			t.Error("tags must be kept when some row has values")
		}
	})

	t.Run("dropped when empty in every row", func(t *testing.T) {
		rows := []map[string]interface{}{
			{"id": 1.0, "name": "a", "tags": []interface{}{}},
			{"id": 2.0, "name": "b", "tags": []interface{}{}},
		}
		if contains(pr.pickColumns(rows), "tags") {
			t.Error("a column empty in every row is noise")
		}
	})
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func TestPickColumns_DropsAColumnThatHoldsObjectsInALaterRow(t *testing.T) {
	firstRowValues := map[string]string{"null": "null", "a scalar": `"none"`, "an empty list": "[]"}

	for name, first := range firstRowValues {
		t.Run(name, func(t *testing.T) {
			got := columns(t, `[{"sid":"a","name":"Oliver Smith","periods":`+first+`},
				{"sid":"b","name":"Working hours","periods":[{"wday":"monday","start_time":"09:00"}]}]`)

			if slices.Contains(got, "periods") {
				t.Errorf("pickColumns = %v, want periods dropped once a row holds objects", got)
			}
		})
	}
}

func TestPrintTable_NeverLeaksGoMapSyntax(t *testing.T) {
	pr, buf := newTestPrinter()
	period := map[string]interface{}{"wday": "monday", "start_time": "09:00"}

	pr.renderGrid([]string{"NAME", "PERIODS"}, buildGrid([][]interface{}{
		{"Working hours", []interface{}{period, period}},
	}), 0)

	if strings.Contains(buf.String(), "map[") {
		t.Errorf("table rendered Go map syntax:\n%s", buf.String())
	}
}

func TestFormatValue_DescribesObjectsInsteadOfDumpingThem(t *testing.T) {
	object := map[string]interface{}{"wday": "monday", "start_time": "09:00"}

	if got := formatValue(object); strings.Contains(got, "map[") {
		t.Errorf("formatValue(object) = %q, want a description rather than Go syntax", got)
	}
	if got := formatValue([]interface{}{object, object}); got != "(2 items)" {
		t.Errorf("formatValue(objects) = %q, want %q", got, "(2 items)")
	}
}
