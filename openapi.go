package steel

import (
	"fmt"
	"reflect"
	"strings"
)

// OpenAPI Security Scheme Types
const (
	SecurityTypeAPIKey        = "apiKey"
	SecurityTypeHTTP          = "http"
	SecurityTypeOAuth2        = "oauth2"
	SecurityTypeOpenIDConnect = "openIdConnect"
)

// API Key locations
const (
	APIKeyInQueryType  = "query"
	APIKeyInHeaderType = "header"
	APIKeyInCookieType = "cookie"
)

// HTTP authentication schemes
const (
	HTTPSchemeBasic  = "basic"
	HTTPSchemeBearer = "bearer"
	HTTPSchemeDigest = "digest"
)

// OAuth2 Flow Types
const (
	OAuth2FlowImplicit          = "implicit"
	OAuth2FlowPassword          = "password"
	OAuth2FlowClientCredentials = "clientCredentials"
	OAuth2FlowAuthorizationCode = "authorizationCode"
)

// OpenAPISecurityScheme OpenAPI Components with Security Schemes
type OpenAPISecurityScheme struct {
	Type             string                 `json:"type"`
	Description      string                 `json:"description,omitempty"`
	Name             string                 `json:"name,omitempty"`             // For apiKey
	In               string                 `json:"in,omitempty"`               // For apiKey
	Scheme           string                 `json:"scheme,omitempty"`           // For http
	BearerFormat     string                 `json:"bearerFormat,omitempty"`     // For http bearer
	Flows            *OpenAPIOAuth2Flows    `json:"flows,omitempty"`            // For oauth2
	OpenIDConnectURL string                 `json:"openIdConnectUrl,omitempty"` // For openIdConnect
	Extensions       map[string]interface{} `json:"-"`                          // For custom extensions
}

type OpenAPIOAuth2Flows struct {
	Implicit          *OpenAPIOAuth2Flow `json:"implicit,omitempty"`
	Password          *OpenAPIOAuth2Flow `json:"password,omitempty"`
	ClientCredentials *OpenAPIOAuth2Flow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OpenAPIOAuth2Flow `json:"authorizationCode,omitempty"`
}

type OpenAPIOAuth2Flow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

// Security Requirement (used in operations)
type OpenAPISecurityRequirement map[string][]string

// Security Provider Interface - allows extensibility
type SecurityProvider interface {
	// RegisterSecuritySchemes registers security schemes with the OpenAPI spec
	RegisterSecuritySchemes(spec *OpenAPISpec)

	// GetSecurityRequirements returns security requirements for a given handler/operation
	GetSecurityRequirements(handlerInfo *HandlerInfo) []OpenAPISecurityRequirement

	// ValidateSecurityRequirement validates if a security requirement is properly configured
	ValidateSecurityRequirement(requirement OpenAPISecurityRequirement) error
}

// Default Security Provider Implementation
type DefaultSecurityProvider struct {
	schemes map[string]OpenAPISecurityScheme
}

func NewDefaultSecurityProvider() *DefaultSecurityProvider {
	return &DefaultSecurityProvider{
		schemes: make(map[string]OpenAPISecurityScheme),
	}
}

func (p *DefaultSecurityProvider) RegisterSecuritySchemes(spec *OpenAPISpec) {
	if spec.Components.SecuritySchemes == nil {
		spec.Components.SecuritySchemes = make(map[string]OpenAPISecurityScheme)
	}

	for name, scheme := range p.schemes {
		spec.Components.SecuritySchemes[name] = scheme
	}
}

func (p *DefaultSecurityProvider) GetSecurityRequirements(handlerInfo *HandlerInfo) []OpenAPISecurityRequirement {
	// Default implementation returns empty - can be overridden
	return []OpenAPISecurityRequirement{}
}

func (p *DefaultSecurityProvider) ValidateSecurityRequirement(requirement OpenAPISecurityRequirement) error {
	// Basic validation - check if referenced schemes exist
	for schemeName := range requirement {
		if _, exists := p.schemes[schemeName]; !exists {
			return fmt.Errorf("security scheme '%s' not found", schemeName)
		}
	}
	return nil
}

