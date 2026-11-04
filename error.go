package cais

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// Error contains the fields of the default JSON error response.
type Error struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string     { return e.Message }
func (e Error) StatusCode() int   { return e.Status }
func (e Error) ErrorCode() string { return e.Code }

// HTTPError lets application errors supply their status and code.
type HTTPError interface {
	error
	StatusCode() int
	ErrorCode() string
}

// ErrorFormatter may extend or replace the default JSON error object.
type ErrorFormatter func(*http.Request, Error) any

type errorFormatterKey struct{}

func withErrorFormatter(request *http.Request, formatter ErrorFormatter) *http.Request {
	if formatter == nil {
		return request
	}
	return request.WithContext(context.WithValue(request.Context(), errorFormatterKey{}, formatter))
}

func renderError(w http.ResponseWriter, request *http.Request, err error) {
	base := Error{Status: http.StatusInternalServerError, Code: "internal_error", Message: "internal server error"}
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		base = Error{Status: httpErr.StatusCode(), Code: httpErr.ErrorCode(), Message: httpErr.Error()}
	}
	if base.Status < 400 || base.Status > 599 {
		base.Status = http.StatusInternalServerError
	}
	if base.Code == "" {
		base.Code = "internal_error"
	}
	if base.Message == "" {
		base.Message = http.StatusText(base.Status)
	}

	value := any(struct {
		Error Error `json:"error"`
	}{Error: base})
	if formatter, ok := request.Context().Value(errorFormatterKey{}).(ErrorFormatter); ok {
		value = formatter(request, base)
	}
	data, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		data, _ = json.Marshal(struct {
			Error Error `json:"error"`
		}{Error: base})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(base.Status)
	_, _ = w.Write(append(data, '\n'))
}
