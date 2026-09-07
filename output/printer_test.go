package output

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/neetozone/neeto-cli-commons/config"
)

func newTestPrinter() (*Printer, *bytes.Buffer) {
	var buf bytes.Buffer
	return &Printer{PriorityFields: config.DefaultPriorityFields(), Out: &buf, Err: &buf}, &buf
}

func newPrettyPrinter() (*Printer, *bytes.Buffer) {
	pr, buf := newTestPrinter()
	tty := true
	pr.tty = &tty
	return pr, buf
}

func TestNew_DefaultsWritersAndPriorityFields(t *testing.T) {
	p := config.Product{PrettyName: "NeetoCal"}
	p.ApplyDerivations()

	pr := New(p)

	if pr.Out == nil || pr.Err == nil {
		t.Error("New must default Out and Err")
	}
	if len(pr.PriorityFields) == 0 || pr.PriorityFields[0] != "sid" {
		t.Errorf("PriorityFields = %v, want the product's list", pr.PriorityFields)
	}

	pr.PriorityFields[0] = "mutated"
	if p.PriorityFields[0] != "sid" {
		t.Error("New must copy the product's priority fields, not alias them")
	}
}

func TestNew_FallsBackToDefaultPriorityFields(t *testing.T) {
	pr := New(config.Product{})

	if len(pr.PriorityFields) == 0 {
		t.Error("New must fall back to the default priority fields")
	}
}

var prettyTTY = true

