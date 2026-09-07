package client

import (
	"net/url"
	"testing"
)

func TestAddPaginationParams_Both(t *testing.T) {
	params := url.Values{}
	AddPaginationParams(params, 3, 25)

	if got := params.Get("page"); got != "3" {
		t.Errorf("page = %q, want 3", got)
	}
	if got := params.Get("page_number"); got != "3" {
		t.Errorf("page_number = %q, want 3", got)
	}
	if got := params.Get("page_size"); got != "25" {
		t.Errorf("page_size = %q, want 25", got)
	}
}

func TestAddPaginationParams_PageOnly(t *testing.T) {
	params := url.Values{}
	AddPaginationParams(params, 2, 0)

	if got := params.Get("page"); got != "2" {
		t.Errorf("page = %q, want 2", got)
	}
	if got := params.Get("page_number"); got != "2" {
		t.Errorf("page_number = %q, want 2", got)
	}
	if got := params.Get("page_size"); got != "" {
		t.Errorf("page_size = %q, want empty", got)
	}
}

func TestAddPaginationParams_PageSizeOnly(t *testing.T) {
	params := url.Values{}
	AddPaginationParams(params, 0, 50)

	if got := params.Get("page"); got != "" {
		t.Errorf("page = %q, want empty", got)
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
