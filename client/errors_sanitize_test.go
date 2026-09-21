package client

import (
	"strings"
	"testing"
)

// A server-supplied error message reaches the terminal through app.go rather
// than the printer, so it has to be sanitized where it is parsed.
func TestParseAPIErrorStripsTerminalEscapes(t *testing.T) {
	// \x1b[2K erases the line and \r returns the cursor, so unsanitized this
	// prints as "All checks passed" over the top of the real failure.
	body := []byte(`{"error":"Invalid name: \u001b[2K\rAll checks passed"}`)

	apiErr := parseAPIError(422, body, errorOptions{binaryName: "neetotest"})

	if strings.ContainsRune(apiErr.Message, 0x1b) {
		t.Errorf("Message still carries ESC: %q", apiErr.Message)
	}
	if strings.ContainsRune(apiErr.Message, '\r') {
		t.Errorf("Message still carries CR: %q", apiErr.Message)
	}
	if !strings.Contains(apiErr.Message, "Invalid name") {
		t.Errorf("sanitizer ate the real message: %q", apiErr.Message)
	}
	if got := apiErr.Error(); strings.ContainsRune(got, 0x1b) {
		t.Errorf("Error() still carries ESC: %q", got)
	}
}

func TestParseAPIErrorStripsEscapesFromDetails(t *testing.T) {
	body := []byte(`{"errors":["first \u001b[31mred\u001b[0m","second \u001b]52;c;cGF5bG9hZA==\u0007"],` +
		`"field_errors":[{"field":"name","message":"bad \u001b[2Kwiped"}]}`)

	apiErr := parseAPIError(422, body, errorOptions{binaryName: "neetotest"})

	if got := apiErr.Error(); strings.ContainsRune(got, 0x1b) {
		t.Errorf("Error() still carries ESC: %q", got)
	}
	for i, detail := range apiErr.Errors {
		if strings.ContainsRune(detail, 0x1b) {
			t.Errorf("Errors[%d] still carries ESC: %q", i, detail)
		}
	}
}

// A message that is nothing but escapes must fall back to the status text
// rather than print as an empty error.
func TestParseAPIErrorFallsBackWhenMessageSanitizesToEmpty(t *testing.T) {
	apiErr := parseAPIError(500, []byte(`{"error":"\u001b[2K\u001b[31m"}`), errorOptions{})

	if apiErr.Message != "Internal Server Error" {
		t.Errorf("Message = %q, want the status text", apiErr.Message)
	}
}