func TestUseJSON(t *testing.T) {
	cases := []struct {
		name string
		pr   Printer
		want bool
	}{
		{"force json", Printer{ForceJSON: true}, true},
		{"quiet", Printer{Quiet: true}, true},
		{"not a terminal", Printer{}, true},
		{"terminal with no flags", Printer{tty: &prettyTTY}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.pr.Out = &bytes.Buffer{}
			if got := tc.pr.UseJSON(); got != tc.want {
				t.Errorf("UseJSON() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPrintMessage(t *testing.T) {
	cases := []struct {
		name string
		mode func(*Printer)
		want string
	}{
		{"json", func(pr *Printer) { pr.ForceJSON = true }, `{"message":"hello world"}`},
		{"quiet", func(pr *Printer) { pr.Quiet = true }, "success"},
		{"toon", func(pr *Printer) { pr.Toon = true }, "hello world"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pr, buf := newTestPrinter()
			tc.mode(pr)

			pr.PrintMessage("hello world")

			if got := strings.TrimSpace(buf.String()); got != tc.want {
				t.Errorf("PrintMessage = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPrint_QuietModeEmitsTheRawBody(t *testing.T) {
	pr, buf := newTestPrinter()
	pr.Quiet = true

	pr.Print(json.RawMessage(`[{"id":1}]`), []Breadcrumb{{Label: "l", Command: "c"}})

	if got := strings.TrimSpace(buf.String()); got != `[{"id":1}]` {
		t.Errorf("quiet output = %q, want the raw data", got)
	}
}

func TestPrint_JSONEnvelopeCarriesBreadcrumbs(t *testing.T) {
	pr, buf := newTestPrinter()
	pr.ForceJSON = true

	pr.Print(json.RawMessage(`{"name":"test"}`), []Breadcrumb{{Label: "details", Command: "app show 1"}})

	var envelope Envelope
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
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
		t.Errorf("breadcrumbs = %v, want one 'details' entry", envelope.Breadcrumbs)
	}
}

func TestPrintWithPagination_JSONEnvelopeCarriesPagination(t *testing.T) {
	pr, buf := newTestPrinter()
	pr.ForceJSON = true

	pr.PrintWithPagination(json.RawMessage(`[{"id":1}]`), json.RawMessage(`{"current_page_number":1,"total_pages":3}`), nil)

	var envelope Envelope
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if envelope.Pagination == nil {
		t.Error("pagination should be present in the envelope")
	}
}

func TestPrintWithPagination_QuietModeDropsPagination(t *testing.T) {
	pr, buf := newTestPrinter()
	pr.Quiet = true

	pr.PrintWithPagination(json.RawMessage(`[{"id":1}]`), json.RawMessage(`{"current_page_number":1}`), nil)

	if got := strings.TrimSpace(buf.String()); got != `[{"id":1}]` {
		t.Errorf("quiet output = %q, want the raw data without pagination", got)
	}
}

func TestPrintQuiet_PrintsOnlyTheIdentifier(t *testing.T) {
	cases := map[string]string{
		`{"sid":"m-001","id":"a1","name":"Intro"}`: "m-001",
		`{"id":"a1","name":"Intro"}`:               "a1",
		`{"meeting":{"sid":"m-002"}}`:              "m-002",
		`{"name":"Intro"}`:                         "Intro",
		`{"email":"a@example.com","role":"owner"}`: "a@example.com",
		`{"name":"Intro","email":"a@example.com"}`: "Intro",
	}

	for payload, want := range cases {
		pr, buf := newTestPrinter()
		pr.Quiet = true

		pr.PrintQuiet(json.RawMessage(payload), nil)

		if got := strings.TrimSpace(buf.String()); got != want {
			t.Errorf("PrintQuiet(%s) = %q, want %q", payload, got, want)
		}
	}
}

func TestPrintQuiet_FallsBackToPrintWhenThereIsNoIdentifier(t *testing.T) {
	pr, buf := newTestPrinter()
	pr.Quiet = true

	pr.PrintQuiet(json.RawMessage(`{"total":3}`), nil)

	if got := strings.TrimSpace(buf.String()); got != `{"total":3}` {
		t.Errorf("PrintQuiet = %q, want the raw body", got)
	}
}

func TestPrintTable_RendersCallerSuppliedColumns(t *testing.T) {
	pr, buf := newPrettyPrinter()

	pr.PrintTable(
		json.RawMessage(`[{"name":"closed","value":{"present":8,"previous":null}}]`),
		[]string{"NAME", "PRESENT", "PREVIOUS"},
		[][]interface{}{{"closed", float64(8), nil}},
		[]Breadcrumb{{Label: "Show", Command: "app show"}},
	)

	out := buf.String()
	for _, want := range []string{"NAME", "PRESENT", "PREVIOUS", "closed", "8", "-", "─", "  Show: app show"} {
		if !strings.Contains(out, want) {
			t.Errorf("PrintTable output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintTable_JSONModePreservesTheRawPayload(t *testing.T) {
	pr, buf := newTestPrinter()
	pr.ForceJSON = true
	raw := json.RawMessage(`[{"name":"closed","value":{"present":8,"previous":null}}]`)

	pr.PrintTable(raw, []string{"NAME", "PRESENT"}, [][]interface{}{{"closed", float64(8)}}, nil)

	var envelope Envelope
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("output is not a valid JSON envelope: %v", err)
	}
	var got, want interface{}
	if err := json.Unmarshal(envelope.Data, &got); err != nil {
		t.Fatalf("envelope data is not valid JSON: %v", err)
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("raw payload is not valid JSON: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("envelope data = %v, want the raw payload %v", got, want)
	}
}

func TestPrintPaginationSummary(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"current_page_number", `{"current_page_number":2,"total_pages":5,"total_records":120}`, "\nPage 2 of 5 (120 total records)\n"},
		{"current_page", `{"current_page":3,"total_pages":5,"total_records":120}`, "\nPage 3 of 5 (120 total records)\n"},
		{"neither", `{"total_pages":5,"total_records":120}`, "\nPage 0 of 5 (120 total records)\n"},
		{"unparseable", `not json`, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pr, buf := newTestPrinter()

			pr.printPaginationSummary(json.RawMessage(tc.raw))

			if buf.String() != tc.want {
				t.Errorf("printPaginationSummary = %q, want %q", buf.String(), tc.want)
			}
		})
	}
}

func TestPrintPaginationSummary_NilPrintsNothing(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printPaginationSummary(nil)

	if buf.String() != "" {
		t.Errorf("printPaginationSummary(nil) = %q, want nothing", buf.String())
	}
}

func TestPrintBreadcrumbs(t *testing.T) {
	pr, buf := newTestPrinter()

	pr.printBreadcrumbs([]Breadcrumb{{Label: "Show", Command: "app show 1"}, {Label: "Update", Command: "app update 1"}})

	want := "\n  Show: app show 1\n  Update: app update 1\n"
	if buf.String() != want {
		t.Errorf("printBreadcrumbs = %q, want %q", buf.String(), want)
	}
}

func TestPrint_Toon(t *testing.T) {
	cases := []struct {
		name       string
		data       string
		pagination string
		want       string
	}{
		{
			name: "object",
			data: `{"meeting":{"id":"m1","name":"Demo"}}`,
			want: "data:\n  meeting:\n    id: m1\n    name: Demo\n",
		},
		{
			name:       "empty list keeps pagination on its own line",
			data:       `[]`,
			pagination: `{"current_page_number":1,"total_pages":1,"total_records":0}`,
			want:       "data[0]:\npagination:\n  current_page_number: 1\n  total_pages: 1\n  total_records: 0\n",
		},
		{
			name:       "rows do not run into pagination",
			data:       `[{"id":"m1","name":"Demo"},{"id":"m2","name":"Intro"}]`,
			pagination: `{"current_page_number":1,"total_pages":1,"total_records":2}`,
			want:       "data[2]{id,name}:\n  m1,Demo\n  m2,Intro\npagination:\n  current_page_number: 1\n  total_pages: 1\n  total_records: 2\n",
		},
		{
			name: "non-JSON falls back to the raw body",
			data: `<html>oops</html>`,
			want: "<html>oops</html>\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pr, buf := newTestPrinter()
			pr.Toon = true

			var pagination json.RawMessage
			if tc.pagination != "" {
				pagination = json.RawMessage(tc.pagination)
			}
			pr.PrintWithPagination(json.RawMessage(tc.data), pagination, []Breadcrumb{{Label: "l", Command: "c"}})

			if buf.String() != tc.want {
				t.Errorf("toon output = %q, want %q", buf.String(), tc.want)
			}
		})
	}
}

func TestPrintPretty_Fixtures(t *testing.T) {
	crumbs := []Breadcrumb{{Label: "Show details", Command: "neetocal meetings show <sid>"}}

	cases := []struct {
		name       string
		data       string
		pagination string
		want       string
	}{
		{
			name: "flat list",
			data: `[{"id":1,"sid":"m-001","name":"Weekly sync","status":"active","duration":30,"email":"a@example.com"},
				{"id":2,"sid":"m-002","name":"A very long meeting name that will certainly need truncating in a narrow table","status":"archived","duration":60,"email":"bb@example.com"}]`,
			pagination: `{"total_records":2,"total_pages":1,"current_page_number":1,"page_size":25}`,
			want: "SID     ID   NAME                                                EMAIL           STATUS     DURATION\n" +
				"─────   ──   ─────────────────────────────────────────────────   ─────────────   ────────   ────────\n" +
				"m-001   1    Weekly sync                                         a@example.com   active     30\n" +
				"m-002   2    A very long meeting name that will certainly n...   bb@example...   archived   60\n" +
				"\nPage 1 of 1 (2 total records)\n" +
				"\n  Show details: neetocal meetings show <sid>\n",
		},
		{
			name: "nested list falls back to record blocks",
			data: `[{"id":1,"title":"Contact form","owner":{"name":"Oliver Smith","email":"o@example.com"},
				"settings":{"public":true,"limits":{"per_day":10,"per_month":100}},
				"responses":[{"label":"Name","value":"Sam"},{"label":"Email","value":"sam@example.com"}]}]`,
			want: "  ID     1\n" +
				"  TITLE  Contact form\n" +
				"  OWNER\n" +
				"    NAME   Oliver Smith\n" +
				"    EMAIL  o@example.com\n" +
				"  RESPONSES\n" +
				"    Name   Sam\n" +
				"    Email  sam@example.com\n" +
				"  SETTINGS\n" +
				"    PUBLIC  Yes\n" +
				"    LIMITS\n" +
				"      PER DAY    10\n" +
				"      PER MONTH  100\n" +
				"\n  Show details: neetocal meetings show <sid>\n",
		},
		{
			name: "scalar arrays become columns",
			data: `[{"id":1,"name":"run-a","tags":["smoke","fast","ci"],"trace_urls":["https://trace.example.com/1"],"empty":[]},
				{"id":2,"name":"run-b","tags":[],"trace_urls":[],"empty":[]}]`,
			want: "ID   NAME    TAGS              TRACE URLS\n" +
				"──   ─────   ───────────────   ───────────────────────────\n" +
				"1    run-a   smoke, fast, ci   https://trace.example.com/1\n" +
				"2    run-b   -                 -\n" +
				"\n  Show details: neetocal meetings show <sid>\n",
		},
		{
			name: "single nested record",
			data: `{"id":42,"sid":"inv-042","name":"Acme Corp","total":1250.5,"currency":"USD",
				"client":{"name":"Acme","contact":{"email":"ap@acme.com","phone":"+1 555 0100"}},
				"line_items":[{"description":"Consulting","hours":10,"rate":125.05}],"notes":null,"paid":false}`,
			want: "  ID        42\n" +
				"  SID       inv-042\n" +
				"  NAME      Acme Corp\n" +
				"  TOTAL     1250.50\n" +
				"  CURRENCY  USD\n" +
				"  CLIENT\n" +
				"    NAME  Acme\n" +
				"    CONTACT\n" +
				"      EMAIL  ap@acme.com\n" +
				"      PHONE  +1 555 0100\n" +
				"  LINE ITEMS\n" +
				"    DESCRIPTION   HOURS   RATE\n" +
				"    ───────────   ─────   ──────\n" +
				"    Consulting    10      125.05\n" +
				"  NOTES     -\n" +
				"  PAID      No\n" +
				"\n  Show details: neetocal meetings show <sid>\n",
		},
		{
			name: "urls",
			data: `[{"id":1,"name":"Demo recording","url":"https://acme.neetorecord.com/watch/abc123def456",
				"thumbnail_url":"https://cdn.example.com/very/long/path/to/a/thumbnail/image/file.png","duration":95}]`,
			want: "ID   NAME            DURATION   THUMBNAIL URL                          URL\n" +
				"──   ─────────────   ────────   ────────────────────────────────────   ─────────────────────────────\n" +
				"1    Demo recor...   95         https://cdn.example.com/very/long/pa   https://acme.neetorecord.com/\n" +
				"                                th/to/a/thumbnail/image/file.png       watch/abc123def456\n" +
				"\n  Show details: neetocal meetings show <sid>\n",
		},
		{
			name:       "empty list",
			data:       `[]`,
			pagination: `{"total_records":0,"total_pages":0,"current_page_number":1,"page_size":25}`,
			want: "No records found.\n" +
				"\nPage 1 of 0 (0 total records)\n" +
				"\n  Show details: neetocal meetings show <sid>\n",
		},
		{
			name: "two columns with nothing hidden",
			data: `[{"id":1,"name":"Only two","note":"cols"}]`,
			want: "ID   NAME       NOTE\n" +
				"──   ────────   ────\n" +
				"1    Only two   cols\n" +
				"\n  Show details: neetocal meetings show <sid>\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pr, buf := newPrettyPrinter()

			var pagination json.RawMessage
			if tc.pagination != "" {
				pagination = json.RawMessage(tc.pagination)
			}
			pr.PrintWithPagination(json.RawMessage(tc.data), pagination, crumbs)

			if buf.String() != tc.want {
				t.Errorf("rendered output differs.\n--- got ---\n%s\n--- want ---\n%s", buf.String(), tc.want)
			}
		})
	}
}

func TestPrint_PrettyPathRendersATableAndBreadcrumbs(t *testing.T) {
	pr, buf := newPrettyPrinter()

	pr.Print(json.RawMessage(`[{"id":1,"name":"Only two","note":"cols"}]`), []Breadcrumb{{Label: "Show", Command: "app show 1"}})

	want := "ID   NAME       NOTE\n" +
		"──   ────────   ────\n" +
		"1    Only two   cols\n" +
		"\n  Show: app show 1\n"
	if buf.String() != want {
		t.Errorf("Print = %q, want %q", buf.String(), want)
	}
}

func TestPrintTable_QuietAndToonModes(t *testing.T) {
	raw := json.RawMessage(`[{"name":"closed","value":8}]`)

	cases := []struct {
		name string
		mode func(*Printer)
		want string
	}{
		{"quiet", func(pr *Printer) { pr.Quiet = true }, `[{"name":"closed","value":8}]` + "\n"},
		{"toon", func(pr *Printer) { pr.Toon = true }, "data[1]{name,value}:\n  closed,8\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pr, buf := newPrettyPrinter()
			tc.mode(pr)

			pr.PrintTable(raw, []string{"NAME", "VALUE"}, [][]interface{}{{"closed", float64(8)}}, nil)

			if buf.String() != tc.want {
				t.Errorf("PrintTable = %q, want %q", buf.String(), tc.want)
			}
		})
	}
}

func TestPrintQuiet_NonQuietModeFallsThroughToPrint(t *testing.T) {
	pr, buf := newPrettyPrinter()

	pr.PrintQuiet(json.RawMessage(`{"sid":"m-001","name":"Intro"}`), nil)

	want := "  SID   m-001\n  NAME  Intro\n"
	if buf.String() != want {
		t.Errorf("PrintQuiet = %q, want %q", buf.String(), want)
	}
}

func TestPrintPaginationSummaryAcceptsEitherTotalKey(t *testing.T) {
	cases := map[string]string{
		`{"current_page_number":2,"total_pages":5,"total_records":42}`: "Page 2 of 5 (42 total records)",
		`{"page":2,"total_pages":5,"total_count":42}`:                  "Page 2 of 5 (42 total records)",
		`{"current_page":3,"total_pages":5,"total_count":42}`:          "Page 3 of 5 (42 total records)",
	}

	for payload, want := range cases {
		pr, buf := newPrettyPrinter()
		pr.printPaginationSummary(json.RawMessage(payload))

		if got := strings.TrimSpace(buf.String()); got != want {
			t.Errorf("payload %s = %q, want %q", payload, got, want)
		}
	}
}

func TestIntFromPrefersPresenceOverNonZero(t *testing.T) {
	cases := []struct {
		body string
		keys []string
		want int
	}{
		{`{"total_records":0,"total_count":250}`, []string{"total_records", "total_count"}, 0},
		{`{"current_page_number":0,"page":3}`, []string{"current_page_number", "current_page", "page"}, 0},
		{`{"total_count":250}`, []string{"total_records", "total_count"}, 250},
		{`{"total_pages":12.0}`, []string{"total_pages"}, 12},
		{`{}`, []string{"total_pages"}, 0},
		{`{"total_pages":"nope"}`, []string{"total_pages"}, 0},
	}

	for _, c := range cases {
		var parsed map[string]json.RawMessage
		if err := json.Unmarshal([]byte(c.body), &parsed); err != nil {
			t.Fatal(err)
		}
		if got := IntFrom(parsed, c.keys...); got != c.want {
			t.Errorf("IntFrom(%s, %v) = %d, want %d", c.body, c.keys, got, c.want)
		}
	}
}

func TestPrintPaginationSummaryReportsAnEmptyPageAsEmpty(t *testing.T) {
	pr, buf := newPrettyPrinter()

	pr.printPaginationSummary(json.RawMessage(`{"current_page_number":0,"total_pages":0,"total_records":0,"total_count":250}`))

	want := "Page 0 of 0 (0 total records)"
	if got := strings.TrimSpace(buf.String()); got != want {
		t.Errorf("summary = %q, want %q; a present zero must beat a later key", got, want)
	}
}
