package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/neetozone/neeto-cli-commons/config"
)

func listPrinterApp(t *testing.T) (*App, *bytes.Buffer) {
	t.Helper()

	p := config.Product{PrettyName: "NeetoWidget", BinaryName: "neetowidget"}
	p.ApplyDerivations()

	a := New(p)
	buf := &bytes.Buffer{}
	a.Printer.Out = buf
	a.Printer.ForceJSON = true
	return a, buf
}

func TestPrintListRendersNestedPagination(t *testing.T) {
	a, buf := listPrinterApp(t)

	a.PrintList(json.RawMessage(`{"apps":[{"id":1}],"pagination":{"total_pages":3,"current_page":2}}`), "apps", nil)

	if !strings.Contains(buf.String(), `"total_pages": 3`) {
		t.Errorf("nested pagination was dropped: %s", buf.String())
	}
}

func TestPrintListRebuildsTopLevelPagination(t *testing.T) {
	a, buf := listPrinterApp(t)

	a.PrintList(json.RawMessage(`{"apps":[{"id":1}],"total_pages":3,"current_page":2,"total_count":25}`), "apps", nil)

	out := buf.String()
	for _, want := range []string{`"total_pages": 3`, `"current_page": 2`, `"total_count": 25`} {
		if !strings.Contains(out, want) {
			t.Errorf("top-level pagination not rebuilt, missing %s in: %s", want, out)
		}
	}
}

func TestPrintListWithoutPaginationStaysUnchanged(t *testing.T) {
	a, buf := listPrinterApp(t)

	a.PrintList(json.RawMessage(`{"apps":[{"id":1}],"current_page":2}`), "apps", nil)

	if strings.Contains(buf.String(), "pagination") {
		t.Errorf("a body with no total_pages must not grow a pagination block: %s", buf.String())
	}
}
