package client

import (
	"encoding/json"
	"fmt"

	"github.com/neetozone/neeto-cli-commons/output"
)

type APIError struct {
	StatusCode int
	Message    string
	Errors     []string
	Suggestion string
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
	for _, detail := range e.Errors {
		msg += "\n  - " + detail
	}
	if e.Suggestion != "" {
		msg += "\n\nSuggestion: " + e.Suggestion
	}
	return msg
}

type errorOptions struct {
	binaryName    string
	suggestionFor func(status int, message string) string
}

func parseAPIError(statusCode int, body []byte, opts errorOptions) *APIError {
	apiErr := &APIError{StatusCode: statusCode}

	var parsed struct {
		Error       string   `json:"error"`
		Errors      []string `json:"errors"`
		Notice      string   `json:"notice"`
		FieldErrors []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"field_errors"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		if parsed.Error != "" {
			apiErr.Message = parsed.Error
		} else if parsed.Notice != "" {
			apiErr.Message = parsed.Notice
		} else if len(parsed.Errors) > 0 {
			apiErr.Message = parsed.Errors[0]
			apiErr.Errors = parsed.Errors[1:]
		}
		for _, fieldErr := range parsed.FieldErrors {
			detail := fieldErr.Message
			if fieldErr.Field != "" {
				detail = fieldErr.Field + ": " + fieldErr.Message
			}
			apiErr.Errors = append(apiErr.Errors, detail)
		}
	}

	// The error path never reaches the printer, so sanitize at parse.
	apiErr.Message = output.SanitizeControlChars(apiErr.Message)
	// A detail that is nothing but escapes would print as a bare bullet, so
	// drop it rather than pad the list with blank lines.
	details := make([]string, 0, len(apiErr.Errors))
	for _, detail := range apiErr.Errors {
		if detail = output.SanitizeControlChars(detail); detail != "" {
			details = append(details, detail)
		}
	}
	apiErr.Errors = details

	if apiErr.Message == "" {
		apiErr.Message = statusText(statusCode)
	}

	if opts.suggestionFor != nil {
		apiErr.Suggestion = opts.suggestionFor(statusCode, apiErr.Message)
	}
	if apiErr.Suggestion == "" {
		apiErr.Suggestion = defaultSuggestion(statusCode, opts.binaryName)
	}

	return apiErr
}

func defaultSuggestion(statusCode int, binaryName string) string {
	switch statusCode {
	case 401:
		return fmt.Sprintf("Authentication session expired. Run '%s login' to re-authenticate.", binaryName)
	case 403:
		return "You do not have permission to perform this action."
	case 404:
		return "Resource not found. Check the ID and try again."
	case 422:
		return fmt.Sprintf("Check required fields with '%s <command> --help'.", binaryName)
	case 429:
		return "Rate limited. Wait and try again."
	}
	return ""
}

func statusText(code int) string {
	switch code {
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 422:
		return "Unprocessable Entity"
	case 429:
		return "Too Many Requests"
	case 500:
		return "Internal Server Error"
	default:
		return fmt.Sprintf("HTTP %d", code)
	}
}