type SecuritySchemeBuilder struct {
	scheme OpenAPISecurityScheme
}

func NewSecurityScheme(schemeType string) *SecuritySchemeBuilder {
	return &SecuritySchemeBuilder{
		scheme: OpenAPISecurityScheme{
			Type: schemeType,
		},
	}
}

func (b *SecuritySchemeBuilder) Description(desc string) *SecuritySchemeBuilder {
	b.scheme.Description = desc
	return b
}

func (b *SecuritySchemeBuilder) APIKey(name, in string) *SecuritySchemeBuilder {
	if b.scheme.Type != SecurityTypeAPIKey {
		panic("APIKey configuration only valid for apiKey type")
	}
	b.scheme.Name = name
	b.scheme.In = in
	return b
}

func (b *SecuritySchemeBuilder) HTTPScheme(scheme string) *SecuritySchemeBuilder {
	if b.scheme.Type != SecurityTypeHTTP {
		panic("HTTPScheme configuration only valid for http type")
	}
	b.scheme.Scheme = scheme
	return b
}

func (b *SecuritySchemeBuilder) BearerFormat(format string) *SecuritySchemeBuilder {
	if b.scheme.Type != SecurityTypeHTTP || b.scheme.Scheme != HTTPSchemeBearer {
		panic("BearerFormat only valid for http bearer type")
	}
	b.scheme.BearerFormat = format
	return b
}

func (b *SecuritySchemeBuilder) OAuth2Flows(flows *OpenAPIOAuth2Flows) *SecuritySchemeBuilder {
	if b.scheme.Type != SecurityTypeOAuth2 {
		panic("OAuth2Flows configuration only valid for oauth2 type")
	}
	b.scheme.Flows = flows
	return b
}

func (b *SecuritySchemeBuilder) OpenIDConnectURL(url string) *SecuritySchemeBuilder {
	if b.scheme.Type != SecurityTypeOpenIDConnect {
		panic("OpenIDConnectURL configuration only valid for openIdConnect type")
	}
	b.scheme.OpenIDConnectURL = url
	return b
}

func (b *SecuritySchemeBuilder) Build() OpenAPISecurityScheme {
	return b.scheme
}

// SecurityHandlerOption handler options for security
type SecurityHandlerOption struct {
	requirements []OpenAPISecurityRequirement
}

func (o *SecurityHandlerOption) ApplyToHandler(info *HandlerInfo) {
	if info.SecurityRequirements == nil {
		info.SecurityRequirements = make([]OpenAPISecurityRequirement, 0)
	}
	info.SecurityRequirements = append(info.SecurityRequirements, o.requirements...)
}

// Security requirement builders
func WithSecurity(requirements ...OpenAPISecurityRequirement) HandlerOption {
	return func(h *HandlerInfo) {
		if h.SecurityRequirements == nil {
			h.SecurityRequirements = make([]OpenAPISecurityRequirement, 0)
		}
		h.SecurityRequirements = append(h.SecurityRequirements, requirements...)
	}
}

func RequireAPIKey(schemeName string) OpenAPISecurityRequirement {
	return OpenAPISecurityRequirement{
		schemeName: []string{},
	}
}

func RequireBearer(schemeName string) OpenAPISecurityRequirement {
	return OpenAPISecurityRequirement{
		schemeName: []string{},
	}
}

func RequireOAuth2(schemeName string, scopes ...string) OpenAPISecurityRequirement {
	return OpenAPISecurityRequirement{
		schemeName: scopes,
	}
}

func RequireBasicAuth(schemeName string) OpenAPISecurityRequirement {
	return OpenAPISecurityRequirement{
		schemeName: []string{},
	}
}

func APIKeyInHeader(name, description string) OpenAPISecurityScheme {
	return NewSecurityScheme(SecurityTypeAPIKey).
		APIKey(name, APIKeyInHeaderType).
		Description(description).
		Build()
}

func APIKeyInQuery(name, description string) OpenAPISecurityScheme {
	return NewSecurityScheme(SecurityTypeAPIKey).
		APIKey(name, APIKeyInQueryType).
		Description(description).
		Build()
}

