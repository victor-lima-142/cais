package cais

import "net/http"

type Route interface {
	resolve(*http.Request) (http.Handler, bool)
}

type route struct {
	method      string
	path        string
	handler     http.Handler
	middlewares []Middleware
}

func (r *route) resolve(request *http.Request) (http.Handler, bool) {
	if request.Method != r.method || request.URL.Path != r.path {
		return nil, false
	}
	return applyChain(r.handler, r.middlewares...), true
}

func newRoute(method, path string, h Handle, mws ...Middleware) Route {
	return &route{method: method, path: path, handler: adaptHandle(h), middlewares: mws}
}

func Get(path string, h Handle, mws ...Middleware) Route {
	return newRoute(http.MethodGet, path, h, mws...)
}

func Post(path string, h Handle, mws ...Middleware) Route {
	return newRoute(http.MethodPost, path, h, mws...)
}

func Put(path string, h Handle, mws ...Middleware) Route {
	return newRoute(http.MethodPut, path, h, mws...)
}

func Delete(path string, h Handle, mws ...Middleware) Route {
	return newRoute(http.MethodDelete, path, h, mws...)
}

func Patch(path string, h Handle, mws ...Middleware) Route {
	return newRoute(http.MethodPatch, path, h, mws...)
}
