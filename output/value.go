package output

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxTableColumns = 7
	ellipsis        = "..."
	minContentWidth = 10
	minColWidth     = minContentWidth + len(ellipsis)
	colPadding      = 3
	maxRenderDepth  = 3
	maxPreviewLen   = 100
	inlineJSONLimit = 80
	fallbackWidth   = 100
)

func FormatHeader(field string) string {
	return strings.ToUpper(strings.ReplaceAll(field, "_", " "))
}

func isScalar(v interface{}) bool {
	switch v.(type) {
	case nil, string, float64, bool, json.Number:
		return true
	}
	return false
}

func isDisplayable(v interface{}) bool {
	if isScalar(v) {
		return true
	}
	arr, ok := v.([]interface{})
	if !ok {
		return false
	}
	for _, item := range arr {
		if !isScalar(item) {
			return false
		}
	}
	return true
}

func isEmptyContainer(v interface{}) bool {
	switch val := v.(type) {
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	}
	return false
}

func allObjects(arr []interface{}) bool {
	for _, item := range arr {
		if _, ok := item.(map[string]interface{}); !ok {
			return false
		}
	}
	return true
}

func formatValue(v interface{}) string {
	if v == nil {
		return "-"
	}
	switch val := v.(type) {
	case bool:
		if val {
			return "Yes"
		}
		return "No"
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case string:
		return val
	case []interface{}:
		if len(val) == 0 {
			return "-"
		}
		if !isDisplayable(val) {
			return "(" + describeValue(val) + ")"
		}
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = formatValue(item)
		}
		return strings.Join(parts, ", ")
	case map[string]interface{}:
		return "(" + describeValue(val) + ")"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func previewScalar(v interface{}) string {
	return truncate(strings.TrimSpace(strings.ReplaceAll(formatValue(v), "\n", " ")), maxPreviewLen)
}

func inlineValue(v interface{}) string {
	switch val := v.(type) {
	case map[string]interface{}:
		return summarizeObject(val)
	case []interface{}:
		return formatArray(val)
	}
	return previewScalar(v)
}

func summarizeObject(obj map[string]interface{}) string {
	if len(obj) == 0 {
		return "-"
	}
	compact, err := json.Marshal(obj)
	if err == nil && (len(compact) <= inlineJSONLimit || containsURL(obj)) {
		return string(compact)
	}
	return "(" + describeValue(obj) + ")"
}

func formatArray(arr []interface{}) string {
	if len(arr) == 0 {
		return "-"
	}

	if isDisplayable(arr) {
		joined := formatValue(arr)
		if displayWidth(joined) <= maxPreviewLen || containsURL(arr) {
			return joined
		}
	}

	if containsURL(arr) {
		if compact, err := json.Marshal(arr); err == nil {
			return string(compact)
		}
	}

	return "(" + describeValue(arr) + ")"
}

func describeValue(v interface{}) string {
	switch val := v.(type) {
	case []interface{}:
		return fmt.Sprintf("%d items", len(val))
	case map[string]interface{}:
		return fmt.Sprintf("%d fields", len(val))
	default:
		return "..."
	}
}

func isURL(s string) bool {
	return hasScheme(s, "http://") || hasScheme(s, "https://")
}

func hasScheme(s, scheme string) bool {
	return len(s) >= len(scheme) && strings.EqualFold(s[:len(scheme)], scheme)
}

func containsURL(v interface{}) bool {
	switch val := v.(type) {
	case string:
		return isURL(val)
	case []interface{}:
		for _, item := range val {
			if containsURL(item) {
				return true
			}
		}
	case map[string]interface{}:
		for _, item := range val {
			if containsURL(item) {
				return true
			}
		}
	}
	return false
}

func displayWidth(s string) int {
	return utf8.RuneCountInString(s)
}

func padRight(s string, width int) string {
	if gap := width - displayWidth(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}

func truncate(s string, maxLen int) string {
	if maxLen < 0 {
		maxLen = 0
	}
	if displayWidth(s) <= maxLen || isURL(s) {
		return s
	}
	runes := []rune(s)
	if maxLen <= len(ellipsis) {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-len(ellipsis)]) + ellipsis
}

func indentPrefix(indent int) string {
	return strings.Repeat("  ", indent)
}
