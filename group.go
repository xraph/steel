package forge_router

import (
	"net/http"
)

// RouteGroup Standard router types (keeping existing functionality)
type RouteGroup struct {
	router     *FastRouter
	prefix     string
	middleware []MiddlewareFunc
}

// Use RouteGroup implementation
func (g *RouteGroup) Use(middleware ...MiddlewareFunc) {
	g.middleware = append(g.middleware, middleware...)
}

// Group creates a new route group with the current group's prefix and middleware
func (g *RouteGroup) Group() Router {
	return &RouteGroup{
		router:     g.router,
		prefix:     g.prefix,
		middleware: append([]MiddlewareFunc{}, g.middleware...),
	}
}

// GroupFunc creates a route group and calls the provided function with it (renamed from Group)
func (g *RouteGroup) GroupFunc(fn func(r Router)) Router {
	subgroup := &RouteGroup{
		router:     g.router,
		prefix:     g.prefix,
		middleware: append([]MiddlewareFunc{}, g.middleware...),
	}
	fn(subgroup)
	return subgroup
}

func (g *RouteGroup) Route(pattern string, fn func(r Router)) Router {
	subgroup := &RouteGroup{
		router:     g.router,
		prefix:     g.prefix + pattern,
		middleware: append([]MiddlewareFunc{}, g.middleware...),
	}
	fn(subgroup)
	return subgroup
}

func (g *RouteGroup) Mount(pattern string, handler http.Handler) {
	g.router.Mount(g.prefix+pattern, handler)
}

func (g *RouteGroup) GET(pattern string, handler HandlerFunc) {
	g.Handle("GET", pattern, handler)
}

func (g *RouteGroup) POST(pattern string, handler HandlerFunc) {
	g.Handle("POST", pattern, handler)
}

func (g *RouteGroup) PUT(pattern string, handler HandlerFunc) {
	g.Handle("PUT", pattern, handler)
}

func (g *RouteGroup) DELETE(pattern string, handler HandlerFunc) {
	g.Handle("DELETE", pattern, handler)
}

func (g *RouteGroup) PATCH(pattern string, handler HandlerFunc) {
	g.Handle("PATCH", pattern, handler)
}

func (g *RouteGroup) HEAD(pattern string, handler HandlerFunc) {
	g.Handle("HEAD", pattern, handler)
}

func (g *RouteGroup) OPTIONS(pattern string, handler HandlerFunc) {
	g.Handle("OPTIONS", pattern, handler)
}

func (g *RouteGroup) Handle(method, pattern string, handler HandlerFunc) {
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	})

	for i := len(g.middleware) - 1; i >= 0; i-- {
		h = g.middleware[i](h)
	}

	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}

	g.router.addRoute(method, g.prefix+pattern, wrappedHandler)
}

func (g *RouteGroup) HandleFunc(method, pattern string, handler http.HandlerFunc) {
	g.Handle(method, pattern, func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	})
}

func (g *RouteGroup) OpinionatedGET(pattern string, handler interface{}, opts ...HandlerOption) {
	g.router.registerOpinionatedHandler("GET", g.prefix+pattern, handler, opts...)
}

func (g *RouteGroup) OpinionatedPOST(pattern string, handler interface{}, opts ...HandlerOption) {
	g.router.registerOpinionatedHandler("POST", g.prefix+pattern, handler, opts...)
}

func (g *RouteGroup) OpinionatedPUT(pattern string, handler interface{}, opts ...HandlerOption) {
	g.router.registerOpinionatedHandler("PUT", g.prefix+pattern, handler, opts...)
}

func (g *RouteGroup) OpinionatedDELETE(pattern string, handler interface{}, opts ...HandlerOption) {
	g.router.registerOpinionatedHandler("DELETE", g.prefix+pattern, handler, opts...)
}

func (g *RouteGroup) OpinionatedPATCH(pattern string, handler interface{}, opts ...HandlerOption) {
	g.router.registerOpinionatedHandler("PATCH", g.prefix+pattern, handler, opts...)
}

func (g *RouteGroup) WebSocket(pattern string, handler interface{}, opts ...AsyncHandlerOption) {
	g.router.WebSocket(g.prefix+pattern, handler, opts...)
}

func (g *RouteGroup) SSE(pattern string, handler interface{}, opts ...AsyncHandlerOption) {
	g.router.SSE(g.prefix+pattern, handler, opts...)
}
