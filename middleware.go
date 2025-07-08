package forgerouter

import (
	"fmt"
	"reflect"
	"time"
)

// =============================================================================
// Opinionated Middleware Specification
// =============================================================================

// OpinionatedMiddleware represents middleware that can work with opinionated handlers
// and contribute to OpenAPI documentation
type OpinionatedMiddleware interface {
	// Process handles the middleware logic with full context
	Process(ctx *MiddlewareContext, next OpinionatedNext) error

	// GetMetadata returns metadata about what this middleware does
	GetMetadata() MiddlewareMetadata
}

// MiddlewareContext provides rich context for opinionated middleware
type MiddlewareContext struct {
	*FastContext

	// Request metadata
	StartTime time.Time
	RequestID string
	UserID    string
	Metadata  map[string]interface{}

	// Handler information
	HandlerInfo *HandlerInfo
	InputValue  reflect.Value
	OutputValue reflect.Value

	// Processing state
	Processed  bool
	Error      error
	StatusCode int
	Headers    map[string]string

	// Middleware chain state
	MiddlewareIndex int
	Middlewares     []OpinionatedMiddleware
}

// OpinionatedNext represents the next step in the middleware chain
type OpinionatedNext func() error

// MiddlewareMetadata provides information about middleware capabilities
type MiddlewareMetadata struct {
	Name        string
	Description string
	Version     string

	// OpenAPI contributions
	SecurityRequirements []OpenAPISecurityRequirement
	Headers              []OpenAPIParameter
	Responses            map[string]OpenAPIResponse
	Tags                 []string

	// Middleware behavior
	ModifiesRequest  bool
	ModifiesResponse bool
	RequiresAuth     bool
	CachingSafe      bool

	// Dependencies
	Dependencies  []string
	ConflictsWith []string
}

// =============================================================================
// Middleware Builder for Easy Creation
// =============================================================================

// MiddlewareBuilder provides a fluent interface for creating opinionated middleware
type MiddlewareBuilder struct {
	name        string
	description string
	beforeFn    BeforeFunc
	afterFn     AfterFunc
	errorFn     ErrorFunc
	metadata    MiddlewareMetadata
}

// BeforeFunc is called before the handler
type BeforeFunc func(ctx *MiddlewareContext) error

// AfterFunc is called after the handler
type AfterFunc func(ctx *MiddlewareContext) error

// ErrorFunc is called when an error occurs
type ErrorFunc func(ctx *MiddlewareContext, err error) error

// NewMiddleware creates a new middleware builder
func NewMiddleware(name string) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		name: name,
		metadata: MiddlewareMetadata{
			Name:          name,
			Headers:       make([]OpenAPIParameter, 0),
			Responses:     make(map[string]OpenAPIResponse),
			Tags:          make([]string, 0),
			Dependencies:  make([]string, 0),
			ConflictsWith: make([]string, 0),
		},
	}
}

// Description sets the middleware description
func (b *MiddlewareBuilder) Description(desc string) *MiddlewareBuilder {
	b.description = desc
	b.metadata.Description = desc
	return b
}

// Before sets the before handler
func (b *MiddlewareBuilder) Before(fn BeforeFunc) *MiddlewareBuilder {
	b.beforeFn = fn
	return b
}

// After sets the after handler
func (b *MiddlewareBuilder) After(fn AfterFunc) *MiddlewareBuilder {
	b.afterFn = fn
	return b
}

// OnError sets the error handler
func (b *MiddlewareBuilder) OnError(fn ErrorFunc) *MiddlewareBuilder {
	b.errorFn = fn
	return b
}

// AddSecurityRequirement adds security requirements
func (b *MiddlewareBuilder) AddSecurityRequirement(req OpenAPISecurityRequirement) *MiddlewareBuilder {
	b.metadata.SecurityRequirements = append(b.metadata.SecurityRequirements, req)
	return b
}

// AddHeader adds a header parameter to OpenAPI spec
func (b *MiddlewareBuilder) AddHeader(name, description string, required bool) *MiddlewareBuilder {
	header := OpenAPIParameter{
		Name:        name,
		In:          "header",
		Required:    required,
		Description: description,
		Schema:      OpenAPISchema{Type: "string"},
	}
	b.metadata.Headers = append(b.metadata.Headers, header)
	return b
}

// AddResponse adds a response to OpenAPI spec
func (b *MiddlewareBuilder) AddResponse(statusCode, description string) *MiddlewareBuilder {
	b.metadata.Responses[statusCode] = OpenAPIResponse{
		Description: description,
		Content: map[string]OpenAPIMediaType{
			"application/json": {
				Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"},
			},
		},
	}
	return b
}

