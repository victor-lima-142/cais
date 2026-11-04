package cais

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

// Handle is the function accepted by every HTTP route.
type Handle func(Request, Response)

// Request exposes the decoded body and HTTP request metadata.
type Request struct {
	request *http.Request
	raw     []byte
	state   *responseState
}

// Body decodes the JSON request body. Without a destination it returns a value
// of type any; with a pointer it also fills and returns the pointed-to value.
func (r Request) Body(target ...any) (any, error) {
	if len(target) == 0 {
		if len(r.raw) == 0 {
			return nil, nil
		}
		var body any
		if err := json.Unmarshal(r.raw, &body); err != nil {
			return nil, invalidJSON(err)
		}
		return body, nil
	}
	if len(target) != 1 {
		return nil, fmt.Errorf("cais: Body accepts at most one destination")
	}
	value := reflect.ValueOf(target[0])
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		return nil, fmt.Errorf("cais: JSON destination must be a non-nil pointer")
	}
	if len(r.raw) == 0 {
		return nil, invalidJSON(io.EOF)
	}
	if err := json.Unmarshal(r.raw, target[0]); err != nil {
		return nil, invalidJSON(err)
	}
	return value.Elem().Interface(), nil
}

// BindJSON decodes the JSON request body into a non-nil pointer.
func (r Request) BindJSON(target any) error {
	_, err := r.Body(target)
	return err
}

func invalidJSON(err error) error {
	return fmt.Errorf("%w: %v", Error{Status: http.StatusBadRequest, Code: "invalid_json", Message: "invalid JSON body"}, err)
}

// Header returns a copy of the request headers.
func (r Request) Header() http.Header {
	if r.request == nil {
		return make(http.Header)
	}
	return r.request.Header.Clone()
}

func (r Request) Context() context.Context {
	if r.state != nil && r.state.ctx != nil {
		return r.state.ctx
	}
	if r.request == nil {
		return context.Background()
	}
	return r.request.Context()
}

// WithContext shares a new context with the response in this handler.
func (r Request) WithContext(ctx context.Context) Request {
	if ctx == nil {
		panic("cais: nil context")
	}
	if r.request != nil {
		r.request = r.request.WithContext(ctx)
	}
	if r.state == nil {
		r.state = &responseState{}
	}
	r.state.ctx = ctx
	return r
}

func (r Request) Authorization() string {
	if r.request == nil {
		return ""
	}
	return r.request.Header.Get("Authorization")
}

func (r Request) Cookies() []*http.Cookie {
	if r.request == nil {
		return nil
	}
	return r.request.Cookies()
}

func adaptHandle(handler Handle) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var raw []byte
		if r.Body != nil {
			var err error
			raw, err = io.ReadAll(r.Body)
			if err != nil {
				renderError(w, r, Error{Status: http.StatusBadRequest, Code: "invalid_json", Message: "invalid JSON body"})
				return
			}
		}
		response := Response{state: &responseState{ctx: r.Context()}}
		handler(Request{request: r, raw: raw, state: response.state}, response)
		response.write(w, r)
	}
}
