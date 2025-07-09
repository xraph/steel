package forgerouter

import (
	"fmt"
)

// OpenAPIBuilder provides a fluent interface for configuring OpenAPI specifications
type OpenAPIBuilder struct {
	router          *ForgeRouter
	info            *OpenAPIInfo
	servers         []OpenAPIServer
	securitySchemes map[string]OpenAPISecurityScheme
	globalSecurity  []OpenAPISecurityRequirement
	tags            []OpenAPITag
	externalDocs    *OpenAPIExternalDocs
	jsonDialect     string
	webhooks        map[string]OpenAPIPath
}

// OpenAPI returns a new OpenAPIBuilder for fluent configuration
func (r *ForgeRouter) OpenAPI() *OpenAPIBuilder {
	return &OpenAPIBuilder{
		router:          r,
		info:            &OpenAPIInfo{},
		servers:         make([]OpenAPIServer, 0),
		securitySchemes: make(map[string]OpenAPISecurityScheme),
		globalSecurity:  make([]OpenAPISecurityRequirement, 0),
		tags:            make([]OpenAPITag, 0),
		webhooks:        make(map[string]OpenAPIPath),
	}
}

// =============================================================================
// Info Configuration
// =============================================================================

// SetTitle sets the API title
func (b *OpenAPIBuilder) SetTitle(title string) *OpenAPIBuilder {
	b.info.Title = title
	return b
}

// SetDescription sets the API description
func (b *OpenAPIBuilder) SetDescription(description string) *OpenAPIBuilder {
	b.info.Description = description
	return b
}

// SetVersion sets the API version
func (b *OpenAPIBuilder) SetVersion(version string) *OpenAPIBuilder {
	b.info.Version = version
	return b
}

// SetSummary sets the API summary (OpenAPI 3.1+)
func (b *OpenAPIBuilder) SetSummary(summary string) *OpenAPIBuilder {
	b.info.Summary = summary
	return b
}

// SetTermsOfService sets the terms of service URL
func (b *OpenAPIBuilder) SetTermsOfService(termsURL string) *OpenAPIBuilder {
	b.info.TermsOfService = termsURL
	return b
}

// SetContact sets the contact information
func (b *OpenAPIBuilder) SetContact(name, url, email string) *OpenAPIBuilder {
	b.info.Contact = &OpenAPIContact{
		Name:  name,
		URL:   url,
		Email: email,
	}
	return b
}

// SetLicense sets the license information
func (b *OpenAPIBuilder) SetLicense(name, url string) *OpenAPIBuilder {
	b.info.License = &OpenAPILicense{
		Name: name,
		URL:  url,
	}
	return b
}

// SetLicenseWithIdentifier sets the license with SPDX identifier (OpenAPI 3.1+)
func (b *OpenAPIBuilder) SetLicenseWithIdentifier(name, identifier string) *OpenAPIBuilder {
	b.info.License = &OpenAPILicense{
		Name:       name,
		Identifier: identifier,
	}
	return b
}

// =============================================================================
// Server Configuration
// =============================================================================

// AddServer adds a server to the API specification
func (b *OpenAPIBuilder) AddServer(url, description string) *OpenAPIBuilder {
	b.servers = append(b.servers, OpenAPIServer{
		URL:         url,
		Description: description,
	})
	return b
}

// AddServerWithVariables adds a server with variables
func (b *OpenAPIBuilder) AddServerWithVariables(url, description string, variables map[string]OpenAPIServerVariable) *OpenAPIBuilder {
	b.servers = append(b.servers, OpenAPIServer{
		URL:         url,
		Description: description,
		Variables:   variables,
	})
	return b
}

// AddDevelopmentServer adds a common development server
func (b *OpenAPIBuilder) AddDevelopmentServer(port int) *OpenAPIBuilder {
	return b.AddServer(fmt.Sprintf("http://localhost:%d", port), "Development server")
}

