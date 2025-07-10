package steel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SteelRouter is a high-performance HTTP router with OpenAPI support
type SteelRouter struct {
	trees                 map[string]*node
	middleware            []MiddlewareFunc
	pool                  sync.Pool
	notFoundHandler       http.Handler
	methodNotAllowed      http.Handler
	options               RouterOptions
	openAPISpec           *OpenAPISpec
	asyncAPISpec          *AsyncAPISpec
	handlers              map[string]*HandlerInfo
	wsHandlers            map[string]*WSHandlerInfo
	sseHandlers           map[string]*SSEHandlerInfo
	connectionManager     *ConnectionManager
	securityProvider      SecurityProvider
	globalSecurity        []OpenAPISecurityRequirement
	opinionatedMiddleware *MiddlewareChain
}

// RouterOptions holds router configuration
type RouterOptions struct {
	RedirectTrailingSlash  bool
	RedirectFixedPath      bool
	HandleMethodNotAllowed bool
	HandleOPTIONS          bool
	OpenAPITitle           string
	OpenAPIVersion         string
	OpenAPIDescription     string
}

// OpinionatedHandler is the new handler type with automatic OpenAPI generation
type OpinionatedHandler[TInput any, TOutput any] func(ctx *Context, input TInput) (*TOutput, error)

// HandlerInfo stores metadata about registered handlers
type HandlerInfo struct {
	Method               string
	Path                 string
	Summary              string
	Description          string
	Tags                 []string
	InputType            reflect.Type
	OutputType           reflect.Type
	Handler              interface{}
	SecurityRequirements []OpenAPISecurityRequirement
	Deprecated           bool
	OperationID          string
}

// OpenAPISpec OpenAPI Schema Types
type OpenAPISpec struct {
	OpenAPI           string                       `json:"openapi"`
	Info              OpenAPIInfo                  `json:"info"`
	JSONSchemaDialect string                       `json:"jsonSchemaDialect,omitempty"`
	Servers           []OpenAPIServer              `json:"servers,omitempty"`
	Paths             map[string]OpenAPIPath       `json:"paths,omitempty"`
	Webhooks          map[string]OpenAPIPath       `json:"webhooks,omitempty"`
	Components        OpenAPIComponents            `json:"components,omitempty"`
	Security          []OpenAPISecurityRequirement `json:"security,omitempty"`
	Tags              []OpenAPITag                 `json:"tags,omitempty"`
	ExternalDocs      *OpenAPIExternalDocs         `json:"externalDocs,omitempty"`
}

type OpenAPIInfo struct {
	Title          string                 `json:"title"`
	Description    string                 `json:"description,omitempty"`
	TermsOfService string                 `json:"termsOfService,omitempty"`
	Contact        *OpenAPIContact        `json:"contact,omitempty"`
	License        *OpenAPILicense        `json:"license,omitempty"`
	Version        string                 `json:"version"`
	Summary        string                 `json:"summary,omitempty"` // New in 3.1
	Extensions     map[string]interface{} `json:"-"`
}

type OpenAPIContact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

type OpenAPILicense struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier,omitempty"` // New in 3.1 (alternative to url)
	URL        string `json:"url,omitempty"`
}

type OpenAPITag struct {
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	ExternalDocs *OpenAPIExternalDocs `json:"externalDocs,omitempty"`
}

type OpenAPIPath map[string]OpenAPIOperation

type OpenAPIOperation struct {
	Summary      string                       `json:"summary,omitempty"`
	Description  string                       `json:"description,omitempty"`
	Tags         []string                     `json:"tags,omitempty"`
	Parameters   []OpenAPIParameter           `json:"parameters,omitempty"`
	RequestBody  *OpenAPIRequestBody          `json:"requestBody,omitempty"`
	Responses    map[string]OpenAPIResponse   `json:"responses"`
	Security     []OpenAPISecurityRequirement `json:"security,omitempty"`
	Deprecated   bool                         `json:"deprecated,omitempty"`
	OperationID  string                       `json:"operationId,omitempty"`
	ExternalDocs *OpenAPIExternalDocs         `json:"externalDocs,omitempty"`
	Servers      []OpenAPIServer              `json:"servers,omitempty"`
}

type OpenAPIParameter struct {
	Name        string        `json:"name"`
	In          string        `json:"in"` // query, path, header
	Required    bool          `json:"required,omitempty"`
	Description string        `json:"description,omitempty"`
	Schema      OpenAPISchema `json:"schema"`
}

type OpenAPIRequestBody struct {
	Required    bool                        `json:"required,omitempty"`
	Description string                      `json:"description,omitempty"`
	Content     map[string]OpenAPIMediaType `json:"content"`
}

type OpenAPIResponse struct {
	Description string                      `json:"description"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty"`
}

type OpenAPIMediaType struct {
	Schema OpenAPISchema `json:"schema"`
}

type OpenAPISchema struct {
	// Core JSON Schema keywords
	Type        interface{}   `json:"type,omitempty"` // Can be string or array of strings in 3.1
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description,omitempty"`
	Default     interface{}   `json:"default,omitempty"`
	Examples    []interface{} `json:"examples,omitempty"` // Changed from example in 3.0
	Const       interface{}   `json:"const,omitempty"`    // New in 3.1
	Enum        []interface{} `json:"enum,omitempty"`

	// Object validation
	Properties           map[string]OpenAPISchema `json:"properties,omitempty"`
	Required             []string                 `json:"required,omitempty"`
	AdditionalProperties interface{}              `json:"additionalProperties,omitempty"` // Can be bool or schema
	PatternProperties    map[string]OpenAPISchema `json:"patternProperties,omitempty"`    // New in 3.1
	PropertyNames        *OpenAPISchema           `json:"propertyNames,omitempty"`        // New in 3.1
	MinProperties        *int                     `json:"minProperties,omitempty"`
	MaxProperties        *int                     `json:"maxProperties,omitempty"`

	// Array validation
	Items            interface{}     `json:"items,omitempty"`            // Can be schema or array of schemas
	PrefixItems      []OpenAPISchema `json:"prefixItems,omitempty"`      // New in 3.1 (replaces items array)
	UnevaluatedItems interface{}     `json:"unevaluatedItems,omitempty"` // New in 3.1
	Contains         *OpenAPISchema  `json:"contains,omitempty"`         // New in 3.1
	MinContains      *int            `json:"minContains,omitempty"`      // New in 3.1
	MaxContains      *int            `json:"maxContains,omitempty"`      // New in 3.1
	MinItems         *int            `json:"minItems,omitempty"`
	MaxItems         *int            `json:"maxItems,omitempty"`
	UniqueItems      *bool           `json:"uniqueItems,omitempty"`

	// String validation
	MinLength        *int           `json:"minLength,omitempty"`
	MaxLength        *int           `json:"maxLength,omitempty"`
	Pattern          string         `json:"pattern,omitempty"`
	Format           string         `json:"format,omitempty"`
	ContentEncoding  string         `json:"contentEncoding,omitempty"`  // New in 3.1
	ContentMediaType string         `json:"contentMediaType,omitempty"` // New in 3.1
	ContentSchema    *OpenAPISchema `json:"contentSchema,omitempty"`    // New in 3.1

	// Numeric validation
	Minimum          *float64    `json:"minimum,omitempty"`
	Maximum          *float64    `json:"maximum,omitempty"`
	ExclusiveMinimum interface{} `json:"exclusiveMinimum,omitempty"` // Can be bool or number in 3.1
	ExclusiveMaximum interface{} `json:"exclusiveMaximum,omitempty"` // Can be bool or number in 3.1
	MultipleOf       *float64    `json:"multipleOf,omitempty"`

	// Composition keywords
	AllOf []OpenAPISchema `json:"allOf,omitempty"`
	AnyOf []OpenAPISchema `json:"anyOf,omitempty"`
	OneOf []OpenAPISchema `json:"oneOf,omitempty"`
	Not   *OpenAPISchema  `json:"not,omitempty"`

	// Conditional keywords (new in 3.1)
	If                *OpenAPISchema           `json:"if,omitempty"`
	Then              *OpenAPISchema           `json:"then,omitempty"`
	Else              *OpenAPISchema           `json:"else,omitempty"`
	DependentSchemas  map[string]OpenAPISchema `json:"dependentSchemas,omitempty"`
	DependentRequired map[string][]string      `json:"dependentRequired,omitempty"`

	// Reference and evaluation
	Ref           string                   `json:"$ref,omitempty"`
	DynamicRef    string                   `json:"$dynamicRef,omitempty"`    // New in 3.1
	DynamicAnchor string                   `json:"$dynamicAnchor,omitempty"` // New in 3.1
	Defs          map[string]OpenAPISchema `json:"$defs,omitempty"`          // New in 3.1 (preferred over definitions)

	// Unevaluated keywords (new in 3.1)
	UnevaluatedProperties interface{} `json:"unevaluatedProperties,omitempty"`

	// OpenAPI specific extensions (not part of JSON Schema)
	Discriminator *OpenAPIDiscriminator `json:"discriminator,omitempty"`
	XML           *OpenAPIXML           `json:"xml,omitempty"`
	ExternalDocs  *OpenAPIExternalDocs  `json:"externalDocs,omitempty"`

	// Deprecated OpenAPI 3.0 fields (maintained for backward compatibility)
	Nullable *bool       `json:"nullable,omitempty"` // Deprecated in 3.1, use type array
	Example  interface{} `json:"example,omitempty"`  // Deprecated in 3.1, use examples

	// Meta-schema
	Schema  string `json:"$schema,omitempty"`  // JSON Schema dialect
	ID      string `json:"$id,omitempty"`      // Schema identifier
	Anchor  string `json:"$anchor,omitempty"`  // Schema anchor
	Comment string `json:"$comment,omitempty"` // Schema comment
}

