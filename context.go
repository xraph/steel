package forgerouter

import (
	"context"
	"encoding/json"
	"net/http"
	"path"
	"strings"
)

// FastContext provides a rich context for opinionated handlers
type FastContext struct {
	Request  *http.Request
	Response http.ResponseWriter
	router   *FastRouter
	params   *Params
}

type contextKey struct{}

var paramsKey = contextKey{}

func ParamsFromContext(ctx context.Context) *Params {
	if params, ok := ctx.Value(paramsKey).(*Params); ok {
		return params
	}
	return &Params{}
}

func URLParam(r *http.Request, key string) string {
	params := ParamsFromContext(r.Context())
	return params.Get(key)
}

// Param FastContext methods
func (c *FastContext) Param(key string) string {
	return c.params.Get(key)
}

func (c *FastContext) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

func (c *FastContext) Header(key string) string {
	return c.Request.Header.Get(key)
}

func (c *FastContext) JSON(status int, data interface{}) error {
	c.Response.Header().Set("Content-Type", "application/json")
	c.Response.WriteHeader(status)
	return json.NewEncoder(c.Response).Encode(data)
}

func (c *FastContext) Status(status int) *FastContext {
	c.Response.WriteHeader(status)
	return c
}

func (c *FastContext) BindJSON(v interface{}) error {
	return json.NewDecoder(c.Request.Body).Decode(v)
}

// Remove method to Params for backtracking
func (p *Params) Remove(key string) {
	for i, k := range p.keys {
		if k == key {
			// Remove by replacing with last element and truncating
			lastIdx := len(p.keys) - 1
			p.keys[i] = p.keys[lastIdx]
			p.values[i] = p.values[lastIdx]
			p.keys = p.keys[:lastIdx]
			p.values = p.values[:lastIdx]
			return
		}
	}
}

// Path cleaning utility
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}

// Convert OpenAPI-style paths {id} to our parameter format :id
func convertOpenAPIPath(path string) string {
	// Replace {param} with :param
	for {
		start := strings.Index(path, "{")
		if start == -1 {
			break
		}
		end := strings.Index(path[start:], "}")
		if end == -1 {
			break
		}
		end += start

		paramName := path[start+1 : end]
		path = path[:start] + ":" + paramName + path[end+1:]
	}
	return path
}
