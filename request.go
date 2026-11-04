package cais

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// Handle is the function accepted by every HTTP route.
type Handle func(Request, Response)

// Request exposes the decoded body and HTTP request metadata.
type Request struct {
	request *http.Request
	body    any
	state   *responseState
}

func (r Request) Body() any { return r.body }

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
		if r.Body != nil {
			raw, err := io.ReadAll(r.Body)
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
		handler(Request{request: r, body: body, state: response.state}, response)
		response.write(w, r)
	}
}