type OpenAPIDiscriminator struct {
	PropertyName string            `json:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty"`
}

type OpenAPIXML struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
	Attribute bool   `json:"attribute,omitempty"`
	Wrapped   bool   `json:"wrapped,omitempty"`
}

type OpenAPIComponents struct {
	Schemas         map[string]OpenAPISchema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]OpenAPISecurityScheme `json:"securitySchemes,omitempty"`
	Responses       map[string]OpenAPIResponse       `json:"responses,omitempty"`
	Parameters      map[string]OpenAPIParameter      `json:"parameters,omitempty"`
	RequestBodies   map[string]OpenAPIRequestBody    `json:"requestBodies,omitempty"`
	Headers         map[string]OpenAPIHeader         `json:"headers,omitempty"`
	Examples        map[string]OpenAPIExample        `json:"examples,omitempty"`
	Links           map[string]OpenAPILink           `json:"links,omitempty"`
	Callbacks       map[string]OpenAPICallback       `json:"callbacks,omitempty"`
}

type OpenAPIHeader struct {
	Description string                    `json:"description,omitempty"`
	Required    bool                      `json:"required,omitempty"`
	Deprecated  bool                      `json:"deprecated,omitempty"`
	Schema      OpenAPISchema             `json:"schema,omitempty"`
	Example     interface{}               `json:"example,omitempty"`
	Examples    map[string]OpenAPIExample `json:"examples,omitempty"`
}

type OpenAPIExample struct {
	Summary       string      `json:"summary,omitempty"`
	Description   string      `json:"description,omitempty"`
	Value         interface{} `json:"value,omitempty"`
	ExternalValue string      `json:"externalValue,omitempty"`
}

type OpenAPILink struct {
	OperationRef string                 `json:"operationRef,omitempty"`
	OperationID  string                 `json:"operationId,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	RequestBody  interface{}            `json:"requestBody,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Server       *OpenAPIServer         `json:"server,omitempty"`
}

type OpenAPICallback map[string]OpenAPIPath

type OpenAPIExternalDocs struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

type OpenAPIServer struct {
	URL         string                           `json:"url"`
	Description string                           `json:"description,omitempty"`
	Variables   map[string]OpenAPIServerVariable `json:"variables,omitempty"`
}

type OpenAPIServerVariable struct {
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default"`
	Description string   `json:"description,omitempty"`
}

type MiddlewareFunc func(http.Handler) http.Handler
type HandlerFunc func(http.ResponseWriter, *http.Request)

type Params struct {
	keys   []string
	values []string
}

func (p *Params) Get(key string) string {
	for i, k := range p.keys {
		if k == key {
			return p.values[i]
		}
	}
	return ""
}

func (p *Params) Set(key, value string) {
	for i, k := range p.keys {
		if k == key {
			p.values[i] = value
			return
		}
	}
	p.keys = append(p.keys, key)
	p.values = append(p.values, value)
}

func (p *Params) Reset() {
	p.keys = p.keys[:0]
	p.values = p.values[:0]
}

// NewRouter creates a new SteelRouter instance
func NewRouter() *SteelRouter {
	return &SteelRouter{
		trees: make(map[string]*node),
		pool: sync.Pool{
			New: func() interface{} {
				return &Params{
					keys:   make([]string, 0, 8),
					values: make([]string, 0, 8),
				}
			},
		},
		options: RouterOptions{
			RedirectTrailingSlash:  true,
			RedirectFixedPath:      true,
			HandleMethodNotAllowed: true,
			HandleOPTIONS:          true,
			OpenAPITitle:           "SteelRouter API",
			OpenAPIVersion:         "1.0.0",
			OpenAPIDescription:     "API documentation generated by SteelRouter",
		},
		openAPISpec: &OpenAPISpec{
			OpenAPI: "3.1.1",
			Info: OpenAPIInfo{
				Title:       "SteelRouter API",
				Version:     "1.0.0",
				Description: "API documentation generated by SteelRouter",
			},
			Paths: make(map[string]OpenAPIPath),
			Components: OpenAPIComponents{
				Schemas:         make(map[string]OpenAPISchema),
				SecuritySchemes: make(map[string]OpenAPISecurityScheme),
			},
		},
		handlers:          make(map[string]*HandlerInfo),
		wsHandlers:        make(map[string]*WSHandlerInfo),
		sseHandlers:       make(map[string]*SSEHandlerInfo),
		connectionManager: NewConnectionManager(),
		securityProvider:  NewDefaultSecurityProvider(),
		globalSecurity:    make([]OpenAPISecurityRequirement, 0),
	}
}

// Router interface for consistent API
type Router interface {
	Use(middleware ...MiddlewareFunc)

	UseOpinionated(middleware ...OpinionatedMiddleware)
	UseOpinionatedIf(condition bool, middleware ...OpinionatedMiddleware)

	Group() Router
	GroupFunc(fn func(r Router)) Router
	Route(pattern string, fn func(r Router)) Router
	Mount(pattern string, handler http.Handler)

	GET(pattern string, handler HandlerFunc)
	POST(pattern string, handler HandlerFunc)
	PUT(pattern string, handler HandlerFunc)
	DELETE(pattern string, handler HandlerFunc)
	PATCH(pattern string, handler HandlerFunc)
	HEAD(pattern string, handler HandlerFunc)
	OPTIONS(pattern string, handler HandlerFunc)

	Handle(method, pattern string, handler HandlerFunc)
	HandleFunc(method, pattern string, handler http.HandlerFunc)

	// Opinionated handlers with OpenAPI generation
	OpinionatedGET(pattern string, handler interface{}, opts ...HandlerOption)
	OpinionatedPOST(pattern string, handler interface{}, opts ...HandlerOption)
	OpinionatedPUT(pattern string, handler interface{}, opts ...HandlerOption)
	OpinionatedDELETE(pattern string, handler interface{}, opts ...HandlerOption)
	OpinionatedPATCH(pattern string, handler interface{}, opts ...HandlerOption)

	// Async handlers with AsyncAPI generation
	WebSocket(pattern string, handler interface{}, opts ...AsyncHandlerOption)
	SSE(pattern string, handler interface{}, opts ...AsyncHandlerOption)
}

// HandlerOption for configuring opinionated handlers
type HandlerOption func(*HandlerInfo)

func WithSummary(summary string) HandlerOption {
	return func(h *HandlerInfo) {
		h.Summary = summary
	}
}

func WithDescription(desc string) HandlerOption {
	return func(h *HandlerInfo) {
		h.Description = desc
	}
}

func WithTags(tags ...string) HandlerOption {
	return func(h *HandlerInfo) {
		h.Tags = tags
	}
}

// Ensure SteelRouter implements Router interface
var _ Router = (*SteelRouter)(nil)

// GET Standard HTTP method handlers
func (r *SteelRouter) GET(pattern string, handler HandlerFunc) {
	r.addRoute("GET", pattern, handler)
}

func (r *SteelRouter) POST(pattern string, handler HandlerFunc) {
	r.addRoute("POST", pattern, handler)
}

func (r *SteelRouter) PUT(pattern string, handler HandlerFunc) {
	r.addRoute("PUT", pattern, handler)
}

func (r *SteelRouter) DELETE(pattern string, handler HandlerFunc) {
	r.addRoute("DELETE", pattern, handler)
}

