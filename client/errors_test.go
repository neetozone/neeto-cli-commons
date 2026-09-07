package client

import (
	"strings"
	"testing"
)

func deskOptions() errorOptions {
	return errorOptions{binaryName: "neetodesk"}
}

func TestParseAPIError_ErrorField(t *testing.T) {
	err := parseAPIError(404, []byte(`{"error":"Item not found"}`), deskOptions())

	if err.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", err.StatusCode)
	}
	if err.Message != "Item not found" {
		t.Errorf("Message = %q, want %q", err.Message, "Item not found")
	}
}

func TestParseAPIError_NoticeField(t *testing.T) {
	err := parseAPIError(429, []byte(`{"notice":"Rate limit exceeded"}`), deskOptions())

	if err.Message != "Rate limit exceeded" {
		t.Errorf("Message = %q, want %q", err.Message, "Rate limit exceeded")
	}
}

func TestParseAPIError_ErrorsArray(t *testing.T) {
	err := parseAPIError(422, []byte(`{"errors":["Name is required","Slug is required"]}`), deskOptions())

	if err.Message != "Name is required" {
		t.Errorf("Message = %q, want %q", err.Message, "Name is required")
	}
	if len(err.Errors) != 1 || err.Errors[0] != "Slug is required" {
		t.Errorf("Errors = %v, want [\"Slug is required\"]", err.Errors)
	}
}

func TestParseAPIError_FieldErrors(t *testing.T) {
	body := []byte(`{"error":"Validation failed","field_errors":[{"field":"name","message":"is required"},{"message":"is invalid"}]}`)
	err := parseAPIError(422, body, deskOptions())

	if err.Message != "Validation failed" {
		t.Errorf("Message = %q, want %q", err.Message, "Validation failed")
	}
	if len(err.Errors) != 2 {
		t.Fatalf("Errors = %v, want 2 entries", err.Errors)
	}
	if err.Errors[0] != "name: is required" {
		t.Errorf("Errors[0] = %q, want %q", err.Errors[0], "name: is required")
	}
	if err.Errors[1] != "is invalid" {
		t.Errorf("Errors[1] = %q, want %q", err.Errors[1], "is invalid")
	}
}

func TestParseAPIError_NoFieldErrorsIsHarmless(t *testing.T) {
	err := parseAPIError(422, []byte(`{"errors":["Name is required"]}`), deskOptions())

	if len(err.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", err.Errors)
	}
}

func TestParseAPIError_InvalidJSON(t *testing.T) {
	err := parseAPIError(500, []byte(`not json`), deskOptions())

	if err.Message != "Internal Server Error" {
		t.Errorf("Message = %q, want %q", err.Message, "Internal Server Error")
	}
}

func TestParseAPIError_EmptyBody(t *testing.T) {
	err := parseAPIError(400, []byte(`{}`), deskOptions())

	if err.Message != "Bad Request" {
		t.Errorf("Message = %q, want %q", err.Message, "Bad Request")
	}
}

func TestParseAPIError_Suggestions(t *testing.T) {
	tests := []struct {
		code     int
		contains string
	}{
		{401, "neetodesk login"},
		{403, "permission"},
		{404, "not found"},
		{422, "neetodesk <command> --help"},
		{429, "Rate limited"},
	}

	for _, tt := range tests {
		err := parseAPIError(tt.code, []byte(`{}`), deskOptions())
		if !strings.Contains(strings.ToLower(err.Suggestion), strings.ToLower(tt.contains)) {
			t.Errorf("parseAPIError(%d).Suggestion = %q, want it to contain %q", tt.code, err.Suggestion, tt.contains)
		}
	}
}

func TestParseAPIError_SuggestionHookOverrides(t *testing.T) {
	opts := deskOptions()
	opts.suggestionFor = func(status int, message string) string {
		if status == 404 {
			return "No app with that slug."
		}
		return ""
	}

	if got := parseAPIError(404, []byte(`{"error":"App not found"}`), opts).Suggestion; got != "No app with that slug." {
		t.Errorf("Suggestion = %q, want the hook's text", got)
	}
	if got := parseAPIError(401, []byte(`{}`), opts).Suggestion; !strings.Contains(got, "neetodesk login") {
		t.Errorf("Suggestion = %q, want the built-in 401 text when the hook returns empty", got)
	}
}

func TestParseAPIError_NoSuggestion(t *testing.T) {
	err := parseAPIError(500, []byte(`{}`), deskOptions())
	if err.Suggestion != "" {
		t.Errorf("Suggestion = %q, want empty for 500", err.Suggestion)
	}
}

func TestAPIError_ErrorString(t *testing.T) {
	err := &APIError{
		StatusCode: 422,
		Message:    "Name is required",
		Errors:     []string{"Slug is required"},
		Suggestion: "Check required fields.",
	}

	got := err.Error()
	for _, want := range []string{"422", "Name is required", "Slug is required", "Check required fields."} {
		if !strings.Contains(got, want) {
			t.Errorf("Error() = %q, want it to contain %q", got, want)
		}
	}
}

func TestAPIError_ErrorString_NoExtras(t *testing.T) {
	err := &APIError{StatusCode: 500, Message: "Internal Server Error"}

	if got, want := err.Error(), "API error (500): Internal Server Error"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestStatusText(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{422, "Unprocessable Entity"},
		{429, "Too Many Requests"},
		{500, "Internal Server Error"},
		{503, "HTTP 503"},
	}

	for _, tt := range tests {
		if got := statusText(tt.code); got != tt.want {
			t.Errorf("statusText(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
