package cais

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
)

// Handle is the function accepted by every HTTP route.
type Handle func(Request, Response)

// Request exposes the decoded body and HTTP request metadata.
type Request struct {
	request *http.Request
	body    any
	raw     []byte
	state   *responseState
}

// Body returns the decoded JSON value. With a pointer, it also decodes into
// that value and returns its typed contents. Decode failures become HTTP 400.
func (r Request) Body(target ...any) any {
	if len(target) == 0 {
		return r.body
	}
	if len(target) != 1 {
		r.rejectBody()
		return nil
	}
	value := reflect.ValueOf(target[0])
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		r.rejectBody()
		return nil
	}
	if len(r.raw) > 0 {
		if err := json.Unmarshal(r.raw, target[0]); err != nil {
			r.rejectBody()
			return nil
		}
	}
	return value.Elem().Interface()
}

func (r Request) rejectBody() {
	if r.state != nil {
		r.state.decodeErr = Error{Status: http.StatusBadRequest, Code: "invalid_json", Message: "invalid JSON body"}
	}
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
		var body any
		var raw []byte
		if r.Body != nil {
			var err error
			raw, err = io.ReadAll(r.Body)
			if err != nil {
				renderError(w, r, Error{Status: http.StatusBadRequest, Code: "invalid_json", Message: "invalid JSON body"})
				return
			}
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &body); err != nil {
					renderError(w, r, Error{Status: http.StatusBadRequest, Code: "invalid_json", Message: "invalid JSON body"})
					return
				}
			}
		}
		response := Response{state: &responseState{ctx: r.Context()}}
		handler(Request{request: r, body: body, raw: raw, state: response.state}, response)
		response.write(w, r)
	}
}