func (r *SteelRouter) PATCH(pattern string, handler HandlerFunc) {
	r.addRoute("PATCH", pattern, handler)
}

func (r *SteelRouter) HEAD(pattern string, handler HandlerFunc) {
	r.addRoute("HEAD", pattern, handler)
}

func (r *SteelRouter) OPTIONS(pattern string, handler HandlerFunc) {
	r.addRoute("OPTIONS", pattern, handler)
}

func (r *SteelRouter) Handle(method, pattern string, handler HandlerFunc) {
	r.addRoute(method, pattern, handler)
}

func (r *SteelRouter) HandleFunc(method, pattern string, handler http.HandlerFunc) {
	r.addRoute(method, pattern, func(w http.ResponseWriter, req *http.Request) {
		handler(w, req)
	})
}

// OpinionatedGET Opinionated handlers with OpenAPI generation
func (r *SteelRouter) OpinionatedGET(pattern string, handler interface{}, opts ...HandlerOption) {
	r.registerOpinionatedHandlerWithMiddleware("GET", pattern, handler, opts...)
}

func (r *SteelRouter) OpinionatedPOST(pattern string, handler interface{}, opts ...HandlerOption) {
	r.registerOpinionatedHandlerWithMiddleware("POST", pattern, handler, opts...)
}

func (r *SteelRouter) OpinionatedPUT(pattern string, handler interface{}, opts ...HandlerOption) {
	r.registerOpinionatedHandlerWithMiddleware("PUT", pattern, handler, opts...)
}

func (r *SteelRouter) OpinionatedDELETE(pattern string, handler interface{}, opts ...HandlerOption) {
	r.registerOpinionatedHandlerWithMiddleware("DELETE", pattern, handler, opts...)
}

func (r *SteelRouter) OpinionatedPATCH(pattern string, handler interface{}, opts ...HandlerOption) {
	r.registerOpinionatedHandlerWithMiddleware("PATCH", pattern, handler, opts...)
}

func (r *SteelRouter) registerOpinionatedHandlerWithMiddleware(method, pattern string, handler interface{}, opts ...HandlerOption) {
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Func {
		panic("handler must be a function")
	}

	if handlerType.NumIn() != 2 || handlerType.NumOut() != 2 {
		panic("handler must have signature func(*Context, InputType) (*OutputType, error)")
	}

	inputType := handlerType.In(1)
	outputType := handlerType.Out(0)

	// Remove pointer from output type for reflection
	if outputType.Kind() == reflect.Ptr {
		outputType = outputType.Elem()
	}

	// Create handler info
	info := &HandlerInfo{
		Method:     method,
		Path:       pattern,
		InputType:  inputType,
		OutputType: outputType,
		Handler:    handler,
	}

	// Apply options
	for _, opt := range opts {
		opt(info)
	}

	// Register in handlers map
	key := method + " " + pattern
	r.handlers[key] = info

	// Generate OpenAPI spec for this handler with middleware enhancements
	r.generateOpenAPIForHandlerWithMiddleware(info)

	// Create wrapper with middleware support
	wrapper := r.createOpinionatedWrapperWithMiddleware(handler, inputType, outputType, info)
	r.addRoute(method, pattern, wrapper)
}

func (r *SteelRouter) generateOpenAPIForHandlerWithMiddleware(info *HandlerInfo) {
	// Generate base operation
	operation := r.generateBaseOperation(info)

	// Add middleware enhancements
	if r.opinionatedMiddleware != nil {
		enhancements := r.opinionatedMiddleware.GetOpenAPIEnhancements()

		// Add security requirements
		if len(enhancements.SecurityRequirements) > 0 {
			if operation.Security == nil {
				operation.Security = make([]OpenAPISecurityRequirement, 0)
			}
			operation.Security = append(operation.Security, enhancements.SecurityRequirements...)
		}

		// Add headers
		for _, header := range enhancements.Headers {
			operation.Parameters = append(operation.Parameters, header)
		}

		// Add responses
		for code, response := range enhancements.Responses {
			operation.Responses[code] = response
		}

		// Add tags
		if len(enhancements.Tags) > 0 {
			operation.Tags = append(operation.Tags, enhancements.Tags...)
		}
	}

	// Add security requirements from handler
	if len(info.SecurityRequirements) > 0 {
		if operation.Security == nil {
			operation.Security = make([]OpenAPISecurityRequirement, 0)
		}
		operation.Security = append(operation.Security, info.SecurityRequirements...)
	} else if len(r.globalSecurity) > 0 {
		// Apply global security if no specific requirements
		if operation.Security == nil {
			operation.Security = make([]OpenAPISecurityRequirement, 0)
		}
		operation.Security = append(operation.Security, r.globalSecurity...)
	}

	// Add to OpenAPI spec
	openAPIPath := r.convertToOpenAPIPath(info.Path)
	if r.openAPISpec.Paths[openAPIPath] == nil {
		r.openAPISpec.Paths[openAPIPath] = make(OpenAPIPath)
	}
	r.openAPISpec.Paths[openAPIPath][strings.ToLower(info.Method)] = operation
}

