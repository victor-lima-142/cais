package cais

import (
	"context"
	"encoding/json"
	"net/http"
)

type responseState struct {
	status  int
	body    any
	hasBody bool
	err     error
	ctx     context.Context
}

// Response collects the result written by a route handler.
type Response struct {
	state *responseState
}

// Context returns the context shared with the request in this handler.
func (r Response) Context() context.Context {
	if r.state == nil || r.state.ctx == nil {
		return context.Background()
	}
	return r.state.ctx
}

// WithContext shares a new context with the request in this handler.
func (r Response) WithContext(ctx context.Context) Response {
	if ctx == nil {
		panic("cais: nil context")
	}
	if r.state == nil {
		r.state = &responseState{}
	}
	r.state.ctx = ctx
	return r
}

// Respond sets the JSON response body.
func (r Response) Respond(body any) Response {
	r.state.body = body
	r.state.hasBody = true
	r.state.err = nil
	return r
}

// Status sets the HTTP status for the response.
func (r Response) Status(status int) Response {
	r.state.status = status
	return r
}

// Error sends an error through the server's standard error formatter.
func (r Response) Error(err error) Response {
	r.state.err = err
	r.state.hasBody = false
	return r
}

func (r Response) write(w http.ResponseWriter, request *http.Request) {
	request = request.WithContext(r.Context())
	if r.state.err != nil {
		renderError(w, request, r.state.err)
		return
	}
	status := r.state.status
	if !r.state.hasBody {
		if status == 0 {
			status = http.StatusNoContent
		}
		w.WriteHeader(status)
		return
	}
	data, err := json.Marshal(r.state.body)
	if err != nil {
		renderError(w, request, err)
		return
	}
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status != http.StatusNoContent && status != http.StatusNotModified {
		_, _ = w.Write(append(data, '\n'))
	}
}
