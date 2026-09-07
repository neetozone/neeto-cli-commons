package output

import (
	"strings"
	"testing"
)

func TestFormatHeader(t *testing.T) {
	cases := map[string]string{
		"id":            "ID",
		"first_name":    "FIRST NAME",
		"trace_urls":    "TRACE URLS",
		"organization":  "ORGANIZATION",
		"total_records": "TOTAL RECORDS",
	}
	for in, want := range cases {
		if got := FormatHeader(in); got != want {
			t.Errorf("FormatHeader(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatValue(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want string
	}{
		{"nil", nil, "-"},
		{"true", true, "Yes"},
		{"false", false, "No"},
		{"whole float", float64(30), "30"},
		{"fractional float", 125.05, "125.05"},
		{"string", "Weekly sync", "Weekly sync"},
		{"scalar array", []interface{}{"u1", "u2", "u3"}, "u1, u2, u3"},
		{"empty array", []interface{}{}, "-"},
		{"mixed array", []interface{}{"a", true, 1.5}, "a, Yes, 1.50"},
		{"boolean array", []interface{}{true, false}, "Yes, No"},
		{"float array", []interface{}{1.0, 2.5}, "1, 2.50"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatValue(tc.in); got != tc.want {
				t.Errorf("formatValue(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsDisplayable(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want bool
	}{
		{"string", "x", true},
		{"nil", nil, true},
		{"scalar slice", []interface{}{"a", "b"}, true},
		{"empty slice", []interface{}{}, true},
		{"object slice", []interface{}{map[string]interface{}{"a": 1}}, false},
		{"object", map[string]interface{}{"a": 1}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isDisplayable(tc.in); got != tc.want {
				t.Errorf("isDisplayable(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsEmptyContainer(t *testing.T) {
	if !isEmptyContainer([]interface{}{}) || !isEmptyContainer(map[string]interface{}{}) {
		t.Error("empty slice and empty map must count as empty containers")
	}
	if isEmptyContainer([]interface{}{1}) || isEmptyContainer("") {
		t.Error("non-empty container and scalar must not count as empty containers")
	}
}

func TestFormatArray(t *testing.T) {
	long := make([]interface{}, 30)
	for i := range long {
		long[i] = "abcdefgh"
	}

	cases := []struct {
		name string
		in   []interface{}
		want string
	}{
		{"empty", []interface{}{}, "-"},
		{"scalars", []interface{}{"smoke", "fast", "ci"}, "smoke, fast, ci"},
		{"too long to inline", long, "(30 items)"},
		{"objects", []interface{}{map[string]interface{}{"a": float64(1)}}, "(1 items)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatArray(tc.in); got != tc.want {
				t.Errorf("formatArray(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormatArray_KeepsURLsWhateverTheLength(t *testing.T) {
	arr := []interface{}{longURL, longURL}

	got := formatArray(arr)

	if !strings.Contains(got, longURL) {
		t.Errorf("formatArray = %q, want it to keep the full URL", got)
	}
}

func TestSummarizeObject(t *testing.T) {
	if got := summarizeObject(map[string]interface{}{}); got != "-" {
		t.Errorf("summarizeObject(empty) = %q, want %q", got, "-")
	}
	if got := summarizeObject(map[string]interface{}{"a": "b"}); got != `{"a":"b"}` {
		t.Errorf("summarizeObject(small) = %q, want compact JSON", got)
	}

	big := map[string]interface{}{}
	for _, k := range []string{"aaaaaaaaaa", "bbbbbbbbbb", "cccccccccc", "dddddddddd", "eeeeeeeeee", "ffffffffff"} {
		big[k] = "0123456789"
	}
	if got := summarizeObject(big); got != "(6 fields)" {
		t.Errorf("summarizeObject(big) = %q, want %q", got, "(6 fields)")
	}
}

func TestSummarizeObject_KeepsURLsWhateverTheLength(t *testing.T) {
	obj := map[string]interface{}{"url": longURL, "padding": strings.Repeat("x", 120)}

	if got := summarizeObject(obj); !strings.Contains(got, longURL) {
		t.Errorf("summarizeObject = %q, want it to keep the URL", got)
	}
}

func TestDescribeValue(t *testing.T) {
	if got := describeValue([]interface{}{1, 2}); got != "2 items" {
		t.Errorf("describeValue(slice) = %q, want %q", got, "2 items")
	}
	if got := describeValue(map[string]interface{}{"a": 1}); got != "1 fields" {
		t.Errorf("describeValue(map) = %q, want %q", got, "1 fields")
	}
	if got := describeValue("x"); got != "..." {
		t.Errorf("describeValue(scalar) = %q, want %q", got, "...")
	}
}

func TestPreviewScalar_TruncatesAndFlattens(t *testing.T) {
	got := previewScalar("line one\nline two " + strings.Repeat("y", 200))

	if displayWidth(got) != maxPreviewLen {
		t.Errorf("previewScalar width = %d, want %d", displayWidth(got), maxPreviewLen)
	}
	if strings.Contains(got, "\n") {
		t.Errorf("previewScalar = %q, want newlines flattened", got)
	}
	if !strings.HasSuffix(got, ellipsis) {
		t.Errorf("previewScalar = %q, want it to end with %q", got, ellipsis)
	}
}

func TestContainsURL_FindsNestedLinks(t *testing.T) {
	nested := map[string]interface{}{
		"attachments": []interface{}{map[string]interface{}{"url": "https://example.com/a"}},
	}
	if !containsURL(nested) {
		t.Error("containsURL should find a URL nested inside an array of objects")
	}
	if containsURL(map[string]interface{}{"name": "no link"}) {
		t.Error("containsURL matched a payload with no link")
	}
}

func TestPadRight(t *testing.T) {
	if got := padRight("ab", 5); got != "ab   " {
		t.Errorf("padRight = %q, want %q", got, "ab   ")
	}
	if got := padRight("abcdef", 3); got != "abcdef" {
		t.Errorf("padRight must never shorten a value, got %q", got)
	}
}