// AddProductionServer adds a common production server
func (b *OpenAPIBuilder) AddProductionServer(domain string) *OpenAPIBuilder {
	return b.AddServer(fmt.Sprintf("https://%s", domain), "Production server")
}

// =============================================================================
// Security Configuration
// =============================================================================

// RegisterSecurityScheme registers a security scheme
func (b *OpenAPIBuilder) RegisterSecurityScheme(name string, scheme OpenAPISecurityScheme) *OpenAPIBuilder {
	b.securitySchemes[name] = scheme
	return b
}

// AddAPIKeyAuth adds API key authentication in header
func (b *OpenAPIBuilder) AddAPIKeyAuth(schemeName, keyName, description string) *OpenAPIBuilder {
	return b.RegisterSecurityScheme(schemeName, APIKeyInHeader(keyName, description))
}

// AddAPIKeyAuthQuery adds API key authentication in query parameter
func (b *OpenAPIBuilder) AddAPIKeyAuthQuery(schemeName, keyName, description string) *OpenAPIBuilder {
	return b.RegisterSecurityScheme(schemeName, APIKeyInQuery(keyName, description))
}

// AddBearerAuth adds JWT Bearer token authentication
func (b *OpenAPIBuilder) AddBearerAuth(schemeName, description, format string) *OpenAPIBuilder {
	return b.RegisterSecurityScheme(schemeName, BearerAuth(description, format))
}

// AddBasicAuth adds HTTP Basic authentication
func (b *OpenAPIBuilder) AddBasicAuth(schemeName, description string) *OpenAPIBuilder {
	return b.RegisterSecurityScheme(schemeName, BasicAuth(description))
}

// AddOAuth2AuthCode adds OAuth2 authorization code flow
func (b *OpenAPIBuilder) AddOAuth2AuthCode(schemeName, authURL, tokenURL, description string, scopes map[string]string) *OpenAPIBuilder {
	return b.RegisterSecurityScheme(schemeName, OAuth2AuthorizationCode(authURL, tokenURL, scopes, description))
}

// AddOAuth2ClientCredentials adds OAuth2 client credentials flow
func (b *OpenAPIBuilder) AddOAuth2ClientCredentials(schemeName, tokenURL, description string, scopes map[string]string) *OpenAPIBuilder {
	return b.RegisterSecurityScheme(schemeName, OAuth2ClientCredentials(tokenURL, scopes, description))
}

// SetGlobalSecurity sets global security requirements
func (b *OpenAPIBuilder) SetGlobalSecurity(requirements ...OpenAPISecurityRequirement) *OpenAPIBuilder {
	b.globalSecurity = requirements
	return b
}

// RequireAuth is a convenience method to set global security for a single scheme
func (b *OpenAPIBuilder) RequireAuth(schemeName string, scopes ...string) *OpenAPIBuilder {
	requirement := OpenAPISecurityRequirement{
		schemeName: scopes,
	}
	return b.SetGlobalSecurity(requirement)
}

// RequireAnyAuth sets up OR-based authentication (user can use any of the specified schemes)
func (b *OpenAPIBuilder) RequireAnyAuth(schemeNames ...string) *OpenAPIBuilder {
	requirements := make([]OpenAPISecurityRequirement, len(schemeNames))
	for i, name := range schemeNames {
		requirements[i] = OpenAPISecurityRequirement{name: []string{}}
	}
	return b.SetGlobalSecurity(requirements...)
}

// RequireAllAuth sets up AND-based authentication (user must use all specified schemes)
func (b *OpenAPIBuilder) RequireAllAuth(schemeNames ...string) *OpenAPIBuilder {
	requirement := make(OpenAPISecurityRequirement)
	for _, name := range schemeNames {
		requirement[name] = []string{}
	}
	return b.SetGlobalSecurity(requirement)
}

// =============================================================================
// Documentation Configuration
// =============================================================================

// AddTag adds a tag to the API specification
func (b *OpenAPIBuilder) AddTag(name, description string) *OpenAPIBuilder {
	b.tags = append(b.tags, OpenAPITag{
		Name:        name,
		Description: description,
	})
	return b
}