func (r *SteelRouter) createOpinionatedWrapperWithMiddleware(handler interface{}, inputType, outputType reflect.Type, handlerInfo *HandlerInfo) HandlerFunc {
	handlerValue := reflect.ValueOf(handler)

	return func(w http.ResponseWriter, req *http.Request) {
		// Get parameters from context
		params := ParamsFromContext(req.Context())

		// Create Context
		fastCtx := &Context{
			Request:  req,
			Response: w,
			router:   r,
			params:   params,
		}

		// Create MiddlewareContext
		ctx := &MiddlewareContext{
			Context:         fastCtx,
			StartTime:       time.Now(),
			RequestID:       req.Header.Get("X-Request-ID"),
			Metadata:        make(map[string]interface{}),
			HandlerInfo:     handlerInfo,
			Headers:         make(map[string]string),
			MiddlewareIndex: 0,
		}

		// Initialize input/output values
		ctx.InputValue = reflect.New(inputType)
		ctx.OutputValue = reflect.New(outputType)

		// Create input instance
		input := ctx.InputValue.Interface()

		// Define the final handler
		finalHandler := func() error {
			// Bind parameters to input struct
			if err := r.bindParameters(fastCtx, input); err != nil {
				var apiErr APIError
				if strings.Contains(err.Error(), "body:") &&
					(strings.Contains(err.Error(), "invalid character") ||
						strings.Contains(err.Error(), "cannot unmarshal") ||
						strings.Contains(err.Error(), "unexpected end of JSON input")) {
					apiErr = BadRequest("Invalid JSON in request body", err.Error())
				} else {
					apiErr = UnprocessableEntity("Parameter binding failed",
						NewFieldError("parameters", err.Error(), nil, "BINDING_ERROR"))
				}

				if httpErr, ok := apiErr.(*HTTPError); ok {
					httpErr.Path = req.URL.Path
					httpErr.RequestID = req.Header.Get("X-Request-ID")
				}
				if valErr, ok := apiErr.(*ValidationError); ok {
					valErr.Path = req.URL.Path
					valErr.RequestID = req.Header.Get("X-Request-ID")
				}

				ctx.Error = apiErr
				return apiErr
			}

			// Call the handler
			results := handlerValue.Call([]reflect.Value{
				reflect.ValueOf(fastCtx),
				reflect.ValueOf(input).Elem(),
			})

			// Handle results
			output := results[0]
			err := results[1]

			// Store output in context
			if !output.IsNil() {
				ctx.OutputValue = output
			}

			// Check for errors
			if !err.IsNil() {
				errVal := err.Interface().(error)
				ctx.Error = errVal
				return errVal
			}

			// This line was removed to fix the bug
			// ctx.Processed = true

			return nil
		}

		// Execute middleware chain
		var chainErr error
		if r.opinionatedMiddleware != nil {
			chainErr = r.opinionatedMiddleware.Process(ctx, finalHandler)
		} else {
			chainErr = finalHandler()
		}

		// Handle final response
		if ctx.Processed && chainErr == nil {
			// Response was handled by middleware
			return
		}

		if chainErr != nil {
			// Handle errors
			if apiErr, ok := chainErr.(APIError); ok {
				r.writeErrorResponse(w, req, apiErr)
				return
			}

			// Handle standard Go errors
			internalErr := InternalServerError("An unexpected error occurred")
			internalErr.Path = req.URL.Path
			internalErr.RequestID = req.Header.Get("X-Request-ID")
			internalErr.Detail = chainErr.Error()
			r.writeErrorResponse(w, req, internalErr)
			return
		}

		// Apply any headers set by middleware
		for key, value := range ctx.Headers {
			w.Header().Set(key, value)
		}

		// Handle successful responses
		if !ctx.OutputValue.IsNil() {
			outputVal := ctx.OutputValue.Interface()

			// Check if it's an APIResponse (custom status code)
			if apiResp, ok := outputVal.(*APIResponse); ok {
				// Set custom headers
				for key, value := range apiResp.Headers {
					w.Header().Set(key, value)
				}

				// Set status code
				if ctx.StatusCode != 0 {
					w.WriteHeader(ctx.StatusCode)
				} else {
					w.WriteHeader(apiResp.StatusCode)
				}

				// Write response body if data exists
				if apiResp.Data != nil {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(apiResp.Data)
				}
				return
			}

			// Standard successful response
			if ctx.StatusCode != 0 {
				w.WriteHeader(ctx.StatusCode)
			}
			fastCtx.JSON(http.StatusOK, outputVal)
		} else {
			// No content response
			if ctx.StatusCode != 0 {
				w.WriteHeader(ctx.StatusCode)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
		}
	}
}

// Register opinionated handler with reflection and OpenAPI generation
func (r *SteelRouter) registerOpinionatedHandler(method, pattern string, handler interface{}, opts ...HandlerOption) {
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Func {
		panic("handler must be a function")
	}

	if handlerType.NumIn() != 2 || handlerType.NumOut() != 2 {
		panic("handler must have signature func(*Context, InputType) (*OutputType, error)")
	}

	inputType := handlerType.In(1)
	outputType := handlerType.Out(0)

	// Remove pointer from output type for reflection
	if outputType.Kind() == reflect.Ptr {
		outputType = outputType.Elem()
	}

	// Create handler info
	info := &HandlerInfo{
		Method:     method,
		Path:       pattern,
		InputType:  inputType,
		OutputType: outputType,
		Handler:    handler,
	}

	// Apply options
	for _, opt := range opts {
		opt(info)
	}

	// Register in handlers map
	key := method + " " + pattern
	r.handlers[key] = info

	// Generate OpenAPI spec for this handler
	r.generateOpenAPIForHandler(info)

	// Create wrapper for standard router
	wrapper := r.createOpinionatedWrapper(handler, inputType, outputType)
	r.addRoute(method, pattern, wrapper)
}

// Create wrapper that handles parameter binding and validation
func (r *SteelRouter) createOpinionatedWrapper(handler interface{}, inputType, outputType reflect.Type) HandlerFunc {
	handlerValue := reflect.ValueOf(handler)

	return func(w http.ResponseWriter, req *http.Request) {
		// Get parameters from context, which were populated by ServeHTTP's findHandler
		params := ParamsFromContext(req.Context())

		// Create Context with enhanced error handling
		ctx := &Context{
			Request:  req,
			Response: w,
			router:   r,
			params:   params,
		}

		// Create input instance
		input := reflect.New(inputType).Interface()

		// Bind parameters to input struct
		if err := r.bindParameters(ctx, input); err != nil {
			// Determine appropriate error status based on the error type
			var apiErr APIError

			// Check if it's a JSON parsing error (should be 400 Bad Request)
			if strings.Contains(err.Error(), "body:") &&
				(strings.Contains(err.Error(), "invalid character") ||
					strings.Contains(err.Error(), "cannot unmarshal") ||
					strings.Contains(err.Error(), "unexpected end of JSON input")) {
				apiErr = BadRequest("Invalid JSON in request body", err.Error())
			} else {
				// Other binding errors are validation errors (422)
				apiErr = UnprocessableEntity("Parameter binding failed",
					NewFieldError("parameters", err.Error(), nil, "BINDING_ERROR"))
			}

			if httpErr, ok := apiErr.(*HTTPError); ok {
				httpErr.Path = req.URL.Path
				httpErr.RequestID = req.Header.Get("X-Request-ID")
			}
			if valErr, ok := apiErr.(*ValidationError); ok {
				valErr.Path = req.URL.Path
				valErr.RequestID = req.Header.Get("X-Request-ID")
			}

			r.writeErrorResponse(w, req, apiErr)
			return
		}

		// Call the handler
		results := handlerValue.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(input).Elem(),
		})

		// Handle results
		output := results[0]
		err := results[1]

		// Check for errors first
		if !err.IsNil() {
			errVal := err.Interface().(error)
			r.handleError(w, req, errVal)
			return
		}

		// Handle successful responses
		if !output.IsNil() {
			outputVal := output.Interface()

			// Check if it's an APIResponse (custom status code)
			if apiResp, ok := outputVal.(*APIResponse); ok {
				// Set custom headers
				for key, value := range apiResp.Headers {
					w.Header().Set(key, value)
				}

				// Set status code
				w.WriteHeader(apiResp.StatusCode)

				// Write response body if data exists
				if apiResp.Data != nil {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(apiResp.Data)
				}
				return
			}

			// Standard successful response
			ctx.JSON(http.StatusOK, outputVal)
		} else {
			// No content response
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

// Enhanced error handling
func (r *SteelRouter) handleError(w http.ResponseWriter, req *http.Request, err error) {
	// Check if it's an APIError
	if apiErr, ok := err.(APIError); ok {
		// Enrich with request context
		if httpErr, ok := apiErr.(*HTTPError); ok {
			if httpErr.Path == "" {
				httpErr.Path = req.URL.Path
			}
			if httpErr.RequestID == "" {
				httpErr.RequestID = req.Header.Get("X-Request-ID")
			}
		}
		r.writeErrorResponse(w, req, apiErr)
		return
	}

	// Handle standard Go errors
	internalErr := InternalServerError("An unexpected error occurred")
	internalErr.Path = req.URL.Path
	internalErr.RequestID = req.Header.Get("X-Request-ID")
	internalErr.Detail = err.Error() // Include original error in detail for debugging

	r.writeErrorResponse(w, req, internalErr)
}

// Write structured error response
func (r *SteelRouter) writeErrorResponse(w http.ResponseWriter, req *http.Request, apiErr APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.StatusCode())

	response := apiErr.ToResponse()
	json.NewEncoder(w).Encode(response)
}

// Bind parameters from request to struct based on tags
func (r *SteelRouter) bindParameters(ctx *Context, input interface{}) error {
	val := reflect.ValueOf(input).Elem()
	typ := val.Type()

	// Bind the JSON body if applicable.
	// Allow body on GET for flexibility, though not standard.
	if ctx.Request.Method == "POST" || ctx.Request.Method == "PUT" || ctx.Request.Method == "PATCH" || ctx.Request.Method == "GET" {
		if ctx.Request.Body != nil && ctx.Request.Body != http.NoBody {
			contentType := ctx.Request.Header.Get("Content-Type")
			if strings.Contains(contentType, "application/json") {

				bodyHandled := false
				// First, check if a specific field is designated to receive the body
				for i := 0; i < val.NumField(); i++ {
					field := val.Field(i)
					fieldType := typ.Field(i)

					if bodyTag := fieldType.Tag.Get("body"); bodyTag != "" && field.CanSet() {
						fieldValue := reflect.New(field.Type())
						if err := ctx.BindJSON(fieldValue.Interface()); err != nil {
							if err != io.EOF {
								return fmt.Errorf("body: %v", err)
							}
						} else {
							field.Set(fieldValue.Elem())
						}
						bodyHandled = true
						break // Assume only one field is the body
					}
				}

				// If no specific body field was found, bind to the whole struct
				if !bodyHandled {
					if err := ctx.BindJSON(input); err != nil {
						if err != io.EOF {
							return fmt.Errorf("body: %v", err)
						}
					}
				}
			}
		}
	}

	// Handle path, query, and header parameters.
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanSet() {
			continue
		}

		// Bind from path
		if paramTag := fieldType.Tag.Get("path"); paramTag != "" {
			value := ctx.Param(paramTag)
			if value != "" {
				if err := r.setFieldValue(field, value); err != nil {
					return fmt.Errorf("path parameter %s: %v", paramTag, err)
				}
			}
		}

		// Bind from query
		if queryTag := fieldType.Tag.Get("query"); queryTag != "" {
			value := ctx.Query(queryTag)
			if value != "" {
				if err := r.setFieldValue(field, value); err != nil {
					return fmt.Errorf("query parameter %s: %v", queryTag, err)
				}
			}
		}

		// Bind from header
		if headerTag := fieldType.Tag.Get("header"); headerTag != "" {
			value := ctx.Header(headerTag)
			if value != "" {
				if err := r.setFieldValue(field, value); err != nil {
					return fmt.Errorf("header %s: %v", headerTag, err)
				}
			}
		}
	}

	return nil
}

