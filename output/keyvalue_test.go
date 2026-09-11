package output

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestDecodeFields_KeepsThePayloadOrder(t *testing.T) {
	fields, ok := decodeFields(json.RawMessage(`{"zebra":1,"apple":2,"mango":3}`))
	if !ok {
		t.Fatal("decodeFields returned not-ok for a valid object")
	}

	var keys []string
	for _, f := range fields {
		keys = append(keys, f.key)
	}
	if strings.Join(keys, ",") != "zebra,apple,mango" {
		t.Errorf("decodeFields keys = %v, want the payload order", keys)
	}
}

func TestDecodeFields_RejectsNonObjects(t *testing.T) {
	for _, raw := range []string{`[1,2]`, `"x"`, `not json`} {
		if _, ok := decodeFields(json.RawMessage(raw)); ok {
			t.Errorf("decodeFields(%q) = ok, want not-ok", raw)
		}
	}
}

func TestPrintPretty_SingleKeyWrapperIsUnwrapped(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`{"client":{"name":"Acme Corp","status":"active"}}`))

	out := buf.String()
	if !strings.Contains(out, "Acme Corp") || !strings.Contains(out, "active") {
		t.Errorf("single-key object not unwrapped:\n%s", out)
	}
	if strings.Contains(out, "CLIENT") {
		t.Errorf("single-key object should print inner fields directly, not the wrapper key:\n%s", out)
	}
}

func TestPrintPretty_MultiKeyExpandsNestedObject(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`{"client":{"name":"Acme Corp","identifier":"eb29be2e8c203a71ec1d","status":"active","currency":"USD"},"recipients":[]}`))

	out := buf.String()
	for _, want := range []string{"CLIENT", "Acme Corp", "eb29be2e8c203a71ec1d", "active", "USD"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "fields)") {
		t.Errorf("output collapsed a nested object to a summary instead of expanding it:\n%s", out)
	}
	if !regexp.MustCompile(`(?m)^  RECIPIENTS +-$`).MatchString(out) {
		t.Errorf("an empty array should render as a dash:\n%s", out)
	}
}

func TestPrintPretty_NestedArrayOfObjectsBecomesATable(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`{"client":{"name":"Acme Corp"},"recipients":[
		{"name":"Oliver Smith","email":"oliver.smith@example.com"},
		{"name":"Sam Smith","email":"sam.smith@example.com"}]}`))

	out := buf.String()
	for _, want := range []string{"RECIPIENTS", "Oliver Smith", "oliver.smith@example.com", "Sam Smith", "─"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "items)") {
		t.Errorf("output collapsed a nested array to a summary instead of expanding it:\n%s", out)
	}
}

func TestPrintPretty_NestingStopsAtMaxRenderDepth(t *testing.T) {
	deep := `{"root":{"l1":{"l2":{"l3":{"aaaaaaaaaa":"0123456789","bbbbbbbbbb":"0123456789",
		"cccccccccc":"0123456789","dddddddddd":"0123456789","eeeeeeeeee":"0123456789","ffffffffff":"0123456789"}}}}}`

	pr, buf := newTestPrinter()
	pr.printPretty(json.RawMessage(deep))

	out := buf.String()
	for _, want := range []string{`(?m)^  L1$`, `(?m)^    L2$`, `(?m)^      L3  \(6 fields\)$`} {
		if !regexp.MustCompile(want).MatchString(out) {
			t.Errorf("pattern %q did not match:\n%s", want, out)
		}
	}
	if strings.Contains(out, "AAAAAAAAAA") {
		t.Errorf("rendering went past depth %d:\n%s", maxRenderDepth, out)
	}
}

func TestPrintPretty_LongScalarIsPreviewed(t *testing.T) {
	longContent := "<p>" + strings.Repeat("lorem ipsum ", 40) + "</p>"
	data := json.RawMessage(`{"article":{"title":"Getting started","html_content":` + strconv.Quote(longContent) + `}}`)

	pr, buf := newTestPrinter()
	pr.printPretty(data)

	out := buf.String()
	if !strings.Contains(out, ellipsis) {
		t.Errorf("long content should be previewed with an ellipsis:\n%s", out)
	}
	if strings.Contains(out, strings.Repeat("lorem ipsum ", 12)) {
		t.Errorf("long content should not be printed in full:\n%s", out)
	}
}

func TestPrintPretty_EmptyContainersRenderAsDash(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`{"id":1,"extras":{},"tags":[]}`))

	out := buf.String()
	for _, want := range []string{`(?m)^  EXTRAS +-$`, `(?m)^  TAGS +-$`} {
		if !regexp.MustCompile(want).MatchString(out) {
			t.Errorf("pattern %q did not match:\n%s", want, out)
		}
	}
}

func TestPrintPretty_NestedURLIsNotHiddenBehindAFieldCount(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`{"booking":{"id":"42","status":"scheduled","meeting":{"sid":"abc","name":"Intro call","slug":"intro-call","duration":30,"url":"` + longURL + `"}}}`))

	if !strings.Contains(buf.String(), longURL) {
		t.Errorf("key-value output = %q, want it to contain the nested URL", buf.String())
	}
}

func TestPrintPretty_NestedArrayURLIsNotHidden(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`{"booking":{"id":"42","attachments":[{"name":"` + strings.Repeat("a", 90) + `","url":"` + longURL + `"}]}}`))

	if !strings.Contains(stripLayout(buf.String()), longURL) {
		t.Errorf("key-value output does not reassemble to the nested URL:\n%s", buf.String())
	}
}