// AddTagWithDocs adds a tag with external documentation
func (b *OpenAPIBuilder) AddTagWithDocs(name, description, docsURL, docsDescription string) *OpenAPIBuilder {
	tag := OpenAPITag{
		Name:        name,
		Description: description,
	}
	if docsURL != "" {
		tag.ExternalDocs = &OpenAPIExternalDocs{
			URL:         docsURL,
			Description: docsDescription,
		}
	}
	b.tags = append(b.tags, tag)
	return b
}

// SetExternalDocs sets external documentation for the entire API
func (b *OpenAPIBuilder) SetExternalDocs(url, description string) *OpenAPIBuilder {
	b.externalDocs = &OpenAPIExternalDocs{
		URL:         url,
		Description: description,
	}
	return b
}

// =============================================================================
// Advanced Configuration
// =============================================================================

// SetJSONSchemaDialect sets the JSON Schema dialect (OpenAPI 3.1+)
func (b *OpenAPIBuilder) SetJSONSchemaDialect(dialect string) *OpenAPIBuilder {
	if ValidateJSONSchemaDialect(dialect) == nil {
		b.jsonDialect = dialect
	}
	return b
}

// AddWebhook adds a webhook to the specification (OpenAPI 3.1+)
func (b *OpenAPIBuilder) AddWebhook(name string, pathItem OpenAPIPath) *OpenAPIBuilder {
	b.webhooks[name] = pathItem
	return b
}

// =============================================================================
// Preset Configurations
// =============================================================================

// WithRESTDefaults applies common REST API defaults
func (b *OpenAPIBuilder) WithRESTDefaults(title, version string) *OpenAPIBuilder {
	return b.
		SetTitle(title).
		SetVersion(version).
		SetDescription("REST API built with ForgeRouter").
		AddTag("default", "Default operations").
		AddDevelopmentServer(8080)
}

// WithMicroserviceDefaults applies common microservice defaults
func (b *OpenAPIBuilder) WithMicroserviceDefaults(serviceName, version string) *OpenAPIBuilder {
	return b.
		SetTitle(serviceName+" API").
		SetVersion(version).
		SetDescription("Microservice API for "+serviceName).
		AddTag("health", "Health check operations").
		AddTag("metrics", "Metrics and monitoring").
		AddBearerAuth("BearerAuth", "JWT Bearer token authentication", "JWT").
		RequireAuth("BearerAuth")
}

// WithPublicAPIDefaults applies defaults for public APIs
func (b *OpenAPIBuilder) WithPublicAPIDefaults(title, version string) *OpenAPIBuilder {
	return b.
		SetTitle(title).
		SetVersion(version).
		SetDescription("Public API with comprehensive documentation").
		SetTermsOfService("https://example.com/terms").
		SetContact("API Support", "https://example.com/support", "support@example.com").
		SetLicense("MIT", "https://opensource.org/licenses/MIT").
		AddAPIKeyAuth("ApiKeyAuth", "X-API-Key", "API key for authentication").
		AddTag("authentication", "Authentication operations").
		AddTag("core", "Core API operations").
		RequireAuth("ApiKeyAuth")
}

// WithEnterpriseDefaults applies defaults for enterprise APIs
func (b *OpenAPIBuilder) WithEnterpriseDefaults(title, version string) *OpenAPIBuilder {
	return b.
		SetTitle(title).
		SetVersion(version).
		SetDescription("Enterprise API with advanced security").
		AddBearerAuth("BearerAuth", "JWT Bearer token", "JWT").
		AddOAuth2AuthCode("OAuth2",
			"https://auth.company.com/oauth/authorize",
			"https://auth.company.com/oauth/token",
			"OAuth2 authorization code flow",
			map[string]string{
				"read":  "Read access to resources",
				"write": "Write access to resources",
				"admin": "Administrative access",
			}).
		RequireAnyAuth("BearerAuth", "OAuth2").
		AddTag("admin", "Administrative operations").
		AddTag("users", "User management").
		AddTag("reports", "Reporting and analytics")
}