// RequiresAuth marks this middleware as requiring authentication
func (b *MiddlewareBuilder) RequiresAuth() *MiddlewareBuilder {
	b.metadata.RequiresAuth = true
	return b
}

// ModifiesRequest marks this middleware as modifying the request
func (b *MiddlewareBuilder) ModifiesRequest() *MiddlewareBuilder {
	b.metadata.ModifiesRequest = true
	return b
}

// ModifiesResponse marks this middleware as modifying the response
func (b *MiddlewareBuilder) ModifiesResponse() *MiddlewareBuilder {
	b.metadata.ModifiesResponse = true
	return b
}

// CachingSafe marks this middleware as safe for caching
func (b *MiddlewareBuilder) CachingSafe() *MiddlewareBuilder {
	b.metadata.CachingSafe = true
	return b
}

// DependsOn adds dependencies
func (b *MiddlewareBuilder) DependsOn(dependencies ...string) *MiddlewareBuilder {
	b.metadata.Dependencies = append(b.metadata.Dependencies, dependencies...)
	return b
}

// ConflictsWith adds conflicts
func (b *MiddlewareBuilder) ConflictsWith(conflicts ...string) *MiddlewareBuilder {
	b.metadata.ConflictsWith = append(b.metadata.ConflictsWith, conflicts...)
	return b
}

// AddTags adds tags
func (b *MiddlewareBuilder) AddTags(tags ...string) *MiddlewareBuilder {
	b.metadata.Tags = append(b.metadata.Tags, tags...)
	return b
}

// Build creates the final middleware
func (b *MiddlewareBuilder) Build() OpinionatedMiddleware {
	return &builtMiddleware{
		name:        b.name,
		description: b.description,
		beforeFn:    b.beforeFn,
		afterFn:     b.afterFn,
		errorFn:     b.errorFn,
		metadata:    b.metadata,
	}
}

// =============================================================================
// Built Middleware Implementation
// =============================================================================

type builtMiddleware struct {
	name        string
	description string
	beforeFn    BeforeFunc
	afterFn     AfterFunc
	errorFn     ErrorFunc
	metadata    MiddlewareMetadata
}

func (m *builtMiddleware) Process(ctx *MiddlewareContext, next OpinionatedNext) error {
	// Execute before function
	if m.beforeFn != nil {
		if err := m.beforeFn(ctx); err != nil {
			if m.errorFn != nil {
				return m.errorFn(ctx, err)
			}
			return err
		}
	}

	// Execute next middleware/handler
	err := next()

	// Execute after function
	if m.afterFn != nil {
		if afterErr := m.afterFn(ctx); afterErr != nil {
			if m.errorFn != nil {
				return m.errorFn(ctx, afterErr)
			}
			return afterErr
		}
	}

	// Handle errors from next
	if err != nil && m.errorFn != nil {
		return m.errorFn(ctx, err)
	}

	return err
}

func (m *builtMiddleware) GetMetadata() MiddlewareMetadata {
	return m.metadata
}

// =============================================================================
// Typed Middleware for Specific Input/Output Types
// =============================================================================

// TypedMiddleware provides type-safe middleware for specific input/output types
type TypedMiddleware[TInput any, TOutput any] interface {
	Process(ctx *FastContext, input *TInput, output *TOutput, next TypedNext[TInput, TOutput]) error
	GetMetadata() MiddlewareMetadata
}

// TypedNext represents the next handler for typed middleware
type TypedNext[TInput any, TOutput any] func(ctx *FastContext, input *TInput) (*TOutput, error)

// TypedMiddlewareAdapter adapts typed middleware to opinionated middleware
func TypedMiddlewareAdapter[TInput any, TOutput any](typed TypedMiddleware[TInput, TOutput]) OpinionatedMiddleware {
	return &typedMiddlewareAdapter[TInput, TOutput]{typed: typed}
}

type typedMiddlewareAdapter[TInput any, TOutput any] struct {
	typed TypedMiddleware[TInput, TOutput]
}

func (a *typedMiddlewareAdapter[TInput, TOutput]) Process(ctx *MiddlewareContext, next OpinionatedNext) error {
	// Extract typed input/output from context
	var input *TInput
	var output *TOutput

	if ctx.InputValue.IsValid() && ctx.InputValue.Type() == reflect.TypeOf((*TInput)(nil)).Elem() {
		input = ctx.InputValue.Addr().Interface().(*TInput)
	}

	if ctx.OutputValue.IsValid() && ctx.OutputValue.Type() == reflect.TypeOf((*TOutput)(nil)).Elem() {
		output = ctx.OutputValue.Addr().Interface().(*TOutput)
	}

	typedNext := func(ctx *FastContext, input *TInput) (*TOutput, error) {
		// Call the original next function
		err := next()
		if err != nil {
			return nil, err
		}
		return output, nil
	}

	return a.typed.Process(ctx.FastContext, input, output, typedNext)
}