// Set field value with type conversion
func (r *SteelRouter) setFieldValue(field reflect.Value, value string) error {
	if value == "" {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	default:
		return fmt.Errorf("unsupported type: %v", field.Kind())
	}

	return nil
}

// Generate OpenAPI specification for handler
func (r *SteelRouter) generateOpenAPIForHandler(info *HandlerInfo) {
	operation := OpenAPIOperation{
		Summary:     info.Summary,
		Description: info.Description,
		Tags:        info.Tags,
		Parameters:  []OpenAPIParameter{},
		Responses:   make(map[string]OpenAPIResponse),
		Deprecated:  info.Deprecated,
		OperationID: info.OperationID,
	}

	// Add security requirements
	if len(info.SecurityRequirements) > 0 {
		operation.Security = info.SecurityRequirements
	} else if len(r.globalSecurity) > 0 {
		// Apply global security if no specific requirements
		operation.Security = r.globalSecurity
	}

	// Validate security requirements if provider exists
	if r.securityProvider != nil {
		for _, req := range operation.Security {
			if err := r.securityProvider.ValidateSecurityRequirement(req); err != nil {
				// Log warning - in production you might want to use a proper logger
				fmt.Printf("Warning: Invalid security requirement for %s %s: %v\n",
					info.Method, info.Path, err)
			}
		}
	}

	// Generate parameters from input struct (existing logic)
	hasBodyParams := false
	var bodySchema OpenAPISchema

	if info.InputType.Kind() == reflect.Struct {
		// Check for body parameters
		for i := 0; i < info.InputType.NumField(); i++ {
			field := info.InputType.Field(i)
			if bodyTag := field.Tag.Get("body"); bodyTag != "" {
				hasBodyParams = true
				bodySchema = r.typeToSchema(info.InputType)
				break
			}
		}

		// Handle other parameter types
		for i := 0; i < info.InputType.NumField(); i++ {
			field := info.InputType.Field(i)

			// Path parameters
			if paramTag := field.Tag.Get("path"); paramTag != "" {
				param := OpenAPIParameter{
					Name:     paramTag,
					In:       "path",
					Required: true,
					Schema:   r.typeToSchema(field.Type),
				}
				if desc := field.Tag.Get("description"); desc != "" {
					param.Description = desc
				}
				operation.Parameters = append(operation.Parameters, param)
			}

			// Query parameters
			if queryTag := field.Tag.Get("query"); queryTag != "" {
				param := OpenAPIParameter{
					Name:     queryTag,
					In:       "query",
					Required: field.Tag.Get("required") == "true",
					Schema:   r.typeToSchema(field.Type),
				}
				if desc := field.Tag.Get("description"); desc != "" {
					param.Description = desc
				}
				operation.Parameters = append(operation.Parameters, param)
			}

			// Header parameters
			if headerTag := field.Tag.Get("header"); headerTag != "" {
				param := OpenAPIParameter{
					Name:     headerTag,
					In:       "header",
					Required: field.Tag.Get("required") == "true",
					Schema:   r.typeToSchema(field.Type),
				}
				if desc := field.Tag.Get("description"); desc != "" {
					param.Description = desc
				}
				operation.Parameters = append(operation.Parameters, param)
			}
		}
	}

	// Add request body if we have body parameters
	if hasBodyParams {
		operation.RequestBody = &OpenAPIRequestBody{
			Required: true,
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: bodySchema,
				},
			},
		}
	}

	// Generate success response schema
	if info.OutputType.Kind() == reflect.Struct {
		operation.Responses["200"] = OpenAPIResponse{
			Description: "Success",
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: r.typeToSchema(info.OutputType),
				},
			},
		}
	} else {
		operation.Responses["204"] = OpenAPIResponse{
			Description: "No Content",
		}
	}

	// Add standard error responses (including security-related ones)
	r.addStandardErrorResponses(&operation, info.Method)

	// Add to OpenAPI spec
	openAPIPath := r.convertToOpenAPIPath(info.Path)
	if r.openAPISpec.Paths[openAPIPath] == nil {
		r.openAPISpec.Paths[openAPIPath] = make(OpenAPIPath)
	}
	r.openAPISpec.Paths[openAPIPath][strings.ToLower(info.Method)] = operation
}

// convertToOpenAPIPath Helper function to convert internal path format to OpenAPI format
func (r *SteelRouter) convertToOpenAPIPath(path string) string {
	// Convert :param to {param} format for OpenAPI
	result := ""
	segments := strings.Split(path, "/")

	for i, segment := range segments {
		if i > 0 {
			result += "/"
		}

		if len(segment) > 0 && segment[0] == ':' {
			// Convert :param to {param}
			result += "{" + segment[1:] + "}"
		} else {
			result += segment
		}
	}

	return result
}

// Add standard error responses to operation
func (r *SteelRouter) addStandardErrorResponses(operation *OpenAPIOperation, method string) {
	// Register error schemas in components if not already present
	r.ensureErrorSchemasRegistered()

	// 400 Bad Request
	operation.Responses["400"] = OpenAPIResponse{
		Description: "Bad Request - Invalid input parameters",
		Content: map[string]OpenAPIMediaType{
			"application/json": {
				Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"},
			},
		},
	}

	// 401 Unauthorized
	operation.Responses["401"] = OpenAPIResponse{
		Description: "Unauthorized - Authentication required",
		Content: map[string]OpenAPIMediaType{
			"application/json": {
				Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"},
			},
		},
	}

	// 403 Forbidden
	operation.Responses["403"] = OpenAPIResponse{
		Description: "Forbidden - Access denied",
		Content: map[string]OpenAPIMediaType{
			"application/json": {
				Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"},
			},
		},
	}

	// 404 Not Found (for methods that access specific resources)
	if method == "GET" || method == "PUT" || method == "DELETE" || method == "PATCH" {
		operation.Responses["404"] = OpenAPIResponse{
			Description: "Not Found - Resource does not exist",
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"},
				},
			},
		}
	}

	// 409 Conflict (for POST and PUT)
	if method == "POST" || method == "PUT" {
		operation.Responses["409"] = OpenAPIResponse{
			Description: "Conflict - Resource already exists or conflict with current state",
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"},
				},
			},
		}
	}

	// 422 Unprocessable Entity (validation errors)
	operation.Responses["422"] = OpenAPIResponse{
		Description: "Unprocessable Entity - Validation failed",
		Content: map[string]OpenAPIMediaType{
			"application/json": {
				Schema: OpenAPISchema{Ref: "#/components/schemas/ValidationErrorResponse"},
			},
		},
	}

	// 429 Too Many Requests
	operation.Responses["429"] = OpenAPIResponse{
		Description: "Too Many Requests - Rate limit exceeded",
		Content: map[string]OpenAPIMediaType{
			"application/json": {
				Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"},
			},
		},
	}

	// 500 Internal Server Error
	operation.Responses["500"] = OpenAPIResponse{
		Description: "Internal Server Error - Unexpected server error",
		Content: map[string]OpenAPIMediaType{
			"application/json": {
				Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"},
			},
		},
	}

	// 503 Service Unavailable
	operation.Responses["503"] = OpenAPIResponse{
		Description: "Service Unavailable - Service temporarily unavailable",
		Content: map[string]OpenAPIMediaType{
			"application/json": {
				Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"},
			},
		},
	}
}