// =============================================================================
// Build Method
// =============================================================================

// Build applies all the configuration to the router and enables OpenAPI
func (b *OpenAPIBuilder) Build() *ForgeRouter {
	// Apply info configuration
	if b.info.Title != "" || b.info.Version != "" || b.info.Description != "" {
		// Merge with existing info, preserving any existing values for unset fields
		currentInfo := b.router.openAPISpec.Info
		if b.info.Title != "" {
			currentInfo.Title = b.info.Title
		}
		if b.info.Version != "" {
			currentInfo.Version = b.info.Version
		}
		if b.info.Description != "" {
			currentInfo.Description = b.info.Description
		}
		if b.info.Summary != "" {
			currentInfo.Summary = b.info.Summary
		}
		if b.info.TermsOfService != "" {
			currentInfo.TermsOfService = b.info.TermsOfService
		}
		if b.info.Contact != nil {
			currentInfo.Contact = b.info.Contact
		}
		if b.info.License != nil {
			currentInfo.License = b.info.License
		}

		b.router.openAPISpec.Info = currentInfo
		b.router.EnableOpenAPI()
	}

	// Apply servers
	if len(b.servers) > 0 {
		b.router.openAPISpec.Servers = b.servers
	}

	// Apply security schemes
	for name, scheme := range b.securitySchemes {
		b.router.RegisterSecurityScheme(name, scheme)
	}

	// Apply global security
	if len(b.globalSecurity) > 0 {
		b.router.SetGlobalSecurity(b.globalSecurity...)
	}

	// Apply tags
	if len(b.tags) > 0 {
		b.router.openAPISpec.Tags = b.tags
	}

	// Apply external docs
	if b.externalDocs != nil {
		b.router.openAPISpec.ExternalDocs = b.externalDocs
	}

	// Apply JSON Schema dialect
	if b.jsonDialect != "" {
		b.router.SetJSONSchemaDialect(b.jsonDialect)
	}

	// Apply webhooks
	if len(b.webhooks) > 0 {
		if b.router.openAPISpec.Webhooks == nil {
			b.router.openAPISpec.Webhooks = make(map[string]OpenAPIPath)
		}
		for name, pathItem := range b.webhooks {
			b.router.openAPISpec.Webhooks[name] = pathItem
		}
	}

	// Enable OpenAPI
	b.router.EnableOpenAPI()

	return b.router
}

// =============================================================================
// Utility Methods
// =============================================================================

// Preview returns the current OpenAPI specification without applying it
func (b *OpenAPIBuilder) Preview() *OpenAPISpec {
	// Create a temporary copy of the router's spec
	preview := *b.router.openAPISpec

	// Apply all configurations to the preview
	if b.info.Title != "" {
		preview.Info.Title = b.info.Title
	}
	if b.info.Version != "" {
		preview.Info.Version = b.info.Version
	}
	if b.info.Description != "" {
		preview.Info.Description = b.info.Description
	}

	if len(b.servers) > 0 {
		preview.Servers = b.servers
	}

	if len(b.globalSecurity) > 0 {
		preview.Security = b.globalSecurity
	}

	return &preview
}

// Validate checks if the current configuration is valid
func (b *OpenAPIBuilder) Validate() error {
	if b.info.Title == "" {
		return fmt.Errorf("API title is required")
	}
	if b.info.Version == "" {
		return fmt.Errorf("API version is required")
	}

	// Validate security requirements reference existing schemes
	for _, requirement := range b.globalSecurity {
		for schemeName := range requirement {
			if _, exists := b.securitySchemes[schemeName]; !exists {
				return fmt.Errorf("security requirement references unknown scheme: %s", schemeName)
			}
		}
	}

	return nil
}

// Reset clears all configuration and returns a fresh builder
func (b *OpenAPIBuilder) Reset() *OpenAPIBuilder {
	return b.router.OpenAPI()
}
