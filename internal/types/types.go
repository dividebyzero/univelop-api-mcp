package types

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Response wraps an HTTP response from the Univelop API.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// JSON returns the body as json.RawMessage for further processing.
func (r *Response) JSON() json.RawMessage {
	return json.RawMessage(r.Body)
}

// APIError represents a structured error from the Univelop API.
type APIError struct {
	StatusCode int
	Method     string
	URL        string
	Msg        string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s %s: API returned %d — %s", e.Method, e.URL, e.StatusCode, e.Msg)
}

// ParseErrorBody attempts to extract a human-readable message from an API error response.
func ParseErrorBody(body []byte, statusCode int) string {
	var raw struct {
		Err     string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &raw) == nil {
		if raw.Message != "" {
			return raw.Message
		}
		if raw.Err != "" {
			return raw.Err
		}
	}
	return fmt.Sprintf("HTTP %d", statusCode)
}