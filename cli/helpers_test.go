package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/neetozone/neeto-cli-commons/config"
	"github.com/spf13/cobra"
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

func paginationCmd(maxPageSize int) *cobra.Command {
	cmd := &cobra.Command{Use: "list"}
	AddPaginationFlags(maxPageSize, cmd)
	return cmd
}

func TestPaginationParamsClampsToTheAdvertisedMax(t *testing.T) {
	a, _ := listPrinterApp(t)
	cmd := paginationCmd(50)
	if err := cmd.Flags().Set("page-size", "500"); err != nil {
		t.Fatal(err)
	}

	if got := a.PaginationParams(cmd).Get("page_size"); got != "50" {
		t.Errorf("page_size = %q, want 50; the flag help promises a maximum of 50", got)
	}
}

func TestPaginationParamsLeavesAValueUnderTheMaxAlone(t *testing.T) {
	a, _ := listPrinterApp(t)
	cmd := paginationCmd(50)
	if err := cmd.Flags().Set("page-size", "20"); err != nil {
		t.Fatal(err)
	}

	if got := a.PaginationParams(cmd).Get("page_size"); got != "20" {
		t.Errorf("page_size = %q, want 20", got)
	}
}

func TestPaginationParamsDefaultsTheMaxToAHundred(t *testing.T) {
	a, _ := listPrinterApp(t)
	cmd := paginationCmd(0)
	if err := cmd.Flags().Set("page-size", "250"); err != nil {
		t.Fatal(err)
	}

	if got := a.PaginationParams(cmd).Get("page_size"); got != "100" {
		t.Errorf("page_size = %q, want 100", got)
	}
}

func TestPrintListReadsNeetoInvoicePaginationKeys(t *testing.T) {
	a, buf := listPrinterApp(t)

	a.PrintList(json.RawMessage(`{"time_entries":[{"id":1}],"total_count":42,"total_pages":5,"page":2,"page_size":10}`), "time_entries", nil)

	out := buf.String()
	for _, want := range []string{`"current_page"`, `"total_count": 42`, `"total_pages": 5`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in: %s", want, out)
		}
	}
}

func TestPrintListIgnoresAPageKeyWithoutTotalPages(t *testing.T) {
	a, buf := listPrinterApp(t)

	a.PrintList(json.RawMessage(`{"time_entries":[{"id":1}],"page":2}`), "time_entries", nil)

	if strings.Contains(buf.String(), "pagination") {
		t.Errorf("a body with no total_pages must not grow a pagination block: %s", buf.String())
	}
}
