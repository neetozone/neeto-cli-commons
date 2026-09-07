package client

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestAddPaginationParams_SendsPageNumberOnly(t *testing.T) {
	params := url.Values{}
	AddPaginationParams(params, 3, 25)

	if got := params.Get("page_number"); got != "3" {
		t.Errorf("page_number = %q, want 3", got)
	}
	if got := params.Get("page_size"); got != "25" {
		t.Errorf("page_size = %q, want 25", got)
	}
	if _, ok := params["page"]; ok {
		t.Errorf("legacy 'page' param sent: %v", params)
	}
	if len(params) != 2 {
		t.Errorf("params = %v, want exactly page_number and page_size", params)
	}
}

func TestAddPaginationParams_PageOnly(t *testing.T) {
	params := url.Values{}
	AddPaginationParams(params, 2, 0)

	if got := params.Get("page_number"); got != "2" {
		t.Errorf("page_number = %q, want 2", got)
	}
	if got := params.Get("page_size"); got != "" {
		t.Errorf("page_size = %q, want empty", got)
	}
	if _, ok := params["page"]; ok {
		t.Errorf("legacy 'page' param sent: %v", params)
	}
}

func TestAddPaginationParams_PageSizeOnly(t *testing.T) {
	params := url.Values{}
	AddPaginationParams(params, 0, 50)

	if got := params.Get("page_number"); got != "" {
		t.Errorf("page_number = %q, want empty", got)
	}
	if got := params.Get("page_size"); got != "50" {
		t.Errorf("page_size = %q, want 50", got)
	}
}

func TestAddPaginationParams_ZeroValues(t *testing.T) {
	params := url.Values{}
	AddPaginationParams(params, 0, 0)

	if len(params) != 0 {
		t.Errorf("params = %v, want empty", params)
	}
}

func TestPagination_AcceptsBothCurrentPageKeys(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"current_page_number", `{"total_records":9,"total_pages":3,"current_page_number":2,"page_size":4}`},
		{"current_page", `{"total_records":9,"total_pages":3,"current_page":2,"page_size":4}`},
	}

	for _, tt := range tests {
		var p Pagination
		if err := json.Unmarshal([]byte(tt.body), &p); err != nil {
			t.Fatalf("%s: Unmarshal error = %v", tt.name, err)
		}
		if p.CurrentPageNumber != 2 {
			t.Errorf("%s: CurrentPageNumber = %d, want 2", tt.name, p.CurrentPageNumber)
		}
		if p.TotalRecords != 9 || p.TotalPages != 3 || p.PageSize != 4 {
			t.Errorf("%s: pagination = %+v", tt.name, p)
		}
	}
}

func TestPagination_PrefersCurrentPageNumber(t *testing.T) {
	var p Pagination
	body := `{"current_page_number":5,"current_page":2}`
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if p.CurrentPageNumber != 5 {
		t.Errorf("CurrentPageNumber = %d, want 5", p.CurrentPageNumber)
	}
}

func TestPagination_Absent(t *testing.T) {
	var p Pagination
	if err := json.Unmarshal([]byte(`{}`), &p); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if p.CurrentPageNumber != 0 {
		t.Errorf("CurrentPageNumber = %d, want 0", p.CurrentPageNumber)
	}
}
