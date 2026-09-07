package client

import (
	"encoding/json"
	"net/url"
	"strconv"
)

type Pagination struct {
	TotalRecords      int `json:"total_records"`
	TotalPages        int `json:"total_pages"`
	CurrentPageNumber int `json:"current_page_number"`
	PageSize          int `json:"page_size"`
}

// UnmarshalJSON accepts both spellings of the current-page key. NeetoDeploy
// returns "current_page" where every other product returns
// "current_page_number".
func (p *Pagination) UnmarshalJSON(data []byte) error {
	type alias Pagination
	var raw struct {
		alias
		CurrentPage int `json:"current_page"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = Pagination(raw.alias)
	if p.CurrentPageNumber == 0 {
		p.CurrentPageNumber = raw.CurrentPage
	}
	return nil
}

// AddPaginationParams sends both spellings of the page key. Product-native
// endpoints read "page"; commons-backed ones read "page_number". Sending only
// one makes --page silently return the first page on half the endpoints.
func AddPaginationParams(params url.Values, page, pageSize int) {
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
		params.Set("page_number", strconv.Itoa(page))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.Itoa(pageSize))
	}
}