// Ensure error schemas are registered in components
func (r *SteelRouter) ensureErrorSchemasRegistered() {
	// ErrorResponse schema (OpenAPI 3.1.1 compliant)
	if _, exists := r.openAPISpec.Components.Schemas["ErrorResponse"]; !exists {
		r.openAPISpec.Components.Schemas["ErrorResponse"] = OpenAPISchema{
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"error": {Ref: "#/components/schemas/ErrorDetail"},
			},
			Required:    []string{"error"},
			Description: "Standard error response format",
			Examples: []interface{}{
				map[string]interface{}{
					"error": map[string]interface{}{
						"status":    400,
						"code":      "BAD_REQUEST",
						"message":   "Invalid input provided",
						"timestamp": "2023-12-25T10:30:00Z",
					},
				},
			},
		}
	}

	// ErrorDetail schema with proper 3.1.1 constraints
	if _, exists := r.openAPISpec.Components.Schemas["ErrorDetail"]; !exists {
		r.openAPISpec.Components.Schemas["ErrorDetail"] = OpenAPISchema{
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"status": {
					Type:        "integer",
					Minimum:     float64Ptr(100),
					Maximum:     float64Ptr(599),
					Description: "HTTP status code",
					Examples:    []interface{}{400, 401, 403, 404, 500},
				},
				"code": {
					Type:        "string",
					Description: "Error code for programmatic handling",
					Examples:    []interface{}{"BAD_REQUEST", "UNAUTHORIZED", "NOT_FOUND"},
					Pattern:     "^[A-Z_]+$",
				},
				"message": {
					Type:        "string",
					Description: "Human-readable error message",
					MinLength:   intPtr(1),
					Examples:    []interface{}{"Invalid input provided", "Authentication required"},
				},
				"detail": {
					Description: "Additional error details",
					Examples:    []interface{}{"Field 'email' is required", nil},
				},
				"timestamp": {
					Type:        "string",
					Format:      "date-time",
					Description: "Error timestamp in RFC3339 format",
					Examples:    []interface{}{"2023-12-25T10:30:00Z"},
				},
				"request_id": {
					Type:        []interface{}{"string", "null"},
					Description: "Request tracking ID",
					Examples:    []interface{}{"req_123456789", nil},
					Pattern:     "^req_[a-zA-Z0-9]+$",
				},
				"path": {
					Type:        []interface{}{"string", "null"},
					Description: "Request path that caused the error",
					Examples:    []interface{}{"/api/users", "/api/orders/123"},
				},
			},
			Required:    []string{"status", "code", "message", "timestamp"},
			Description: "Detailed error information",
		}
	}

	// ValidationErrorResponse with enhanced field validation
	if _, exists := r.openAPISpec.Components.Schemas["ValidationErrorResponse"]; !exists {
		r.openAPISpec.Components.Schemas["ValidationErrorResponse"] = OpenAPISchema{
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"error": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"status": {
							Const:       422,
							Description: "HTTP status code",
						},
						"code": {
							Const:       "VALIDATION_FAILED",
							Description: "Error code",
						},
						"message": {
							Type:        "string",
							Description: "Error message",
							Examples:    []interface{}{"Validation failed"},
						},
						"detail": {
							Type: "array",
							Items: OpenAPISchema{
								Ref: "#/components/schemas/FieldError",
							},
							Description: "Field-specific validation errors",
							MinItems:    intPtr(1),
						},
						"timestamp": {
							Type:   "string",
							Format: "date-time",
						},
						"request_id": {Type: []interface{}{"string", "null"}},
						"path":       {Type: []interface{}{"string", "null"}},
					},
					Required: []string{"status", "code", "message", "detail", "timestamp"},
				},
			},
			Required:    []string{"error"},
			Description: "Validation error response with field-specific details",
		}
	}

	// FieldError schema with comprehensive validation
	if _, exists := r.openAPISpec.Components.Schemas["FieldError"]; !exists {
		r.openAPISpec.Components.Schemas["FieldError"] = OpenAPISchema{
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"field": {
					Type:        "string",
					Description: "Field name that failed validation",
					MinLength:   intPtr(1),
					Examples:    []interface{}{"email", "user.name", "items[0].price"},
				},
				"message": {
					Type:        "string",
					Description: "Validation error message",
					MinLength:   intPtr(1),
					Examples:    []interface{}{"Invalid email format", "Field is required"},
				},
				"value": {
					Description: "Value that failed validation",
					Examples:    []interface{}{"invalid-email", "", 123},
				},
				"code": {
					Type:        []interface{}{"string", "null"},
					Description: "Validation error code",
					Examples:    []interface{}{"INVALID_FORMAT", "REQUIRED", "OUT_OF_RANGE"},
					Pattern:     "^[A-Z_]+$",
				},
			},
			Required:    []string{"field", "message"},
			Description: "Field-specific validation error",
		}
	}
}

// Convert Go type to OpenAPI schema
func (r *SteelRouter) typeToSchema(t reflect.Type) OpenAPISchema {
	// Handle pointer types by dereferencing
	if t.Kind() == reflect.Ptr {
		return r.typeToSchema(t.Elem())
	}

	// Handle special named types first
	if t.PkgPath() != "" && t.Name() != "" {
		switch t.String() {
		case "time.Time":
			return OpenAPISchema{
				Type:        "string",
				Format:      "date-time",
				Description: "RFC3339 date-time format",
				Examples:    []interface{}{"2023-12-25T10:30:00Z"},
			}
		case "time.Duration":
			return OpenAPISchema{
				Type:        "string",
				Description: "Duration in Go format (e.g., '1h30m', '5s')",
				Examples:    []interface{}{"1h30m", "5s", "100ms"},
				Pattern:     `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`,
			}
		}

		// Handle other standard library types
		switch t.PkgPath() {
		case "net/url":
			if t.Name() == "URL" {
				return OpenAPISchema{
					Type:     "string",
					Format:   "uri",
					Examples: []interface{}{"https://example.com/path?query=value"},
				}
			}
		case "net":
			if t.Name() == "IP" {
				return OpenAPISchema{
					AnyOf: []OpenAPISchema{
						{Type: "string", Format: "ipv4", Examples: []interface{}{"192.168.1.1"}},
						{Type: "string", Format: "ipv6", Examples: []interface{}{"2001:db8::1"}},
					},
				}
			}
		case "encoding/json":
			if t.Name() == "RawMessage" {
				return OpenAPISchema{
					Description: "Raw JSON data",
					Examples:    []interface{}{map[string]interface{}{"key": "value"}},
				}
			}
		}
	}

	switch t.Kind() {
	case reflect.String:
		return OpenAPISchema{Type: "string"}

	// Integer types - use proper JSON Schema type arrays for nullable types
	case reflect.Int:
		return OpenAPISchema{Type: "integer", Format: "int64"}
	case reflect.Int8:
		return OpenAPISchema{
			Type:        "integer",
			Minimum:     float64Ptr(-128),
			Maximum:     float64Ptr(127),
			Description: "8-bit signed integer",
		}
	case reflect.Int16:
		return OpenAPISchema{
			Type:        "integer",
			Minimum:     float64Ptr(-32768),
			Maximum:     float64Ptr(32767),
			Description: "16-bit signed integer",
		}
	case reflect.Int32:
		return OpenAPISchema{Type: "integer", Format: "int32"}
	case reflect.Int64:
		return OpenAPISchema{Type: "integer", Format: "int64"}

	// Unsigned integer types
	case reflect.Uint:
		return OpenAPISchema{
			Type:        "integer",
			Minimum:     float64Ptr(0),
			Description: "Unsigned integer",
		}
	case reflect.Uint8:
		return OpenAPISchema{
			Type:        "integer",
			Minimum:     float64Ptr(0),
			Maximum:     float64Ptr(255),
			Description: "8-bit unsigned integer (byte)",
		}
	case reflect.Uint16:
		return OpenAPISchema{
			Type:        "integer",
			Minimum:     float64Ptr(0),
			Maximum:     float64Ptr(65535),
			Description: "16-bit unsigned integer",
		}
	case reflect.Uint32:
		return OpenAPISchema{
			Type:        "integer",
			Format:      "int64",
			Minimum:     float64Ptr(0),
			Description: "32-bit unsigned integer",
		}
	case reflect.Uint64:
		return OpenAPISchema{
			Type:        "integer",
			Format:      "int64",
			Minimum:     float64Ptr(0),
			Description: "64-bit unsigned integer",
		}

	// Floating point types
	case reflect.Float32:
		return OpenAPISchema{Type: "number", Format: "float"}
	case reflect.Float64:
		return OpenAPISchema{Type: "number", Format: "double"}

	case reflect.Bool:
		return OpenAPISchema{Type: "boolean"}

	// Collection types
	case reflect.Slice, reflect.Array:
		elemSchema := r.typeToSchema(t.Elem())
		schema := OpenAPISchema{
			Type:  "array",
			Items: elemSchema,
		}

		// Special handling for byte slices
		if t.Elem().Kind() == reflect.Uint8 {
			schema.Description = "Base64 encoded binary data"
			schema.ContentEncoding = "base64"
			schema.Type = "string"
			schema.Items = nil
		}

		return schema

	case reflect.Map:
		keyType := t.Key()
		valueSchema := r.typeToSchema(t.Elem())

		// OpenAPI 3.1.1 supports string keys in objects
		if keyType.Kind() == reflect.String {
			return OpenAPISchema{
				Type:                 "object",
				AdditionalProperties: valueSchema,
				Description:          "Map with string keys",
			}
		} else {
			// For non-string keys, use patternProperties or represent as array
			return OpenAPISchema{
				Type: "array",
				Items: OpenAPISchema{
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"key":   r.typeToSchema(keyType),
						"value": valueSchema,
					},
					Required: []string{"key", "value"},
				},
				Description: "Map represented as array of key-value pairs",
			}
		}

	case reflect.Struct:
		// Check if this is a named type that should be a component reference
		if t.Name() != "" {
			schemaName := t.Name()

			// Register the schema in components if not already present
			if _, exists := r.openAPISpec.Components.Schemas[schemaName]; !exists {
				// Temporarily set a placeholder to prevent infinite recursion
				r.openAPISpec.Components.Schemas[schemaName] = OpenAPISchema{Type: "object"}

				// Generate the actual schema
				schema := r.generateStructSchema(t)
				r.openAPISpec.Components.Schemas[schemaName] = schema
			}

			// Return a reference to the component
			return OpenAPISchema{
				Ref: "#/components/schemas/" + schemaName,
			}
		}

		// For anonymous structs, generate inline schema
		return r.generateStructSchema(t)

	case reflect.Interface:
		// Handle interface{} and any types - OpenAPI 3.1.1 way
		if t.NumMethod() == 0 { // empty interface
			return OpenAPISchema{
				Description: "Any value (interface{})",
				// In OpenAPI 3.1.1, omitting type allows any type
			}
		}
		// For non-empty interfaces
		return OpenAPISchema{
			Type:        "object",
			Description: "Interface type - actual structure may vary",
		}

	default:
		return OpenAPISchema{
			Description: "Unknown type, represented as any value",
			// No type specified allows any value in 3.1.1
		}
	}
}

