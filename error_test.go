package cais

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorsUseStandardRenderer(t *testing.T) {
	engine := NewEngine().Append(Post("/items", func(request Request, response Response) {
		response.Error(Error{Status: http.StatusUnprocessableEntity, Code: "invalid_item", Message: "item is invalid"})
	}))
	for _, tc := range []struct {
		path   string
		body   string
		status int
		code   string
	}{
		{"/missing", "", http.StatusNotFound, "not_found"},
		{"/items", `{`, http.StatusBadRequest, "invalid_json"},
		{"/items", `{}`, http.StatusUnprocessableEntity, "invalid_item"},
	} {
		response := httptest.NewRecorder()
		engine.(http.Handler).ServeHTTP(response, httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body)))
		if response.Code != tc.status {
			t.Errorf("%s: status = %d", tc.path, response.Code)
		}
		var body struct {
			Error Error `json:"error"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if body.Error.Code != tc.code {
			t.Errorf("%s: error = %+v", tc.path, body.Error)
		}
	}
}

func TestClientCanExtendErrorResponse(t *testing.T) {
	engine := NewEngine().FormatErrors(func(request *http.Request, base Error) any {
		return struct {
			Error     Error  `json:"error"`
			RequestID string `json:"request_id"`
		}{Error: base, RequestID: request.Header.Get("X-Request-ID")}
	}).Append(Get("/item", func(_ Request, response Response) {
		response.Error(Error{Status: http.StatusUnprocessableEntity, Code: "invalid_item", Message: "item is invalid"})
	}))
	request := httptest.NewRequest(http.MethodGet, "/item", nil)
	request.Header.Set("X-Request-ID", "request-1")
	response := httptest.NewRecorder()
	engine.(http.Handler).ServeHTTP(response, request)

	var body struct {
		Error     Error  `json:"error"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "invalid_item" || body.RequestID != "request-1" {
		t.Errorf("response = %+v", body)
	}
}