func (a *typedMiddlewareAdapter[TInput, TOutput]) GetMetadata() MiddlewareMetadata {
	return a.typed.GetMetadata()
}

// =============================================================================
// Middleware Chain Management
// =============================================================================

// MiddlewareChain manages a chain of opinionated middleware
type MiddlewareChain struct {
	middlewares []OpinionatedMiddleware
	router      *FastRouter
}

// NewMiddlewareChain creates a new middleware chain
func NewMiddlewareChain(router *FastRouter) *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]OpinionatedMiddleware, 0),
		router:      router,
	}
}

// Use adds middleware to the chain
func (c *MiddlewareChain) Use(middleware ...OpinionatedMiddleware) *MiddlewareChain {
	c.middlewares = append(c.middlewares, middleware...)
	return c
}

// UseIf conditionally adds middleware to the chain
func (c *MiddlewareChain) UseIf(condition bool, middleware ...OpinionatedMiddleware) *MiddlewareChain {
	if condition {
		c.middlewares = append(c.middlewares, middleware...)
	}
	return c
}

// Process executes the middleware chain
func (c *MiddlewareChain) Process(ctx *MiddlewareContext, handler OpinionatedNext) error {
	ctx.MiddlewareIndex = 0
	ctx.Middlewares = c.middlewares

	var next OpinionatedNext
	next = func() error {
		if ctx.MiddlewareIndex >= len(c.middlewares) {
			return handler()
		}

		middleware := c.middlewares[ctx.MiddlewareIndex]
		ctx.MiddlewareIndex++

		return middleware.Process(ctx, next)
	}

	return next()
}

// ValidateChain validates the middleware chain for conflicts and dependencies
func (c *MiddlewareChain) ValidateChain() error {
	middlewareNames := make(map[string]bool)

	// Collect all middleware names
	for _, middleware := range c.middlewares {
		metadata := middleware.GetMetadata()
		middlewareNames[metadata.Name] = true
	}

	// Check dependencies and conflicts
	for _, middleware := range c.middlewares {
		metadata := middleware.GetMetadata()

		// Check dependencies
		for _, dep := range metadata.Dependencies {
			if !middlewareNames[dep] {
				return fmt.Errorf("middleware '%s' depends on '%s' which is not present", metadata.Name, dep)
			}
		}

		// Check conflicts
		for _, conflict := range metadata.ConflictsWith {
			if middlewareNames[conflict] {
				return fmt.Errorf("middleware '%s' conflicts with '%s'", metadata.Name, conflict)
			}
		}
	}

	return nil
}

// GetOpenAPIEnhancements collects OpenAPI enhancements from all middleware
func (c *MiddlewareChain) GetOpenAPIEnhancements() OpenAPIEnhancements {
	enhancements := OpenAPIEnhancements{
		SecurityRequirements: make([]OpenAPISecurityRequirement, 0),
		Headers:              make([]OpenAPIParameter, 0),
		Responses:            make(map[string]OpenAPIResponse),
		Tags:                 make([]string, 0),
	}

	for _, middleware := range c.middlewares {
		metadata := middleware.GetMetadata()

		enhancements.SecurityRequirements = append(enhancements.SecurityRequirements, metadata.SecurityRequirements...)
		enhancements.Headers = append(enhancements.Headers, metadata.Headers...)
		enhancements.Tags = append(enhancements.Tags, metadata.Tags...)

		for code, response := range metadata.Responses {
			enhancements.Responses[code] = response
		}
	}

	return enhancements
}

// OpenAPIEnhancements represents OpenAPI spec enhancements from middleware
type OpenAPIEnhancements struct {
	SecurityRequirements []OpenAPISecurityRequirement
	Headers              []OpenAPIParameter
	Responses            map[string]OpenAPIResponse
	Tags                 []string
}

// =============================================================================
// Convenience Methods for Common Middleware Combinations
// =============================================================================

