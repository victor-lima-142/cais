package cais

import (
	"net/http"
	"strings"
)

// RouteGroup composes routes and middleware under an optional prefix.
type RouteGroup interface {
	Route
	Prefix(string) RouteGroup
	Name(string) RouteGroup
	AddMiddleware(...Middleware) RouteGroup
	Append(Route, ...Middleware) RouteGroup
	Get(string, Handle, ...Middleware) RouteGroup
	Post(string, Handle, ...Middleware) RouteGroup
	Put(string, Handle, ...Middleware) RouteGroup
	Delete(string, Handle, ...Middleware) RouteGroup
	Patch(string, Handle, ...Middleware) RouteGroup
}

type middlewareRoute struct {
	route       Route
	middlewares []Middleware
}

func (r *middlewareRoute) resolve(request *http.Request) (http.Handler, bool) {
	h, ok := r.route.resolve(request)
	if !ok {
		return nil, false
	}
	return applyChain(h, r.middlewares...), true
}

type group struct {
	prefix      string
	name        string
	routes      []Route
	middlewares []Middleware
}

func Group(configure func(RouteGroup)) RouteGroup {
	g := &group{}
	if configure != nil {
		configure(g)
	}
	return g
}

func (g *group) Prefix(prefix string) RouteGroup {
	g.prefix = strings.TrimRight(prefix, "/")
	return g
}

func (g *group) Name(name string) RouteGroup {
	g.name = name
	return g
}

func (g *group) AddMiddleware(middlewares ...Middleware) RouteGroup {
	g.middlewares = append(g.middlewares, middlewares...)
	return g
}

func (g *group) Append(child Route, middlewares ...Middleware) RouteGroup {
	if len(middlewares) > 0 {
		child = &middlewareRoute{route: child, middlewares: middlewares}
	}
	g.routes = append(g.routes, child)
	return g
}

func (g *group) Get(path string, h Handle, mws ...Middleware) RouteGroup {
	return g.Append(Get(path, h, mws...))
}

func (g *group) Post(path string, h Handle, mws ...Middleware) RouteGroup {
	return g.Append(Post(path, h, mws...))
}

func (g *group) Put(path string, h Handle, mws ...Middleware) RouteGroup {
	return g.Append(Put(path, h, mws...))
}

func (g *group) Delete(path string, h Handle, mws ...Middleware) RouteGroup {
	return g.Append(Delete(path, h, mws...))
}

func (g *group) Patch(path string, h Handle, mws ...Middleware) RouteGroup {
	return g.Append(Patch(path, h, mws...))
}

func (g *group) resolve(request *http.Request) (http.Handler, bool) {
	if !strings.HasPrefix(request.URL.Path, g.prefix) {
		return nil, false
	}
	path := strings.TrimPrefix(request.URL.Path, g.prefix)
	if path == "" {
		path = "/"
	} else if !strings.HasPrefix(path, "/") {
		return nil, false
	}
	childRequest := request.Clone(request.Context())
	childRequest.URL.Path = path
	for _, child := range g.routes {
		if h, ok := child.resolve(childRequest); ok {
			return applyChain(h, g.middlewares...), true
		}
	}
	return nil, false
}
