package forge_router

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
)

// FastRouter is a high-performance HTTP router with OpenAPI support
type FastRouter struct {
	trees             map[string]*node
	middleware        []MiddlewareFunc
	pool              sync.Pool
	notFoundHandler   http.Handler
	methodNotAllowed  http.Handler
	options           RouterOptions
	openAPISpec       *OpenAPISpec
	asyncAPISpec      *AsyncAPISpec
	handlers          map[string]*HandlerInfo
	wsHandlers        map[string]*WSHandlerInfo
	sseHandlers       map[string]*SSEHandlerInfo
	connectionManager *ConnectionManager
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
type OpinionatedHandler[TInput any, TOutput any] func(ctx *FastContext, input TInput) (*TOutput, error)

// HandlerInfo stores metadata about registered handlers
type HandlerInfo struct {
	Method      string
	Path        string
	Summary     string
	Description string
	Tags        []string
	InputType   reflect.Type
	OutputType  reflect.Type
	Handler     interface{}
}

// OpenAPI Schema Types
type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       OpenAPIInfo            `json:"info"`
	Paths      map[string]OpenAPIPath `json:"paths"`
	Components OpenAPIComponents      `json:"components"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type OpenAPIPath map[string]OpenAPIOperation

type OpenAPIOperation struct {
	Summary     string                     `json:"summary,omitempty"`
	Description string                     `json:"description,omitempty"`
	Tags        []string                   `json:"tags,omitempty"`
	Parameters  []OpenAPIParameter         `json:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses"`
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
	Type                 string                   `json:"type,omitempty"`
	Format               string                   `json:"format,omitempty"`
	Description          string                   `json:"description,omitempty"`
	Properties           map[string]OpenAPISchema `json:"properties,omitempty"`
	Required             []string                 `json:"required,omitempty"`
	Items                *OpenAPISchema           `json:"items,omitempty"`
	Ref                  string                   `json:"$ref,omitempty"`
	AdditionalProperties *OpenAPISchema           `json:"additionalProperties,omitempty"`

	// Validation fields
	Example   interface{} `json:"example,omitempty"`
	Default   interface{} `json:"default,omitempty"`
	Minimum   *string     `json:"minimum,omitempty"`
	Maximum   *string     `json:"maximum,omitempty"`
	Pattern   string      `json:"pattern,omitempty"`
	MinLength *int        `json:"minLength,omitempty"`
	MaxLength *int        `json:"maxLength,omitempty"`
	MinItems  *int        `json:"minItems,omitempty"`
	MaxItems  *int        `json:"maxItems,omitempty"`

	// Enum support
	Enum []interface{} `json:"enum,omitempty"`

	// For numbers
	MultipleOf       *float64 `json:"multipleOf,omitempty"`
	ExclusiveMinimum *bool    `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum *bool    `json:"exclusiveMaximum,omitempty"`
}

type OpenAPIComponents struct {
	Schemas map[string]OpenAPISchema `json:"schemas"`
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

// NewRouter creates a new FastRouter instance
func NewRouter() *FastRouter {
	return &FastRouter{
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
			OpenAPITitle:           "FastRouter API",
			OpenAPIVersion:         "1.0.0",
			OpenAPIDescription:     "API documentation generated by FastRouter",
		},
		openAPISpec: &OpenAPISpec{
			OpenAPI: "3.0.0",
			Info: OpenAPIInfo{
				Title:       "FastRouter API",
				Version:     "1.0.0",
				Description: "API documentation generated by FastRouter",
			},
			Paths: make(map[string]OpenAPIPath),
			Components: OpenAPIComponents{
				Schemas: make(map[string]OpenAPISchema),
			},
		},
		handlers:          make(map[string]*HandlerInfo),
		wsHandlers:        make(map[string]*WSHandlerInfo),
		sseHandlers:       make(map[string]*SSEHandlerInfo),
		connectionManager: NewConnectionManager(),
	}
}

// Router interface for consistent API
type Router interface {
	Use(middleware ...MiddlewareFunc)
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

// Ensure FastRouter implements Router interface
var _ Router = (*FastRouter)(nil)

// Standard HTTP method handlers
func (r *FastRouter) GET(pattern string, handler HandlerFunc) {
	r.addRoute("GET", pattern, handler)
}

func (r *FastRouter) POST(pattern string, handler HandlerFunc) {
	r.addRoute("POST", pattern, handler)
}

func (r *FastRouter) PUT(pattern string, handler HandlerFunc) {
	r.addRoute("PUT", pattern, handler)
}

func (r *FastRouter) DELETE(pattern string, handler HandlerFunc) {
	r.addRoute("DELETE", pattern, handler)
}

func (r *FastRouter) PATCH(pattern string, handler HandlerFunc) {
	r.addRoute("PATCH", pattern, handler)
}

func (r *FastRouter) HEAD(pattern string, handler HandlerFunc) {
	r.addRoute("HEAD", pattern, handler)
}

func (r *FastRouter) OPTIONS(pattern string, handler HandlerFunc) {
	r.addRoute("OPTIONS", pattern, handler)
}

func (r *FastRouter) Handle(method, pattern string, handler HandlerFunc) {
	r.addRoute(method, pattern, handler)
}

func (r *FastRouter) HandleFunc(method, pattern string, handler http.HandlerFunc) {
	r.addRoute(method, pattern, func(w http.ResponseWriter, req *http.Request) {
		handler(w, req)
	})
}

// Opinionated handlers with OpenAPI generation
func (r *FastRouter) OpinionatedGET(pattern string, handler interface{}, opts ...HandlerOption) {
	r.registerOpinionatedHandler("GET", pattern, handler, opts...)
}

func (r *FastRouter) OpinionatedPOST(pattern string, handler interface{}, opts ...HandlerOption) {
	r.registerOpinionatedHandler("POST", pattern, handler, opts...)
}

func (r *FastRouter) OpinionatedPUT(pattern string, handler interface{}, opts ...HandlerOption) {
	r.registerOpinionatedHandler("PUT", pattern, handler, opts...)
}

func (r *FastRouter) OpinionatedDELETE(pattern string, handler interface{}, opts ...HandlerOption) {
	r.registerOpinionatedHandler("DELETE", pattern, handler, opts...)
}

func (r *FastRouter) OpinionatedPATCH(pattern string, handler interface{}, opts ...HandlerOption) {
	r.registerOpinionatedHandler("PATCH", pattern, handler, opts...)
}

// Register opinionated handler with reflection and OpenAPI generation
func (r *FastRouter) registerOpinionatedHandler(method, pattern string, handler interface{}, opts ...HandlerOption) {
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Func {
		panic("handler must be a function")
	}

	if handlerType.NumIn() != 2 || handlerType.NumOut() != 2 {
		panic("handler must have signature func(*FastContext, InputType) (*OutputType, error)")
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
func (r *FastRouter) createOpinionatedWrapper(handler interface{}, inputType, outputType reflect.Type) HandlerFunc {
	handlerValue := reflect.ValueOf(handler)

	return func(w http.ResponseWriter, req *http.Request) {
		// Get parameters from context, which were populated by ServeHTTP's findHandler
		params := ParamsFromContext(req.Context())

		// Create FastContext with enhanced error handling
		ctx := &FastContext{
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
func (r *FastRouter) handleError(w http.ResponseWriter, req *http.Request, err error) {
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
func (r *FastRouter) writeErrorResponse(w http.ResponseWriter, req *http.Request, apiErr APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.StatusCode())

	response := apiErr.ToResponse()
	json.NewEncoder(w).Encode(response)
}

// Bind parameters from request to struct based on tags
func (r *FastRouter) bindParameters(ctx *FastContext, input interface{}) error {
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
func (r *FastRouter) setFieldValue(field reflect.Value, value string) error {
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
func (r *FastRouter) generateOpenAPIForHandler(info *HandlerInfo) {
	operation := OpenAPIOperation{
		Summary:     info.Summary,
		Description: info.Description,
		Tags:        info.Tags,
		Parameters:  []OpenAPIParameter{},
		Responses:   make(map[string]OpenAPIResponse),
	}

	// Check if this handler has body parameters
	hasBodyParams := false
	var bodySchema OpenAPISchema

	// Generate parameters from input struct
	if info.InputType.Kind() == reflect.Struct {
		// First pass: check for body parameters
		for i := 0; i < info.InputType.NumField(); i++ {
			field := info.InputType.Field(i)
			if bodyTag := field.Tag.Get("body"); bodyTag != "" {
				hasBodyParams = true
				// Create a schema for the entire struct when body params are present
				bodySchema = r.typeToSchema(info.InputType)
				break
			}
		}

		// Second pass: handle other parameter types
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

	// Add standard error responses
	r.addStandardErrorResponses(&operation, info.Method)

	// Add to OpenAPI spec - ensure proper path format
	openAPIPath := r.convertToOpenAPIPath(info.Path)
	if r.openAPISpec.Paths[openAPIPath] == nil {
		r.openAPISpec.Paths[openAPIPath] = make(OpenAPIPath)
	}
	r.openAPISpec.Paths[openAPIPath][strings.ToLower(info.Method)] = operation
}

// convertToOpenAPIPath Helper function to convert internal path format to OpenAPI format
func (r *FastRouter) convertToOpenAPIPath(path string) string {
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
func (r *FastRouter) addStandardErrorResponses(operation *OpenAPIOperation, method string) {
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
func (r *FastRouter) ensureErrorSchemasRegistered() {
	// ErrorResponse schema
	if _, exists := r.openAPISpec.Components.Schemas["ErrorResponse"]; !exists {
		r.openAPISpec.Components.Schemas["ErrorResponse"] = OpenAPISchema{
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"error": {Ref: "#/components/schemas/ErrorDetail"},
			},
			Required: []string{"error"},
		}
	}

	// ErrorDetail schema
	if _, exists := r.openAPISpec.Components.Schemas["ErrorDetail"]; !exists {
		r.openAPISpec.Components.Schemas["ErrorDetail"] = OpenAPISchema{
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"status": {
					Type:        "integer",
					Description: "HTTP status code",
					Example:     400,
				},
				"code": {
					Type:        "string",
					Description: "Error code for programmatic handling",
					Example:     "BAD_REQUEST",
				},
				"message": {
					Type:        "string",
					Description: "Human-readable error message",
					Example:     "Invalid input provided",
				},
				"detail": {
					Description: "Additional error details",
				},
				"timestamp": {
					Type:        "string",
					Format:      "date-time",
					Description: "Error timestamp",
				},
				"request_id": {
					Type:        "string",
					Description: "Request tracking ID",
				},
				"path": {
					Type:        "string",
					Description: "Request path that caused the error",
				},
			},
			Required: []string{"status", "code", "message", "timestamp"},
		}
	}

	// ValidationErrorResponse schema
	if _, exists := r.openAPISpec.Components.Schemas["ValidationErrorResponse"]; !exists {
		r.openAPISpec.Components.Schemas["ValidationErrorResponse"] = OpenAPISchema{
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"error": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"status": {
							Type:        "integer",
							Description: "HTTP status code",
							Example:     422,
						},
						"code": {
							Type:        "string",
							Description: "Error code",
							Example:     "VALIDATION_FAILED",
						},
						"message": {
							Type:        "string",
							Description: "Error message",
							Example:     "Validation failed",
						},
						"detail": {
							Type: "array",
							Items: &OpenAPISchema{
								Ref: "#/components/schemas/FieldError",
							},
							Description: "Field-specific validation errors",
						},
						"timestamp": {
							Type:   "string",
							Format: "date-time",
						},
						"request_id": {Type: "string"},
						"path":       {Type: "string"},
					},
					Required: []string{"status", "code", "message", "timestamp"},
				},
			},
			Required: []string{"error"},
		}
	}

	// FieldError schema
	if _, exists := r.openAPISpec.Components.Schemas["FieldError"]; !exists {
		r.openAPISpec.Components.Schemas["FieldError"] = OpenAPISchema{
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"field": {
					Type:        "string",
					Description: "Field name that failed validation",
					Example:     "email",
				},
				"message": {
					Type:        "string",
					Description: "Validation error message",
					Example:     "Invalid email format",
				},
				"value": {
					Description: "Value that failed validation",
					Example:     "invalid-email",
				},
				"code": {
					Type:        "string",
					Description: "Validation error code",
					Example:     "INVALID_FORMAT",
				},
			},
			Required: []string{"field", "message"},
		}
	}
}

// Convert Go type to OpenAPI schema
// Convert Go type to OpenAPI schema (ENHANCED VERSION with full Go type support)
func (r *FastRouter) typeToSchema(t reflect.Type) OpenAPISchema {
	// Handle pointer types by dereferencing
	if t.Kind() == reflect.Ptr {
		return r.typeToSchema(t.Elem())
	}

	// Handle special named types first
	if t.PkgPath() != "" && t.Name() != "" {
		// Handle time.Time and time.Duration specifically
		switch t.String() {
		case "time.Time":
			return OpenAPISchema{
				Type:        "string",
				Format:      "date-time",
				Description: "RFC3339 date-time format",
			}
		case "time.Duration":
			return OpenAPISchema{
				Type:        "string",
				Description: "Duration in Go format (e.g., '1h30m', '5s')",
			}
		}

		// Handle other standard library types
		switch t.PkgPath() {
		case "net/url":
			if t.Name() == "URL" {
				return OpenAPISchema{
					Type:   "string",
					Format: "uri",
				}
			}
		case "net":
			if t.Name() == "IP" {
				return OpenAPISchema{
					Type:   "string",
					Format: "ipv4", // or ipv6, but ipv4 is more common default
				}
			}
		case "encoding/json":
			if t.Name() == "RawMessage" {
				return OpenAPISchema{
					Type:        "object",
					Description: "Raw JSON data",
				}
			}
		}
	}

	switch t.Kind() {
	case reflect.String:
		return OpenAPISchema{Type: "string"}

	// Integer types
	case reflect.Int:
		return OpenAPISchema{Type: "integer", Format: "int64"} // Platform dependent, but int64 is safe
	case reflect.Int8:
		return OpenAPISchema{Type: "integer", Format: "int32", Description: "8-bit signed integer"}
	case reflect.Int16:
		return OpenAPISchema{Type: "integer", Format: "int32", Description: "16-bit signed integer"}
	case reflect.Int32:
		return OpenAPISchema{Type: "integer", Format: "int32"}
	case reflect.Int64:
		return OpenAPISchema{Type: "integer", Format: "int64"}

	// Unsigned integer types
	case reflect.Uint:
		return OpenAPISchema{Type: "integer", Format: "int64", Description: "Unsigned integer"}
	case reflect.Uint8:
		// Special case: uint8 is often used for bytes, but in API context usually integer
		return OpenAPISchema{Type: "integer", Format: "int32", Description: "8-bit unsigned integer (byte)"}
	case reflect.Uint16:
		return OpenAPISchema{Type: "integer", Format: "int32", Description: "16-bit unsigned integer"}
	case reflect.Uint32:
		return OpenAPISchema{Type: "integer", Format: "int64", Description: "32-bit unsigned integer"}
	case reflect.Uint64:
		return OpenAPISchema{Type: "integer", Format: "int64", Description: "64-bit unsigned integer"}
	case reflect.Uintptr:
		return OpenAPISchema{Type: "string", Description: "Pointer address as string"}

	// Floating point types
	case reflect.Float32:
		return OpenAPISchema{Type: "number", Format: "float"}
	case reflect.Float64:
		return OpenAPISchema{Type: "number", Format: "double"}

	// Complex types
	case reflect.Complex64:
		return OpenAPISchema{
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"real": {Type: "number", Format: "float"},
				"imag": {Type: "number", Format: "float"},
			},
			Required:    []string{"real", "imag"},
			Description: "Complex number with real and imaginary parts",
		}
	case reflect.Complex128:
		return OpenAPISchema{
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"real": {Type: "number", Format: "double"},
				"imag": {Type: "number", Format: "double"},
			},
			Required:    []string{"real", "imag"},
			Description: "Complex number with real and imaginary parts",
		}

	case reflect.Bool:
		return OpenAPISchema{Type: "boolean"}

	// Collection types
	case reflect.Slice, reflect.Array:
		elemSchema := r.typeToSchema(t.Elem())
		schema := OpenAPISchema{
			Type:  "array",
			Items: &elemSchema,
		}

		// Add description for byte slices (common for binary data)
		if t.Elem().Kind() == reflect.Uint8 {
			schema.Description = "Base64 encoded binary data"
			schema.Format = "byte"
		}

		return schema

	case reflect.Map:
		// Handle map types
		keyType := t.Key()
		valueSchema := r.typeToSchema(t.Elem())

		// OpenAPI only supports string keys in objects
		if keyType.Kind() == reflect.String {
			return OpenAPISchema{
				Type:                 "object",
				AdditionalProperties: &valueSchema,
				Description:          "Map with string keys",
			}
		} else {
			// For non-string keys, represent as array of key-value pairs
			return OpenAPISchema{
				Type: "array",
				Items: &OpenAPISchema{
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
		// Handle interface{} and any types
		if t.NumMethod() == 0 { // empty interface
			return OpenAPISchema{
				Description: "Any value (interface{})",
				// No type specified - allows any type
			}
		}
		// For non-empty interfaces, we can't really represent them well in OpenAPI
		return OpenAPISchema{
			Type:        "object",
			Description: "Interface type - actual structure may vary",
		}

	case reflect.Chan:
		// Channels don't make sense in HTTP APIs, but handle gracefully
		return OpenAPISchema{
			Type:        "string",
			Description: "Channel type (not serializable in HTTP APIs)",
		}

	case reflect.Func:
		// Functions don't make sense in HTTP APIs, but handle gracefully
		return OpenAPISchema{
			Type:        "string",
			Description: "Function type (not serializable in HTTP APIs)",
		}

	case reflect.UnsafePointer:
		return OpenAPISchema{
			Type:        "string",
			Description: "Unsafe pointer (not serializable in HTTP APIs)",
		}

	default:
		return OpenAPISchema{
			Type:        "string",
			Description: "Unknown type, represented as string",
		}
	}
}

// Enhanced generateStructSchema with better field handling
func (r *FastRouter) generateStructSchema(t reflect.Type) OpenAPISchema {
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
			// For embedded structs, merge their properties
			if field.Type.Kind() == reflect.Struct || (field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct) {
				embeddedSchema := r.typeToSchema(field.Type)
				if embeddedSchema.Properties != nil {
					for propName, propSchema := range embeddedSchema.Properties {
						schema.Properties[propName] = propSchema
					}
				}
				if embeddedSchema.Required != nil {
					schema.Required = append(schema.Required, embeddedSchema.Required...)
				}
			}
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse json tag to get field name and options
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

		// Add example from tag if present
		if example := field.Tag.Get("example"); example != "" {
			fieldSchema.Example = example
		}

		// Add format from tag if present (overrides auto-detected format)
		if format := field.Tag.Get("format"); format != "" {
			fieldSchema.Format = format
		}

		// Add validation constraints
		if min := field.Tag.Get("min"); min != "" {
			fieldSchema.Minimum = &min
		}
		if max := field.Tag.Get("max"); max != "" {
			fieldSchema.Maximum = &max
		}
		if pattern := field.Tag.Get("pattern"); pattern != "" {
			fieldSchema.Pattern = pattern
		}

		schema.Properties[fieldName] = fieldSchema

		// Check if field is required (required tag or not omitempty and not pointer)
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
func (r *FastRouter) EnableOpenAPI() {
	// Serve OpenAPI spec
	r.GET("/openapi", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(r.openAPISpec)
	})

	// Swagger UI
	r.GET("/openapi/swagger", func(w http.ResponseWriter, req *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation - Swagger UI</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.25.0/swagger-ui.css" />
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@3.25.0/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: '/openapi',
            dom_id: '#swagger-ui',
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIBundle.presets.standalone
            ]
        });
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

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
        apiDescriptionUrl="/openapi"
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
        data-url="/openapi"
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
    <redoc spec-url="/openapi"></redoc>
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
        
        <a href="/openapi" class="spec-link">View Raw OpenAPI Spec</a>
    </div>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})
}

// Use Standard router implementation (keeping existing functionality)
func (r *FastRouter) Use(middleware ...MiddlewareFunc) {
	r.middleware = append(r.middleware, middleware...)
}

// Group creates a new route group with empty prefix
func (r *FastRouter) Group() Router {
	return &RouteGroup{
		router:     r,
		prefix:     "",
		middleware: make([]MiddlewareFunc, 0),
	}
}

// GroupFunc creates a route group and calls the provided function with it (renamed from Group)
func (r *FastRouter) GroupFunc(fn func(r Router)) Router {
	group := &RouteGroup{
		router:     r,
		prefix:     "",
		middleware: make([]MiddlewareFunc, 0),
	}
	fn(group)
	return group
}

func (r *FastRouter) Route(pattern string, fn func(r Router)) Router {
	group := &RouteGroup{
		router: r,
		prefix: pattern,
	}
	fn(group)
	return group
}

func (r *FastRouter) Mount(pattern string, handler http.Handler) {
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

func (r *FastRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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
func (r *FastRouter) findFixedPath(method, path string, params *Params) (string, bool) {
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
func (r *FastRouter) pathExistsForOtherMethods(path, currentMethod string) bool {
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
func (r *FastRouter) getAllowedMethods(path string) []string {
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
func (r *FastRouter) findHandler(method, path string, params *Params) HandlerFunc {
	root := r.trees[method]
	if root == nil {
		return nil
	}

	return root.findHandler(path, params)
}

// Enhanced addRoute with better path normalization
func (r *FastRouter) addRoute(method, path string, handler HandlerFunc) {
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
func (r *FastRouter) SetTrailingSlashRedirect(enabled bool) {
	r.options.RedirectTrailingSlash = enabled
}

func (r *FastRouter) SetFixedPathRedirect(enabled bool) {
	r.options.RedirectFixedPath = enabled
}

func (r *FastRouter) SetMethodNotAllowedHandler(handler http.Handler) {
	r.methodNotAllowed = handler
}

func (r *FastRouter) SetNotFoundHandler(handler http.Handler) {
	r.notFoundHandler = handler
}

func (r *FastRouter) extractURLParams(path, method string, params *Params) {
	// Find the route pattern and extract parameters properly
	if root := r.trees[method]; root != nil {
		r.matchAndExtractParams(root, path, params)
	}
}

// Helper method to properly extract parameters from matched routes
func (r *FastRouter) matchAndExtractParams(n *node, path string, params *Params) bool {
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