func BearerAuth(description, format string) OpenAPISecurityScheme {
	builder := NewSecurityScheme(SecurityTypeHTTP).
		HTTPScheme(HTTPSchemeBearer).
		Description(description)

	if format != "" {
		builder = builder.BearerFormat(format)
	}

	return builder.Build()
}

func BasicAuth(description string) OpenAPISecurityScheme {
	return NewSecurityScheme(SecurityTypeHTTP).
		HTTPScheme(HTTPSchemeBasic).
		Description(description).
		Build()
}

func OAuth2AuthorizationCode(authURL, tokenURL string, scopes map[string]string, description string) OpenAPISecurityScheme {
	flows := &OpenAPIOAuth2Flows{
		AuthorizationCode: &OpenAPIOAuth2Flow{
			AuthorizationURL: authURL,
			TokenURL:         tokenURL,
			Scopes:           scopes,
		},
	}

	return NewSecurityScheme(SecurityTypeOAuth2).
		OAuth2Flows(flows).
		Description(description).
		Build()
}

func OAuth2ClientCredentials(tokenURL string, scopes map[string]string, description string) OpenAPISecurityScheme {
	flows := &OpenAPIOAuth2Flows{
		ClientCredentials: &OpenAPIOAuth2Flow{
			TokenURL: tokenURL,
			Scopes:   scopes,
		},
	}

	return NewSecurityScheme(SecurityTypeOAuth2).
		OAuth2Flows(flows).
		Description(description).
		Build()
}

// RegisterSecurityScheme Enhanced Router methods for security
func (r *SteelRouter) RegisterSecurityScheme(name string, scheme OpenAPISecurityScheme) {
	if r.securityProvider == nil {
		r.securityProvider = NewDefaultSecurityProvider()
	}

	if defaultProvider, ok := r.securityProvider.(*DefaultSecurityProvider); ok {
		defaultProvider.schemes[name] = scheme
	}

	// Ensure components exist
	if r.openAPISpec.Components.SecuritySchemes == nil {
		r.openAPISpec.Components.SecuritySchemes = make(map[string]OpenAPISecurityScheme)
	}

	r.openAPISpec.Components.SecuritySchemes[name] = scheme
}

func (r *SteelRouter) SetSecurityProvider(provider SecurityProvider) {
	r.securityProvider = provider

	// Re-register schemes with the new provider
	if r.openAPISpec != nil {
		provider.RegisterSecuritySchemes(r.openAPISpec)
	}
}

func (r *SteelRouter) SetOpenAPIInfo(info OpenAPIInfo) {
	r.openAPISpec.Info = info
}

func (r *SteelRouter) AddServer(server OpenAPIServer) {
	r.openAPISpec.Servers = append(r.openAPISpec.Servers, server)
}

func (r *SteelRouter) SetJSONSchemaDialect(dialect string) {
	// Only set if explicitly requested by the user
	// Common values:
	// "https://spec.openapis.org/oas/3.1/dialect/base" - Default OpenAPI 3.1.1
	// "https://json-schema.org/draft/2020-12/schema" - Full JSON Schema Draft 2020-12
	// "" - Omit the field (recommended)

	if dialect == "" {
		// Omit the field by not setting it
		return
	}

	if ValidateJSONSchemaDialect(dialect) != nil {
		return
	}

	r.openAPISpec.JSONSchemaDialect = dialect
}

// AddWebhook Add webhook support (new in OpenAPI 3.1.1)
func (r *SteelRouter) AddWebhook(name string, pathItem OpenAPIPath) {
	if r.openAPISpec.Webhooks == nil {
		r.openAPISpec.Webhooks = make(map[string]OpenAPIPath)
	}
	r.openAPISpec.Webhooks[name] = pathItem
}

// SetGlobalSecurity Global security requirements (applied to all operations unless overridden)
func (r *SteelRouter) SetGlobalSecurity(requirements ...OpenAPISecurityRequirement) {
	r.globalSecurity = requirements
}

func (r *SteelRouter) GetOpenAPISpec() *OpenAPISpec {
	// Return a copy to prevent external modifications
	spec := *r.openAPISpec
	return &spec
}