// Enhanced generateStructSchema with better field handling
func (r *SteelRouter) generateStructSchema(t reflect.Type) OpenAPISchema {
	schema := OpenAPISchema{
		Type:       "object",
		Properties: make(map[string]OpenAPISchema),
		Required:   []string{},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Handle embedded fields
		if field.Anonymous {
			if field.Type.Kind() == reflect.Struct || (field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct) {
				embeddedSchema := r.typeToSchema(field.Type)

				// In OpenAPI 3.1.1, we can use allOf for composition
				if len(schema.Properties) == 0 && len(schema.Required) == 0 {
					// First embedded struct, use its properties directly
					if embeddedSchema.Properties != nil {
						for propName, propSchema := range embeddedSchema.Properties {
							schema.Properties[propName] = propSchema
						}
					}
					if embeddedSchema.Required != nil {
						schema.Required = append(schema.Required, embeddedSchema.Required...)
					}
				} else {
					// Multiple embedded structs, use allOf
					if schema.AllOf == nil {
						// Convert current schema to allOf
						currentSchema := OpenAPISchema{
							Type:       "object",
							Properties: schema.Properties,
							Required:   schema.Required,
						}
						schema.AllOf = []OpenAPISchema{currentSchema}
						schema.Properties = nil
						schema.Required = nil
					}
					schema.AllOf = append(schema.AllOf, embeddedSchema)
				}
			}
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse json tag
		parts := strings.Split(jsonTag, ",")
		fieldName := parts[0]
		if fieldName == "" {
			fieldName = field.Name
		}

		// Check for omitempty option
		omitEmpty := false
		for _, part := range parts[1:] {
			if part == "omitempty" {
				omitEmpty = true
				break
			}
		}

		// Generate schema for field type
		fieldSchema := r.typeToSchema(field.Type)

		// Add description from tag if present
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema.Description = desc
		}

		// Add examples from tag if present (OpenAPI 3.1.1 uses examples array)
		if example := field.Tag.Get("example"); example != "" {
			fieldSchema.Examples = []interface{}{example}
		}

		// Add format from tag if present
		if format := field.Tag.Get("format"); format != "" {
			fieldSchema.Format = format
		}

		// Add validation constraints
		if min := field.Tag.Get("min"); min != "" {
			if minVal, err := strconv.ParseFloat(min, 64); err == nil {
				fieldSchema.Minimum = &minVal
			}
		}
		if max := field.Tag.Get("max"); max != "" {
			if maxVal, err := strconv.ParseFloat(max, 64); err == nil {
				fieldSchema.Maximum = &maxVal
			}
		}
		if pattern := field.Tag.Get("pattern"); pattern != "" {
			fieldSchema.Pattern = pattern
		}

		// Handle nullable fields (OpenAPI 3.1.1 style)
		if field.Type.Kind() == reflect.Ptr {
			// For pointer types, allow null in addition to the base type
			baseType := fieldSchema.Type
			if baseType != nil {
				fieldSchema.Type = []interface{}{baseType, "null"}
			}
		}

		schema.Properties[fieldName] = fieldSchema

		// Check if field is required
		isRequired := field.Tag.Get("required") == "true"
		if !isRequired && !omitEmpty && field.Type.Kind() != reflect.Ptr {
			isRequired = true
		}

		if isRequired {
			schema.Required = append(schema.Required, fieldName)
		}
	}
	return schema
}

