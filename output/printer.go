package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/alpkeskin/gotoon"
	"golang.org/x/term"

	"github.com/neetozone/neeto-cli-commons/config"
)

type Breadcrumb struct {
	Label   string `json:"label"`
	Command string `json:"command"`
}

type Envelope struct {
	Data        json.RawMessage `json:"data"`
	Breadcrumbs []Breadcrumb    `json:"breadcrumbs,omitempty"`
	Pagination  json.RawMessage `json:"pagination,omitempty"`
}

type Printer struct {
	ForceJSON bool
	Quiet     bool
	Toon      bool
	Verbose   bool

	PriorityFields []string

	Out io.Writer
	Err io.Writer

	tty *bool
}

func New(p config.Product) *Printer {
	fields := p.PriorityFields
	if len(fields) == 0 {
		fields = config.DefaultPriorityFields()
	}
	return &Printer{
		PriorityFields: append([]string(nil), fields...),
		Out:            os.Stdout,
		Err:            os.Stderr,
	}
}

func (pr *Printer) w() io.Writer {
	if pr.Out == nil {
		return os.Stdout
	}
	return pr.Out
}

func (pr *Printer) errw() io.Writer {
	if pr.Err == nil {
		return os.Stderr
	}
	return pr.Err
}

func (pr *Printer) priorityFields() []string {
	if len(pr.PriorityFields) == 0 {
		return config.DefaultPriorityFields()
	}
	return pr.PriorityFields
}

func IsTTY() bool { return isTerminal(os.Stdout) }

func TerminalWidth() int {
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		return width
	}
	return fallbackWidth
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func (pr *Printer) UseJSON() bool {
	return pr.ForceJSON || pr.Quiet || !pr.isTTY()
}

func (pr *Printer) isTTY() bool {
	if pr.tty != nil {
		return *pr.tty
	}
	return isTerminal(pr.w())
}

func (pr *Printer) terminalWidth() int {
	if f, ok := pr.w().(*os.File); ok {
		if width, _, err := term.GetSize(int(f.Fd())); err == nil && width > 0 {
			return width
		}
	}
	return fallbackWidth
}

func (pr *Printer) Print(data json.RawMessage, breadcrumbs []Breadcrumb) {
	if pr.Toon {
		pr.printToon(data, nil)
		return
	}

	if pr.Quiet {
		_, _ = fmt.Fprintln(pr.w(), string(data))
		return
	}

	if pr.UseJSON() {
		pr.printEnvelope(data, breadcrumbs, nil)
		return
	}

	pr.printPretty(data)
	pr.printBreadcrumbs(breadcrumbs)
}

func (pr *Printer) PrintWithPagination(data, pagination json.RawMessage, breadcrumbs []Breadcrumb) {
	if pr.Toon {
		pr.printToon(data, pagination)
		return
	}

	if pr.Quiet {
		_, _ = fmt.Fprintln(pr.w(), string(data))
		return
	}

	if pr.UseJSON() {
		pr.printEnvelope(data, breadcrumbs, pagination)
		return
	}

	pr.printPretty(data)
	pr.printPaginationSummary(pagination)
	pr.printBreadcrumbs(breadcrumbs)
}

func (pr *Printer) PrintTable(data json.RawMessage, headers []string, rows [][]interface{}, breadcrumbs []Breadcrumb) {
	if pr.Toon {
		pr.printToon(data, nil)
		return
	}

	if pr.Quiet {
		_, _ = fmt.Fprintln(pr.w(), string(data))
		return
	}

	if pr.UseJSON() {
		pr.printEnvelope(data, breadcrumbs, nil)
		return
	}

	pr.renderGrid(headers, buildGrid(rows), 0)
	pr.printBreadcrumbs(breadcrumbs)
}

func (pr *Printer) PrintMessage(msg string) {
	if pr.Quiet {
		_, _ = fmt.Fprintln(pr.w(), "success")
		return
	}

	if pr.Toon {
		_, _ = fmt.Fprintln(pr.w(), msg)
		return
	}

	if pr.UseJSON() {
		data, _ := json.Marshal(map[string]string{"message": msg})
		_, _ = fmt.Fprintln(pr.w(), string(data))
		return
	}

	_, _ = fmt.Fprintln(pr.w(), msg)
}