// generateBaseOperation creates the base OpenAPI operation from handler info
func (r *SteelRouter) generateBaseOperation(info *HandlerInfo) OpenAPIOperation {
	operation := OpenAPIOperation{
		Summary:     info.Summary,
		Description: info.Description,
		Tags:        info.Tags,
		Parameters:  []OpenAPIParameter{},
		Responses:   make(map[string]OpenAPIResponse),
		Deprecated:  info.Deprecated,
		OperationID: info.OperationID,
	}

	// Generate parameters from input struct
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

	// Add standard error responses
	r.addStandardErrorResponses(&operation, info.Method)

	return operation
}

// Enhanced OpenAPI generation to include security
func (r *SteelRouter) generateOpenAPIForHandlerWithSecurity(info *HandlerInfo) {
	// Generate base operation
	operation := r.generateBaseOperation(info)

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
				// Log warning or handle validation error
				fmt.Printf("Warning: Invalid security requirement: %v\n", err)
			}
		}
	}

	// Add to spec
	openAPIPath := r.convertToOpenAPIPath(info.Path)
	if r.openAPISpec.Paths[openAPIPath] == nil {
		r.openAPISpec.Paths[openAPIPath] = make(OpenAPIPath)
	}
	r.openAPISpec.Paths[openAPIPath][strings.ToLower(info.Method)] = operation
}

// Pre-defined security schemes for common use cases
var (
	// DefaultJWTBearer JWT Bearer token in Authorization header
	DefaultJWTBearer = BearerAuth("JWT Bearer token authentication", "JWT")

	// DefaultAPIKeyHeader API Key in X-API-Key header
	DefaultAPIKeyHeader = APIKeyInHeader("X-API-Key", "API key authentication via header")

	// DefaultAPIKeyQuery API Key in query parameter
	DefaultAPIKeyQuery = APIKeyInQuery("api_key", "API key authentication via query parameter")

	// DefaultHTTPBasic Basic HTTP authentication
	DefaultHTTPBasic = BasicAuth("HTTP Basic authentication")

	// CommonOAuth2Scopes Common OAuth2 scopes
	CommonOAuth2Scopes = map[string]string{
		"read":  "Read access to protected resources",
		"write": "Write access to protected resources",
		"admin": "Administrative access to all resources",
		"user":  "User profile access",
		"email": "Email address access",
	}
)

// Extensibility example - Custom Security Provider
type CustomSecurityProvider struct {
	*DefaultSecurityProvider
	customSchemes map[string]OpenAPISecurityScheme
}

func NewCustomSecurityProvider() *CustomSecurityProvider {
	return &CustomSecurityProvider{
		DefaultSecurityProvider: NewDefaultSecurityProvider(),
		customSchemes:           make(map[string]OpenAPISecurityScheme),
	}
}

func (p *CustomSecurityProvider) AddCustomScheme(name string, scheme OpenAPISecurityScheme) {
	p.customSchemes[name] = scheme
	p.schemes[name] = scheme
}

func (p *CustomSecurityProvider) RegisterSecuritySchemes(spec *OpenAPISpec) {
	// Call parent implementation
	p.DefaultSecurityProvider.RegisterSecuritySchemes(spec)

	// Add custom schemes
	for name, scheme := range p.customSchemes {
		spec.Components.SecuritySchemes[name] = scheme
	}
}
func WithDeprecated(deprecated bool) HandlerOption {
	return func(h *HandlerInfo) {
		h.Deprecated = deprecated
	}
}

func WithOperationID(operationID string) HandlerOption {
	return func(h *HandlerInfo) {
		h.OperationID = operationID
	}
}

func ValidateJSONSchemaDialect(dialect string) error {
	validDialects := []string{
		"", // Omit (recommended)
		"https://spec.openapis.org/oas/3.1/dialect/base", // OpenAPI 3.1.1 default
		"https://json-schema.org/draft/2020-12/schema",   // Full JSON Schema
	}

	for _, valid := range validDialects {
		if dialect == valid {
			return nil
		}
	}

	return fmt.Errorf("unsupported JSON Schema dialect: %s. Use one of: %v", dialect, validDialects)
}
