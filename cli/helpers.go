package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/neetozone/neeto-cli-commons/client"
	"github.com/neetozone/neeto-cli-commons/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	defaultMaxPageSize    = 100
	maxPageSizeAnnotation = "neeto_max_page_size"
)

func (a *App) Client(cmd *cobra.Command) (*client.Client, error) {
	subdomain := ""
	if a.Auth.RequiresSubdomain() {
		subdomain, _ = cmd.Flags().GetString("subdomain")
	}
	creds, err := a.Auth.SelectCredentials(subdomain)
	if err != nil {
		return nil, err
	}
	return client.New(a.Product, creds, a.Build.Version), nil
}

func (a *App) PrintList(data json.RawMessage, resourceKey string, breadcrumbs []output.Breadcrumb) {
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		a.Printer.Print(data, breadcrumbs)
		return
	}

	items, hasItems := parsed[resourceKey]
	pagination := parsed["pagination"]
	if pagination == nil {
		pagination = inlinePagination(parsed)
	}

	switch {
	case hasItems && pagination != nil:
		a.Printer.PrintWithPagination(items, pagination, breadcrumbs)
	case hasItems:
		a.Printer.Print(items, breadcrumbs)
	default:
		a.Printer.Print(data, breadcrumbs)
	}
}

func inlinePagination(parsed map[string]json.RawMessage) json.RawMessage {
	totalPages := output.IntFrom(parsed, "total_pages")
	if totalPages == 0 {
		return nil
	}

	block := client.Pagination{
		TotalRecords:      output.IntFrom(parsed, "total_records", "total_count"),
		TotalPages:        totalPages,
		CurrentPageNumber: output.IntFrom(parsed, "current_page_number", "current_page", "page"),
		PageSize:          output.IntFrom(parsed, "page_size", "per_page"),
	}

	out, err := json.Marshal(block)
	if err != nil {
		return nil
	}
	return out
}

func intFrom(parsed map[string]json.RawMessage, keys ...string) int {
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

func (a *App) PrintResource(data json.RawMessage, breadcrumbs []output.Breadcrumb) {
	a.Printer.Print(data, breadcrumbs)
}

func (a *App) PrintActionResult(data json.RawMessage, breadcrumbs []output.Breadcrumb) {
	a.Printer.PrintQuiet(data, breadcrumbs)
}

func (a *App) PrintMessage(msg string) { a.Printer.PrintMessage(msg) }

func (a *App) PaginationParams(cmd *cobra.Command) url.Values {
	params := url.Values{}
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	if max := maxPageSize(cmd); pageSize > max {
		pageSize = max
	}
	client.AddPaginationParams(params, page, pageSize)
	return params
}

func AddPaginationFlags(maxPageSize int, cmds ...*cobra.Command) {
	if maxPageSize <= 0 {
		maxPageSize = defaultMaxPageSize
	}
	for _, cmd := range cmds {
		cmd.Flags().Int("page", 0, "Page number")
		cmd.Flags().Int("page-size", 0, fmt.Sprintf("Items per page (max %d)", maxPageSize))
		if cmd.Annotations == nil {
			cmd.Annotations = map[string]string{}
		}
		cmd.Annotations[maxPageSizeAnnotation] = strconv.Itoa(maxPageSize)
	}
}

func maxPageSize(cmd *cobra.Command) int {
	if cmd == nil {
		return defaultMaxPageSize
	}
	n, err := strconv.Atoi(cmd.Annotations[maxPageSizeAnnotation])
	if err != nil || n <= 0 {
		return defaultMaxPageSize
	}
	return n
}

func MarkFlagsRequired(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		_ = cmd.MarkFlagRequired(name)
	}
}

func isRequired(flag *pflag.Flag) bool {
	if flag.Annotations == nil {
		return false
	}
	values, ok := flag.Annotations[cobra.BashCompOneRequiredFlag]
	return ok && len(values) > 0 && values[0] == "true"
}

func AllowJSONFileToSatisfyRequiredFlags(cmd *cobra.Command) {
	previous := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if previous != nil {
			if err := previous(cmd, args); err != nil {
				return err
			}
		}
		jsonFile, _ := cmd.Flags().GetString("json-file")
		if jsonFile == "" {
			return nil
		}
		fileData, err := ReadJSONFile(jsonFile)
		if err != nil {
			return err
		}
		nested := unwrapSingleObject(fileData)
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			key := strings.ReplaceAll(flag.Name, "-", "_")
			if jsonValueSatisfiesFlag(fileData[key]) || jsonValueSatisfiesFlag(nested[key]) {
				delete(flag.Annotations, cobra.BashCompOneRequiredFlag)
			}
		})
		return nil
	}
}

func unwrapSingleObject(data map[string]any) map[string]any {
	if len(data) != 1 {
		return nil
	}
	for _, v := range data {
		if inner, ok := v.(map[string]any); ok {
			return inner
		}
	}
	return nil
}

func jsonValueSatisfiesFlag(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	default:
		return true
	}
}

func ReadJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Could not read file %s: %w", path, err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("Invalid JSON in %s: %w", path, err)
	}
	return result, nil
}

func SplitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
