package cais

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestEngineAndGroups(t *testing.T) {
	var order []string
	middleware := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}

	engine := NewEngine()
	engine.Post("/direct", func(request Request, response Response) {
		order = append(order, "direct")
		body, err := request.Body()
		if err != nil {
			response.Error(err)
			return
		}
		response.Respond(body).Status(http.StatusCreated)
	}, middleware("route"))

	engine.Group(func(group RouteGroup) {
		child := Get("/hello", func(_ Request, response Response) {
			order = append(order, "hello")
			response.Respond("Hello World")
		})
		group.Append(child, middleware("append"))
	}).Prefix("/api").AddMiddleware(middleware("group"))

	for _, tc := range []struct {
		method string
		path   string
		input  string
		status int
		want   any
	}{
		{http.MethodPost, "/direct", `{"name":"book"}`, http.StatusCreated, map[string]any{"name": "book"}},
		{http.MethodGet, "/api/hello", "", http.StatusOK, "Hello World"},
	} {
		request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.input))
		response := httptest.NewRecorder()
		engine.(http.Handler).ServeHTTP(response, request)
		if response.Code != tc.status {
			t.Errorf("%s: status = %d", tc.path, response.Code)
		}
		var body any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Errorf("%s: invalid JSON: %v", tc.path, err)
		}
		if !reflect.DeepEqual(body, tc.want) {
			t.Errorf("%s: body = %v", tc.path, body)
		}
	}
	if want := []string{"route", "direct", "group", "append", "hello"}; !reflect.DeepEqual(order, want) {
		t.Errorf("middleware order = %v, want %v", order, want)
	}
}

func TestStandaloneGroupMustBeAttached(t *testing.T) {
	var order []string
	middleware := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	group := Group(func(group RouteGroup) {
		group.Get("/me", func(_ Request, response Response) {
			order = append(order, "handler")
			response.Respond("Is Me!")
		})
	}).Prefix("/auth").AddMiddleware(middleware("auth")).AddMiddleware(middleware("log"))
	engine := NewEngine().Append(group)
	response := httptest.NewRecorder()
	engine.(http.Handler).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/me", nil))
	if response.Code != http.StatusOK {
		t.Errorf("status = %d", response.Code)
	}
	if want := []string{"auth", "log", "handler"}; !reflect.DeepEqual(order, want) {
		t.Errorf("middleware order = %v, want %v", order, want)
	}
}

func TestRequestMetadata(t *testing.T) {
	type contextKey struct{}
	engine := NewEngine().Get("/metadata", func(req Request, response Response) {
		if req.Header().Get("X-Trace-ID") != "trace-1" || req.Authorization() != "Bearer secret" {
			t.Errorf("headers unavailable")
		}
		if req.Context().Value(contextKey{}) != "context-value" {
			t.Errorf("context unavailable")
		}
		cookies := req.Cookies()
		if len(cookies) != 1 || cookies[0].Value != "abc" {
			t.Errorf("cookies = %v", cookies)
		}
		response.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/metadata", nil)
	request.Header.Set("X-Trace-ID", "trace-1")
	request.Header.Set("Authorization", "Bearer secret")
	request.AddCookie(&http.Cookie{Name: "session", Value: "abc"})
	request = request.WithContext(context.WithValue(request.Context(), contextKey{}, "context-value"))
	response := httptest.NewRecorder()
	engine.(http.Handler).ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Errorf("status = %d", response.Code)
	}
}

func TestRequestAndResponseShareContext(t *testing.T) {
	type contextKey struct{}
	engine := NewEngine().FormatErrors(func(request *http.Request, base Error) any {
		return map[string]any{"error": base, "context": request.Context().Value(contextKey{})}
	})
	engine.Get("/context", func(req Request, response Response) {
		if req.Context().Value(contextKey{}) != "initial" || response.Context().Value(contextKey{}) != "initial" {
			t.Error("initial context was not shared")
		}
		req.WithContext(context.WithValue(req.Context(), contextKey{}, "request"))
		if response.Context().Value(contextKey{}) != "request" {
			t.Error("response did not receive request context")
		}
		response.WithContext(context.WithValue(response.Context(), contextKey{}, "response"))
		if req.Context().Value(contextKey{}) != "response" {
			t.Error("request did not receive response context")
		}
		response.Error(Error{Status: http.StatusBadRequest, Code: "test", Message: "test error"})
	})
	request := httptest.NewRequest(http.MethodGet, "/context", nil)
	request = request.WithContext(context.WithValue(request.Context(), contextKey{}, "initial"))
	response := httptest.NewRecorder()
	engine.(http.Handler).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["context"] != "response" {
		t.Errorf("context in error response = %v", body["context"])
	}
}

func TestRequestBodyCanDecodeIntoDestination(t *testing.T) {
	type item struct {
		Name string `json:"name"`
	}
	engine := NewEngine().Post("/items", func(request Request, response Response) {
		var body item
		decoded, err := request.Body(&body)
		if err != nil {
			response.Error(err)
			return
		}
		if body.Name == "book" && decoded != body {
			t.Errorf("decoded body = %v, destination = %+v", decoded, body)
		}
		response.Respond(body).Status(http.StatusCreated)
	})
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"name":"book"}`, http.StatusCreated},
		{`{"name":42}`, http.StatusBadRequest},
		{`{`, http.StatusBadRequest},
	} {
		response := httptest.NewRecorder()
		engine.(http.Handler).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(tc.body)))
		if response.Code != tc.status {
			t.Errorf("body %s: status = %d, want %d", tc.body, response.Code, tc.status)
		}
		if !json.Valid(response.Body.Bytes()) {
			t.Errorf("body %s: invalid JSON response %q", tc.body, response.Body.String())
		}
	}
}

func TestBindJSONReturnsDecodeError(t *testing.T) {
	type item struct {
		Name string `json:"name"`
	}
	engine := NewEngine().Post("/bind", func(request Request, response Response) {
		var body item
		if err := request.BindJSON(&body); err != nil {
			response.Error(err)
			return
		}
		response.Respond(body)
	})
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"name":"book"}`, http.StatusOK},
		{`{"name":42}`, http.StatusBadRequest},
		{"", http.StatusBadRequest},
	} {
		response := httptest.NewRecorder()
		engine.(http.Handler).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/bind", strings.NewReader(tc.body)))
		if response.Code != tc.status {
			t.Errorf("body %s: status = %d, want %d", tc.body, response.Code, tc.status)
		}
	}
}

func TestResponseJSON(t *testing.T) {
	type item struct {
		Name string `json:"name"`
	}
	engine := NewEngine()
	engine.Get("/map", func(_ Request, response Response) {
		response.JSON(http.StatusAccepted, map[string]string{"name": "book"})
	})
	engine.Get("/dto", func(_ Request, response Response) {
		response.JSON(http.StatusCreated, item{Name: "book"})
	})

	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/map", http.StatusAccepted},
		{"/dto", http.StatusCreated},
	} {
		response := httptest.NewRecorder()
		engine.(http.Handler).ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if response.Code != tc.status {
			t.Errorf("%s: status = %d, want %d", tc.path, response.Code, tc.status)
		}
		if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Errorf("%s: Content-Type = %q", tc.path, got)
		}
		var body item
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Name != "book" {
			t.Errorf("%s: body = %+v, error = %v", tc.path, body, err)
		}
	}
}
