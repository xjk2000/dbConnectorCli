package protocol

import (
	"encoding/json"
	"io"
)

type Error struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Retryable  bool   `json:"retryable"`
	DriverCode string `json:"driverCode,omitempty"`
}

func NewError(code, message string, retryable bool) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Retryable: retryable,
	}
}

func Success(engine, profile, resultType string, elapsedMs int64, fields map[string]any) map[string]any {
	resp := map[string]any{
		"ok":        true,
		"engine":    engine,
		"profile":   profile,
		"type":      resultType,
		"elapsedMs": elapsedMs,
	}
	for key, value := range fields {
		resp[key] = value
	}
	return resp
}

func Failure(engine, profile string, err *Error, elapsedMs int64) map[string]any {
	return map[string]any{
		"ok":        false,
		"engine":    engine,
		"profile":   profile,
		"error":     err,
		"elapsedMs": elapsedMs,
	}
}

func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
