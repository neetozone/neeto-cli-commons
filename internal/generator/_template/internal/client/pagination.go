package client

import (
	"fmt"
	"net/url"
)

type Pagination struct {
	TotalRecords      int `json:"total_records"`
	TotalPages        int `json:"total_pages"`
	CurrentPageNumber int `json:"current_page_number"`
	PageSize          int `json:"page_size"`
}

func AddPaginationParams(params url.Values, page, pageSize int) {
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}
	if pageSize > 0 {
		params.Set("page_size", fmt.Sprintf("%d", pageSize))
	}
}