func (pr *Printer) PrintQuiet(data json.RawMessage, breadcrumbs []Breadcrumb) {
	if pr.Quiet {
		if id := extractIdentifier(data); id != "" {
			_, _ = fmt.Fprintln(pr.w(), id)
			return
		}
	}

	pr.Print(data, breadcrumbs)
}

func extractIdentifier(data json.RawMessage) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return ""
	}

	if len(raw) == 1 {
		for _, v := range raw {
			var inner map[string]json.RawMessage
			if json.Unmarshal(v, &inner) == nil {
				raw = inner
			}
		}
	}

	for _, key := range []string{"sid", "id", "name", "email"} {
		if v, ok := raw[key]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				return s
			}
		}
	}

	return ""
}

func (pr *Printer) printEnvelope(data json.RawMessage, breadcrumbs []Breadcrumb, pagination json.RawMessage) {
	envelope := Envelope{Data: data, Breadcrumbs: breadcrumbs, Pagination: pagination}
	out, _ := json.MarshalIndent(envelope, "", "  ")
	_, _ = fmt.Fprintln(pr.w(), string(out))
}

func (pr *Printer) printPretty(data json.RawMessage) {
	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err == nil {
		if len(rows) == 0 {
			_, _ = fmt.Fprintln(pr.w(), "No records found.")
			return
		}
		flattened, answers := flattenLabelValues(rows)
		if pr.Verbose || pr.tableUninformative(flattened, answers) {
			var raws []json.RawMessage
			if err := json.Unmarshal(data, &raws); err == nil && len(raws) == len(rows) {
				pr.printRecordBlocks(raws, rows)
				return
			}
		}
		pr.printTable(flattened, answers, 0)
		return
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		if inner, ok := singleNestedObject(data); ok {
			pr.printObject(inner, 1)
			return
		}
		pr.printObject(data, 1)
		return
	}

	_, _ = fmt.Fprintln(pr.w(), string(data))
}

func (pr *Printer) printToon(data, pagination json.RawMessage) {
	out, err := encodeToon(data, pagination)
	if err != nil {
		_, _ = fmt.Fprintln(pr.w(), string(data))
		return
	}
	_, _ = fmt.Fprintln(pr.w(), out)
}

func encodeToon(data, pagination json.RawMessage) (string, error) {
	var decodedData interface{}
	if err := json.Unmarshal(data, &decodedData); err != nil {
		return "", err
	}

	envelope := map[string]interface{}{"data": decodedData}

	if pagination != nil {
		var decodedPagination interface{}
		if err := json.Unmarshal(pagination, &decodedPagination); err != nil {
			return "", err
		}
		envelope["pagination"] = decodedPagination
	}

	return gotoon.Encode(envelope)
}

func (pr *Printer) printPaginationSummary(pagination json.RawMessage) {
	if pagination == nil {
		return
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(pagination, &parsed); err != nil {
		return
	}

	page := IntFrom(parsed, "current_page_number", "current_page", "page")
	total := IntFrom(parsed, "total_records", "total_count")
	totalPages := IntFrom(parsed, "total_pages")

	_, _ = fmt.Fprintf(pr.w(), "\nPage %d of %d (%d total records)\n", page, totalPages, total)
}

func IntFrom(parsed map[string]json.RawMessage, keys ...string) int {
	for _, key := range keys {
		raw, ok := parsed[key]
		if !ok {
			continue
		}
		var n float64
		if json.Unmarshal(raw, &n) == nil {
			return int(n)
		}
	}
	return 0
}

func (pr *Printer) printBreadcrumbs(breadcrumbs []Breadcrumb) {
	if len(breadcrumbs) == 0 {
		return
	}

	_, _ = fmt.Fprintln(pr.w())
	for _, b := range breadcrumbs {
		_, _ = fmt.Fprintf(pr.w(), "  %s: %s\n", b.Label, b.Command)
	}
}