// EnableOpenAPI mount OpenAPI documentation
func (r *SteelRouter) EnableOpenAPI() {
	// Serve OpenAPI spec
	r.GET("/openapi.json", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(r.openAPISpec)
	})

	// Swagger UI
	r.addSwaggerUIEndpoint()

	// Stoplight Elements
	r.GET("/openapi/spotlight", func(w http.ResponseWriter, req *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation - Stoplight Elements</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <script src="https://unpkg.com/@stoplight/elements/web-components.min.js"></script>
    <link rel="stylesheet" href="https://unpkg.com/@stoplight/elements/styles.min.css">
</head>
<body>
    <elements-api
        apiDescriptionUrl="/openapi.json"
        router="hash"
        layout="sidebar"
    />
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// Scalar
	r.GET("/openapi/scalar", func(w http.ResponseWriter, req *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation - Scalar</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
    <script
        id="api-reference"
        data-url="/openapi.json"
        data-configuration='{
            "theme": "default",
            "layout": "modern",
            "showSidebar": true,
            "hideModels": false,
            "searchHotKey": "k"
        }'
    ></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// ReDoc
	r.GET("/openapi/redoc", func(w http.ResponseWriter, req *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation - ReDoc</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
    </style>
</head>
<body>
    <redoc spec-url="/openapi.json"></redoc>
    <script src="https://cdn.jsdelivr.net/npm/redoc@2.1.3/bundles/redoc.standalone.js"></script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// Documentation index page
	r.GET("/openapi/docs", func(w http.ResponseWriter, req *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 2rem;
            background-color: #f8f9fa;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: white;
            padding: 2rem;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            margin-bottom: 1rem;
        }
        .docs-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 1rem;
            margin-top: 2rem;
        }
        .doc-card {
            border: 1px solid #e9ecef;
            border-radius: 6px;
            padding: 1.5rem;
            text-decoration: none;
            color: #333;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .doc-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 8px rgba(0,0,0,0.1);
            text-decoration: none;
        }
        .doc-card h3 {
            margin-top: 0;
            color: #007bff;
        }
        .doc-card p {
            margin-bottom: 0;
            color: #666;
        }
        .spec-link {
            display: inline-block;
            margin-top: 1rem;
            padding: 0.5rem 1rem;
            background: #007bff;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            transition: background 0.2s;
        }
        .spec-link:hover {
            background: #0056b3;
            text-decoration: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>API Documentation</h1>
        <p>Choose your preferred documentation viewer:</p>
        
        <div class="docs-grid">
            <a href="/openapi/swagger" class="doc-card">
                <h3>Swagger UI</h3>
                <p>Interactive API documentation with try-it-out functionality</p>
            </a>
            
            <a href="/openapi/redoc" class="doc-card">
                <h3>ReDoc</h3>
                <p>Beautiful, responsive API documentation</p>
            </a>
            
            <a href="/openapi/scalar" class="doc-card">
                <h3>Scalar</h3>
                <p>Modern, fast API documentation with excellent UX</p>
            </a>
            
            <a href="/openapi/spotlight" class="doc-card">
                <h3>Stoplight Elements</h3>
                <p>Comprehensive API documentation with advanced features</p>
            </a>
        </div>
        
        <a href="/openapi.json" class="spec-link">View Raw OpenAPI Spec</a>
    </div>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})
}

// Use Standard router implementation (keeping existing functionality)
func (r *SteelRouter) Use(middleware ...MiddlewareFunc) {
	r.middleware = append(r.middleware, middleware...)
}

// Group creates a new route group with empty prefix
func (r *SteelRouter) Group() Router {
	return &RouteGroup{
		router:     r,
		prefix:     "",
		middleware: make([]MiddlewareFunc, 0),
	}
}

// GroupFunc creates a route group and calls the provided function with it (renamed from Group)
func (r *SteelRouter) GroupFunc(fn func(r Router)) Router {
	group := &RouteGroup{
		router:     r,
		prefix:     "",
		middleware: make([]MiddlewareFunc, 0),
	}
	fn(group)
	return group
}

func (r *SteelRouter) Route(pattern string, fn func(r Router)) Router {
	group := &RouteGroup{
		router: r,
		prefix: pattern,
	}
	fn(group)
	return group
}

func (r *SteelRouter) Mount(pattern string, handler http.Handler) {
	mountHandler := http.StripPrefix(pattern, handler)

	// The pattern for the router should match all sub-paths.
	if !strings.HasSuffix(pattern, "/") {
		pattern += "/"
	}
	fullPattern := pattern + "*"

	// The wrapper function to be registered with the router.
	mountFunc := func(w http.ResponseWriter, req *http.Request) {
		mountHandler.ServeHTTP(w, req)
	}

	// Mount should work for any method.
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, method := range methods {
		r.addRoute(method, fullPattern, mountFunc)
	}
}

func (r *SteelRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	params := r.pool.Get().(*Params)
	params.Reset()
	defer r.pool.Put(params)

	path := req.URL.Path
	method := req.Method

	// Clean the path
	if path != "/" {
		path = cleanPath(path)
	}

	handler := r.findHandler(method, path, params)

	// If no handler found, try trailing slash redirection
	if handler == nil && r.options.RedirectTrailingSlash {
		var redirectPath string
		var redirectHandler HandlerFunc

		if len(path) > 1 && path[len(path)-1] == '/' {
			// Try without trailing slash
			redirectPath = path[:len(path)-1]
			params.Reset()
			redirectHandler = r.findHandler(method, redirectPath, params)
		} else {
			// Try with trailing slash
			redirectPath = path + "/"
			params.Reset()
			redirectHandler = r.findHandler(method, redirectPath, params)
		}

		if redirectHandler != nil {
			// Redirect to the correct path
			query := req.URL.RawQuery
			if query != "" {
				redirectPath += "?" + query
			}

			statusCode := http.StatusMovedPermanently
			if method != "GET" {
				statusCode = http.StatusPermanentRedirect
			}

			w.Header().Set("Location", redirectPath)
			w.WriteHeader(statusCode)
			return
		}
	}

	// If still no handler found, try fixed path redirection
	if handler == nil && r.options.RedirectFixedPath {
		fixedPath, found := r.findFixedPath(method, path, params)
		if found {
			query := req.URL.RawQuery
			if query != "" {
				fixedPath += "?" + query
			}

			w.Header().Set("Location", fixedPath)
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}
	}

	// Handle automatic OPTIONS responses after redirection and before 404/405 checks.
	// This allows OPTIONS requests to be handled even if no explicit OPTIONS handler is registered.
	if handler == nil && method == "OPTIONS" && r.options.HandleOPTIONS {
		allows := r.getAllowedMethods(path)
		if len(allows) > 0 {
			// Create a default handler so middleware can be applied (e.g., for CORS headers).
			handler = func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Allow", strings.Join(allows, ", "))
				// Use 204 No Content for preflight requests as it's common practice.
				w.WriteHeader(http.StatusNoContent)
			}
		}
	}

	if handler == nil {
		// Check if method is not allowed
		if r.options.HandleMethodNotAllowed && r.pathExistsForOtherMethods(path, method) {
			if r.methodNotAllowed != nil {
				r.methodNotAllowed.ServeHTTP(w, req)
			} else {
				w.Header().Set("Allow", strings.Join(r.getAllowedMethods(path), ", "))
				w.WriteHeader(http.StatusMethodNotAllowed)
				w.Write([]byte("Method Not Allowed"))
			}
			return
		}

		if r.notFoundHandler != nil {
			r.notFoundHandler.ServeHTTP(w, req)
		} else {
			http.NotFound(w, req)
		}
		return
	}

	// Set parameters in request context
	ctx := context.WithValue(req.Context(), paramsKey, params)
	req = req.WithContext(ctx)

	// Apply middleware and call handler
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handler(w, req)
	})

	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}

	h.ServeHTTP(w, req)
}

// Find a fixed path by trying case-insensitive and cleaned path variations
func (r *SteelRouter) findFixedPath(method, path string, params *Params) (string, bool) {
	cleanPath := cleanPath(path)
	if cleanPath != path {
		params.Reset()
		if handler := r.findHandler(method, cleanPath, params); handler != nil {
			return cleanPath, true
		}
	}

	// Try case-insensitive matching (if enabled)
	lowerPath := strings.ToLower(path)
	if lowerPath != path {
		params.Reset()
		if handler := r.findHandler(method, lowerPath, params); handler != nil {
			return lowerPath, true
		}
	}

	return "", false
}

// Check if path exists for other HTTP methods
func (r *SteelRouter) pathExistsForOtherMethods(path, currentMethod string) bool {
	for method := range r.trees {
		if method != currentMethod {
			params := r.pool.Get().(*Params)
			params.Reset()
			if handler := r.findHandler(method, path, params); handler != nil {
				r.pool.Put(params)
				return true
			}
			r.pool.Put(params)
		}
	}
	return false
}

// Get all allowed methods for a path
func (r *SteelRouter) getAllowedMethods(path string) []string {
	var methods []string
	for method := range r.trees {
		params := r.pool.Get().(*Params)
		params.Reset()
		if handler := r.findHandler(method, path, params); handler != nil {
			methods = append(methods, method)
		}
		r.pool.Put(params)
	}
	return methods
}

// Enhanced findHandler with better parameter extraction
func (r *SteelRouter) findHandler(method, path string, params *Params) HandlerFunc {
	root := r.trees[method]
	if root == nil {
		return nil
	}

	return root.findHandler(path, params)
}

// Enhanced addRoute with better path normalization
func (r *SteelRouter) addRoute(method, path string, handler HandlerFunc) {
	if path == "" {
		panic("path cannot be empty")
	}

	if path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}

	// Convert OpenAPI-style path parameters {id} to our format :id
	path = convertOpenAPIPath(path)

	if r.trees[method] == nil {
		r.trees[method] = &node{}
	}
	r.trees[method].addRoute(path, handler)
}

// SetTrailingSlashRedirect Add configuration methods
func (r *SteelRouter) SetTrailingSlashRedirect(enabled bool) {
	r.options.RedirectTrailingSlash = enabled
}

func (r *SteelRouter) SetFixedPathRedirect(enabled bool) {
	r.options.RedirectFixedPath = enabled
}

func (r *SteelRouter) SetMethodNotAllowedHandler(handler http.Handler) {
	r.methodNotAllowed = handler
}

func (r *SteelRouter) SetNotFoundHandler(handler http.Handler) {
	r.notFoundHandler = handler
}

func (r *SteelRouter) extractURLParams(path, method string, params *Params) {
	// Find the route pattern and extract parameters properly
	if root := r.trees[method]; root != nil {
		r.matchAndExtractParams(root, path, params)
	}
}

// Helper method to properly extract parameters from matched routes
func (r *SteelRouter) matchAndExtractParams(n *node, path string, params *Params) bool {
	if len(path) == 0 {
		return n.handler != nil
	}

	// Try exact matches first (static segments)
	for _, child := range n.children {
		if child.isParam || child.wildcard {
			continue
		}

		if strings.HasPrefix(path, child.path) {
			remainingPath := path[len(child.path):]
			if len(remainingPath) == 0 {
				return child.handler != nil
			}
			if len(remainingPath) > 0 && remainingPath[0] == '/' {
				return r.matchAndExtractParams(child, remainingPath, params)
			}
		}
	}

	// Try parameter matches
	for _, child := range n.children {
		if !child.isParam {
			continue
		}

		// Find the end of this parameter segment
		end := strings.IndexByte(path, '/')
		if end == -1 {
			end = len(path)
		}

		if end > 0 {
			// Store parameter value
			paramValue := path[:end]
			params.Set(child.paramName, paramValue)

			if end == len(path) {
				return child.handler != nil
			}

			// Continue with remaining path
			if r.matchAndExtractParams(child, path[end:], params) {
				return true
			}

			// Remove parameter if path doesn't match (backtrack)
			params.Remove(child.paramName)
		}
	}

	// Try wildcard matches
	for _, child := range n.children {
		if child.wildcard {
			return child.handler != nil
		}
	}

	return false
}