func TestPrintPretty_VerboseReplacesTheTableWithRecordBlocks(t *testing.T) {
	pr, buf := newTestPrinter()
	pr.Verbose = true

	pr.printPretty(json.RawMessage(`[{
		"id": "3ad844c3-2785-4ddf-87a4-b48195fc6360",
		"created_at": "2026-02-09T05:20:59.464Z",
		"user_agent": {"name":"Chrome","operating_system":"macOS","ip_address":"1.2.3.4"},
		"responses": [
			{"id":"r1","label":"Email","kind":"email","value":"foo@bar.com"},
			{"id":"r2","label":"Full Name","kind":"text","value":"John Doe"}
		]
	}]`))

	out := buf.String()
	for _, want := range []string{
		"3ad844c3-2785-4ddf-87a4-b48195fc6360",
		"RESPONSES", "Email", "foo@bar.com", "Full Name", "John Doe",
		"USER AGENT", "Chrome", "macOS", "1.2.3.4",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("record block output is missing %q:\n%s", want, out)
		}
	}
	for _, pattern := range []string{
		`(?m)^  RESPONSES$`,
		`(?m)^  USER AGENT$`,
		`(?m)^    Email\b`,
		`(?m)^    Full Name\b`,
		`(?m)^    NAME\b`,
	} {
		if !regexp.MustCompile(pattern).MatchString(out) {
			t.Errorf("block structure not rendered: pattern %q did not match:\n%s", pattern, out)
		}
	}
	if strings.Contains(out, "─") {
		t.Errorf("record blocks must not draw a table separator:\n%s", out)
	}
}

func TestPrintPretty_AnswersBecomeColumnsInsteadOfRecordBlocks(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`[{
		"id": "3ad844c3-2785-4ddf-87a4-b48195fc6360",
		"created_at": "2026-02-09T05:20:59.464Z",
		"user_agent": {"name":"Chrome","operating_system":"macOS","ip_address":"1.2.3.4"},
		"responses": [
			{"id":"r1","label":"Email","kind":"email","value":"foo@bar.com"},
			{"id":"r2","label":"Full Name","kind":"text","value":"John Doe"}
		]
	}]`))

	out := buf.String()
	if !strings.Contains(out, "─") {
		t.Errorf("an answered form should render as a table:\n%s", out)
	}
	for _, want := range []string{"EMAIL", "FULL NAME", "foo@bar.com", "John Doe"} {
		if !strings.Contains(out, want) {
			t.Errorf("table is missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "RESPONSES") {
		t.Errorf("the answer list should be spread across columns, not kept as one:\n%s", out)
	}
}

func TestPrintPretty_RecordBlocksAreSeparatedByABlankLine(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`[{"id":1,"owner":{"name":"Oliver Smith"}},{"id":2,"owner":{"name":"Sam Smith"}}]`))

	if !strings.Contains(buf.String(), "\n\n") {
		t.Errorf("consecutive record blocks should be separated by a blank line:\n%s", buf.String())
	}
}

func TestPrintPretty_RecordBlockLeadsWithTheTableColumns(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`[{"zebra":"z","owner":{"name":"Oliver Smith"},"title":"Contact form"}]`))

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	var labels []string
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
			labels = append(labels, strings.Fields(line)[0])
		}
	}
	if strings.Join(labels, ",") != "TITLE,ZEBRA,OWNER" {
		t.Errorf("record block field order = %v, want the table columns first, then scalars, then nested", labels)
	}
}

func TestPrintPretty_ListsWithEnoughColumnsStayTables(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`[
		{"id":"f1","title":"Contact form","state":"published","submissions_count":3,"created_at":"2026-02-09T05:20:59.464Z"},
		{"id":"f2","title":"Survey","state":"draft","submissions_count":0,"created_at":"2026-02-10T05:20:59.464Z"}]`))

	out := buf.String()
	for _, want := range []string{"TITLE", "─", "Contact form", "Survey"} {
		if !strings.Contains(out, want) {
			t.Errorf("forms list should stay a table; missing %q:\n%s", want, out)
		}
	}
}

func TestPrintPretty_NonJSONBodyIsPrintedRaw(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPretty(json.RawMessage(`<html>oops</html>`))

	if buf.String() != "<html>oops</html>\n" {
		t.Errorf("printPretty = %q, want the raw body on its own line", buf.String())
	}
}

func TestPrintArray_FallsBackToIndentedJSONForScalarLists(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printArray(json.RawMessage(`[1,2,3]`), 1)

	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("indented JSON line is missing its prefix: %q", line)
		}
	}
	if !strings.Contains(buf.String(), "1") {
		t.Errorf("indented JSON is missing the values:\n%s", buf.String())
	}
}

func TestIsLabelValueList(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"labels and values", `[{"label":"Email","value":"a@b.c"}]`, true},
		{"name stands in for label", `[{"name":"Email","value":"a@b.c"}]`, true},
		{"no value key", `[{"label":"Email"}]`, false},
		{"no label key", `[{"value":"a@b.c"}]`, false},
		{"empty", `[]`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var items []interface{}
			if err := json.Unmarshal([]byte(tc.raw), &items); err != nil {
				t.Fatalf("bad sample json: %v", err)
			}
			if got := isLabelValueList(items); got != tc.want {
				t.Errorf("isLabelValueList(%s) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}