// // WithBasicSecurity adds basic security middleware (CORS, Security Headers, Request ID)
// func (r *FastRouter) WithBasicSecurity(corsConfig ...CORSConfig) *FastRouter {
// 	r.UseOpinionated(
// 		OpinionatedRequestID(),
// 		OpinionatedCORS(corsConfig...),
// 	)
//
// 	// Add regular middleware for security headers (no opinionated version needed)
// 	r.Use(SecureHeaders())
//
// 	return r
// }
//
// // WithAuthentication adds JWT authentication with optional rate limiting
// func (r *FastRouter) WithAuthentication(jwtConfig JWTConfig, rateLimitConfig ...RateLimitConfig) *FastRouter {
// 	r.UseOpinionated(OpinionatedJWT(jwtConfig))
//
// 	if len(rateLimitConfig) > 0 {
// 		r.UseOpinionated(OpinionatedRateLimit(rateLimitConfig[0]))
// 	}
//
// 	return r
// }
//
// // WithFullStack adds a complete middleware stack for production
// func (r *FastRouter) WithFullStack(jwtConfig JWTConfig) *FastRouter {
// 	// Add regular middleware first
// 	r.Use(RequestID())
// 	r.Use(Logger)
// 	r.Use(Recoverer)
// 	r.Use(SecureHeaders())
// 	r.Use(Compression())
// 	r.Use(BodyLimit(10 << 20)) // 10MB limit
//
// 	// Add opinionated middleware
// 	r.UseOpinionated(
// 		OpinionatedCORS(),
// 		OpinionatedRateLimit(RateLimitConfig{
// 			RequestsPerSecond: 100,
// 			BurstSize:         200,
// 		}),
// 		OpinionatedJWT(jwtConfig),
// 	)
//
// 	return r
// }
//
// // WithDevelopment adds middleware suitable for development
// func (r *FastRouter) WithDevelopment() *FastRouter {
// 	r.Use(RequestID())
// 	r.Use(Logger)
// 	r.Use(Recoverer)
//
// 	r.UseOpinionated(
// 		OpinionatedCORS(CORSConfig{
// 			AllowedOrigins:   []string{"*"},
// 			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
// 			AllowedHeaders:   []string{"*"},
// 			AllowCredentials: true,
// 		}),
// 	)
//
// 	return r
// }

// =============================================================================
// Router Integration
// =============================================================================

// UseOpinionated adds opinionated middleware to the router
func (r *FastRouter) UseOpinionated(middleware ...OpinionatedMiddleware) {
	if r.opinionatedMiddleware == nil {
		r.opinionatedMiddleware = NewMiddlewareChain(r)
	}
	r.opinionatedMiddleware.Use(middleware...)
}

// UseOpinionatedIf conditionally adds opinionated middleware
func (r *FastRouter) UseOpinionatedIf(condition bool, middleware ...OpinionatedMiddleware) {
	if condition {
		r.UseOpinionated(middleware...)
	}
}

// GetOpinionatedMiddleware returns the opinionated middleware chain
func (r *FastRouter) GetOpinionatedMiddleware() *MiddlewareChain {
	if r.opinionatedMiddleware == nil {
		r.opinionatedMiddleware = NewMiddlewareChain(r)
	}
	return r.opinionatedMiddleware
}

// =============================================================================
// Middleware Validation and Info
// =============================================================================

// ValidateMiddleware validates the middleware configuration
func (r *FastRouter) ValidateMiddleware() error {
	if r.opinionatedMiddleware == nil {
		return nil // No middleware to validate
	}

	return r.opinionatedMiddleware.ValidateChain()
}

// GetMiddlewareInfo returns information about registered middleware
func (r *FastRouter) GetMiddlewareInfo() []MiddlewareMetadata {
	if r.opinionatedMiddleware == nil {
		return []MiddlewareMetadata{}
	}

	var info []MiddlewareMetadata
	for _, middleware := range r.opinionatedMiddleware.middlewares {
		info = append(info, middleware.GetMetadata())
	}

	return info
}

// PrintMiddlewareInfo prints middleware information for debugging
func (r *FastRouter) PrintMiddlewareInfo() {
	info := r.GetMiddlewareInfo()
	if len(info) == 0 {
		fmt.Println("No opinionated middleware registered")
		return
	}

	fmt.Println("=== Opinionated Middleware Info ===")
	for i, middleware := range info {
		fmt.Printf("%d. %s\n", i+1, middleware.Name)
		fmt.Printf("   Description: %s\n", middleware.Description)
		fmt.Printf("   Modifies Request: %v\n", middleware.ModifiesRequest)
		fmt.Printf("   Modifies Response: %v\n", middleware.ModifiesResponse)
		fmt.Printf("   Requires Auth: %v\n", middleware.RequiresAuth)
		fmt.Printf("   Caching Safe: %v\n", middleware.CachingSafe)

		if len(middleware.Dependencies) > 0 {
			fmt.Printf("   Dependencies: %v\n", middleware.Dependencies)
		}
		if len(middleware.ConflictsWith) > 0 {
			fmt.Printf("   Conflicts: %v\n", middleware.ConflictsWith)
		}

		fmt.Println()
	}
}
