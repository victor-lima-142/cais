package cais

import "net/http"

type Server interface {
	Get(string, Handle, ...Middleware) Server
	Post(string, Handle, ...Middleware) Server
	Put(string, Handle, ...Middleware) Server
	Delete(string, Handle, ...Middleware) Server
	Patch(string, Handle, ...Middleware) Server
	Group(func(RouteGroup)) RouteGroup
	Append(...Route) Server
	UseJSON() Server
	FormatErrors(ErrorFormatter) Server
	Run(string) error
}

type engine struct {
	root      *group
	useJSON   bool
	formatter ErrorFormatter
}

func NewEngine() Server {
	return &engine{root: &group{}}
}

func (e *engine) Get(path string, h Handle, mws ...Middleware) Server {
	e.root.Get(path, h, mws...)
	return e
}

func (e *engine) Post(path string, h Handle, mws ...Middleware) Server {
	e.root.Post(path, h, mws...)
	return e
}

func (e *engine) Put(path string, h Handle, mws ...Middleware) Server {
	e.root.Put(path, h, mws...)
	return e
}

func (e *engine) Delete(path string, h Handle, mws ...Middleware) Server {
	e.root.Delete(path, h, mws...)
	return e
}

func (e *engine) Patch(path string, h Handle, mws ...Middleware) Server {
	e.root.Patch(path, h, mws...)
	return e
}

func (e *engine) Group(configure func(RouteGroup)) RouteGroup {
	g := Group(configure)
	e.root.Append(g)
	return g
}

func (e *engine) Append(routes ...Route) Server {
	for _, route := range routes {
		e.root.Append(route)
	}
	return e
}

// UseJSON sets JSON as the default response content type.
func (e *engine) UseJSON() Server {
	e.useJSON = true
	return e
}

func (e *engine) FormatErrors(formatter ErrorFormatter) Server {
	e.formatter = formatter
	return e
}

func (e *engine) Run(addr string) error {
	return http.ListenAndServe(addr, e)
}

func (e *engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = withErrorFormatter(r, e.formatter)
	if e.useJSON {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	if h, ok := e.root.resolve(r); ok {
		h.ServeHTTP(w, r)
		return
	}
	renderError(w, r, Error{Status: http.StatusNotFound, Code: "not_found", Message: "not found"})
}
