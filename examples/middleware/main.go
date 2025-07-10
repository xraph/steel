package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xraph/steel"
	"github.com/xraph/steel/middleware"
)

// Example JWT secret (use environment variable in production)
var jwtSecret = []byte("your-secret-key-change-this-in-production")

// User represents a user in our system
type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

// Custom claims for JWT
type CustomClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// API Request/Response types
type LoginRequest struct {
	Username string `json:"username" description:"User's username"`
	Password string `json:"password" description:"User's password"`
}

type LoginResponse struct {
	Token     string    `json:"token" description:"JWT access token"`
	ExpiresAt time.Time `json:"expires_at" description:"Token expiration time"`
	User      User      `json:"user" description:"User information"`
}

type CreateUserRequest struct {
	Username string   `json:"username" description:"Username for the new user"`
	Email    string   `json:"email" description:"Email address"`
	Password string   `json:"password" description:"User password"`
	Roles    []string `json:"roles,omitempty" description:"User roles"`
}

type CreateUserResponse struct {
	ID      string `json:"id" description:"Generated user ID"`
	Message string `json:"message" description:"Success message"`
}

type GetUserRequest struct {
	UserID string `path:"user_id" description:"User ID to retrieve"`
}

type GetUserResponse struct {
	User User `json:"user" description:"User information"`
}

type ListUsersRequest struct {
	Page     int    `query:"page" description:"Page number (default: 1)"`
	PageSize int    `query:"page_size" description:"Page size (default: 10)"`
	Role     string `query:"role" description:"Filter by role"`
}

type ListUsersResponse struct {
	Users []User `json:"users" description:"List of users"`
	Total int    `json:"total" description:"Total number of users"`
	Page  int    `json:"page" description:"Current page"`
}

// Mock user storage (use database in production)
var users = map[string]User{
	"user1": {
		ID:       "user1",
		Username: "admin",
		Email:    "admin@example.com",
		Roles:    []string{"admin", "user"},
	},
	"user2": {
		ID:       "user2",
		Username: "john",
		Email:    "john@example.com",
		Roles:    []string{"user"},
	},
}

var userCredentials = map[string]string{
	"admin": "password123",
	"john":  "password456",
}

func main() {
	router := steel.NewRouter()

	router.EnableOpenAPI()
	router.EnableAsyncAPI()

	// ==========================================================================
	// Configure OpenAPI documentation with security schemes
	// ==========================================================================
	router.OpenAPI().
		SetTitle("User Management API").
		SetVersion("2.0.0").
		SetDescription("Comprehensive user management API with JWT authentication").
		SetContact("API Team", "https://example.com/support", "api@example.com").
		SetLicense("MIT", "https://opensource.org/licenses/MIT").

		// Add multiple servers
		AddDevelopmentServer(8080).
		AddServer("https://api.example.com", "Production").
		AddServer("https://staging-api.example.com", "Staging").

		// Configure authentication schemes
		AddBearerAuth("JWTAuth", "JWT Bearer token authentication", "JWT").
		AddAPIKeyAuth("ApiKeyAuth", "X-API-Key", "API key authentication").

		// Set global security (users can use either JWT or API key)
		RequireAnyAuth("JWTAuth", "ApiKeyAuth").

		// Add comprehensive tags
		AddTag("auth", "Authentication and authorization").
		AddTag("users", "User management operations").
		AddTag("admin", "Administrative operations").

		// External documentation
		SetExternalDocs("https://docs.example.com", "Complete API Documentation").
		Build()

	// ==========================================================================
	// Regular Middleware Stack (applied to all routes)
	// ==========================================================================

	// Basic middleware
	router.Use(middleware.RequestID())
	router.Use(steel.Logger)
	router.Use(steel.Recoverer)

	// Security middleware
	router.Use(middleware.SecureHeaders())
	router.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000", "https://app.example.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-API-Key"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	// Performance middleware
	router.Use(middleware.Compression())
	router.Use(middleware.BodyLimit(10 << 20)) // 10MB limit

	// Monitoring middleware
	metrics := middleware.NewMetrics()
	router.Use(middleware.MetricsMiddleware(metrics, middleware.MetricsConfig{
		SkipFunc: func(r *http.Request) bool {
			// Skip metrics for health checks and static assets
			return r.URL.Path == "/health" || r.URL.Path == "/metrics"
		},
		GroupedPath: func(path string) string {
			// Group similar paths for metrics
			if path == "/api/users" {
				return "/api/users"
			}
			if len(path) > 10 && path[:10] == "/api/users" {
				return "/api/users/{id}"
			}
			return path
		},
	}))

	// ==========================================================================
	// Opinionated Middleware (applied to opinionated handlers)
	// ==========================================================================

	// Rate limiting with different configs for different endpoints
	strictRateLimit := middleware.OpinionatedRateLimit(middleware.RateLimitConfig{
		RequestsPerSecond: 5,  // 5 requests per second
		BurstSize:         10, // Allow bursts up to 10
		KeyFunc: func(r *http.Request) string {
			// Rate limit by IP + User-Agent for stricter control
			return r.RemoteAddr + ":" + r.UserAgent()
		},
	})

	normalRateLimit := middleware.OpinionatedRateLimit(middleware.RateLimitConfig{
		RequestsPerSecond: 50,  // 50 requests per second
		BurstSize:         100, // Allow bursts up to 100
	})

	// JWT Authentication middleware
	jwtAuth := middleware.OpinionatedJWT(middleware.JWTConfig{
		SigningKey:    jwtSecret,
		SigningMethod: jwt.SigningMethodHS256,
		TokenLookup:   "header:Authorization",
		AuthScheme:    "Bearer",
		Claims:        &CustomClaims{},
		SkipFunc: func(r *http.Request) bool {
			// Skip JWT for login endpoint
			return r.URL.Path == "/api/auth/login"
		},
	})

	// Custom authorization middleware
	adminAuth := steel.NewMiddleware("admin_auth").
		Description("Require admin role for access").
		DependsOn("jwt_auth").
		Before(func(ctx *steel.MiddlewareContext) error {
			// Extract user roles from JWT claims
			if claims, exists := ctx.Metadata["jwt_claims"]; exists {
				if customClaims, ok := claims.(*CustomClaims); ok {
					for _, role := range customClaims.Roles {
						if role == "admin" {
							return nil // User has admin role
						}
					}
				}
			}
			return steel.Forbidden("Admin access required")
		}).
		RequiresAuth().
		AddResponse("403", "Forbidden - Admin access required").
		Build()

	// Custom audit logging middleware
	auditLog := steel.NewMiddleware("audit_log").
		Description("Audit logging for sensitive operations").
		After(func(ctx *steel.MiddlewareContext) error {
			// Log sensitive operations
			if ctx.Request.Method != "GET" { // Log all non-GET operations
				userID := middleware.GetUserID(ctx.Request)
				requestID := middleware.GetRequestID(ctx.Request)

				log.Printf("AUDIT: User %s performed %s %s (Request ID: %s, Status: %d)",
					userID, ctx.Request.Method, ctx.Request.URL.Path, requestID, ctx.StatusCode)
			}
			return nil
		}).
		Build()

	// Apply different middleware combinations to different route groups

	// ==========================================================================
	// Public Routes (no authentication required)
	// ==========================================================================

	// Health check endpoint (minimal middleware)
	router.GET("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
	})

	// Metrics endpoint
	router.GET("/metrics", func(w http.ResponseWriter, r *http.Request) {
		stats := metrics.GetStats()
		w.Header().Set("Content-Type", "application/json")

		// Convert to JSON manually for simplicity
		w.Write([]byte(`{"metrics": "available", "endpoints": ` + fmt.Sprintf("%d", len(stats)) + `}`))
	})

	// ==========================================================================
	// Authentication Routes
	// ==========================================================================

	router.Route("/api/auth", func(r steel.Router) {
		// Apply strict rate limiting to auth endpoints
		r.UseOpinionated(strictRateLimit)

		// Login endpoint (no JWT required, but rate limited)
		r.OpinionatedPOST("/login", func(ctx *steel.Context, req LoginRequest) (*LoginResponse, error) {
			// Validate credentials
			expectedPassword, exists := userCredentials[req.Username]
			if !exists || expectedPassword != req.Password {
				return nil, ctx.Unauthorized("Invalid username or password")
			}

			// Find user
			var user User
			for _, u := range users {
				if u.Username == req.Username {
					user = u
					break
				}
			}

			// Create JWT token
			expiresAt := time.Now().Add(24 * time.Hour)
			claims := &CustomClaims{
				UserID:   user.ID,
				Username: user.Username,
				Roles:    user.Roles,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(expiresAt),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					Issuer:    "user-api",
					Subject:   user.ID,
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, err := token.SignedString(jwtSecret)
			if err != nil {
				return nil, ctx.InternalError("Failed to generate token")
			}

			return &LoginResponse{
				Token:     tokenString,
				ExpiresAt: expiresAt,
				User:      user,
			}, nil
		},
			steel.WithSummary("User login"),
			steel.WithDescription("Authenticate user and return JWT token"),
			steel.WithTags("auth"),
		)
	})

	// ==========================================================================
	// User Management Routes (requires authentication)
	// ==========================================================================

	router.Route("/api/users", func(r steel.Router) {
		// Apply authentication and normal rate limiting
		r.UseOpinionated(normalRateLimit, jwtAuth, auditLog)

		// Get current user info
		r.OpinionatedGET("/me", func(ctx *steel.Context, req struct{}) (*GetUserResponse, error) {
			userID := middleware.GetUserID(ctx.Request)
			if userID == "" {
				return nil, ctx.Unauthorized("User ID not found in token")
			}

			user, exists := users[userID]
			if !exists {
				return nil, ctx.NotFound("User")
			}

			return &GetUserResponse{User: user}, nil
		},
			steel.WithSummary("Get current user"),
			steel.WithDescription("Get information about the currently authenticated user"),
			steel.WithTags("users", "profile"),
		)

		// List users (basic users can see limited info)
		r.OpinionatedGET("", func(ctx *steel.Context, req ListUsersRequest) (*ListUsersResponse, error) {
			// Set defaults
			if req.Page <= 0 {
				req.Page = 1
			}
			if req.PageSize <= 0 {
				req.PageSize = 10
			}

			var filteredUsers []User
			for _, user := range users {
				// Filter by role if specified
				if req.Role != "" {
					hasRole := false
					for _, role := range user.Roles {
						if role == req.Role {
							hasRole = true
							break
						}
					}
					if !hasRole {
						continue
					}
				}
				filteredUsers = append(filteredUsers, user)
			}

			// Simple pagination
			start := (req.Page - 1) * req.PageSize
			end := start + req.PageSize
			if start >= len(filteredUsers) {
				filteredUsers = []User{}
			} else if end > len(filteredUsers) {
				filteredUsers = filteredUsers[start:]
			} else {
				filteredUsers = filteredUsers[start:end]
			}

			return &ListUsersResponse{
				Users: filteredUsers,
				Total: len(users),
				Page:  req.Page,
			}, nil
		},
			steel.WithSummary("List users"),
			steel.WithDescription("List users with optional filtering and pagination"),
			steel.WithTags("users"),
		)

		// Get specific user
		r.OpinionatedGET("/{user_id}", func(ctx *steel.Context, req GetUserRequest) (*GetUserResponse, error) {
			user, exists := users[req.UserID]
			if !exists {
				return nil, ctx.NotFound("User")
			}

			return &GetUserResponse{User: user}, nil
		},
			steel.WithSummary("Get user by ID"),
			steel.WithDescription("Get detailed information about a specific user"),
			steel.WithTags("users"),
		)
	})

	// ==========================================================================
	// Admin Routes (requires admin role)
	// ==========================================================================

	router.Route("/api/admin", func(r steel.Router) {
		// Apply authentication, admin authorization, and audit logging
		r.UseOpinionated(normalRateLimit, jwtAuth, adminAuth, auditLog)

		// Create new user (admin only)
		r.OpinionatedPOST("/users", func(ctx *steel.Context, req CreateUserRequest) (*CreateUserResponse, error) {
			// Generate user ID
			userID := generateUserID()

			// Create new user
			newUser := User{
				ID:       userID,
				Username: req.Username,
				Email:    req.Email,
				Roles:    req.Roles,
			}

			// Check if username already exists
			for _, user := range users {
				if user.Username == req.Username {
					return nil, ctx.Conflict("Username already exists")
				}
			}

			// Store user
			users[userID] = newUser
			userCredentials[req.Username] = req.Password

			return &CreateUserResponse{
				ID:      userID,
				Message: "User created successfully",
			}, nil
		},
			steel.WithSummary("Create new user"),
			steel.WithDescription("Create a new user (admin only)"),
			steel.WithTags("admin", "users"),
		)

		// Admin-only user statistics
		r.OpinionatedGET("/stats", func(ctx *steel.Context, req struct{}) (*struct {
			TotalUsers     int                    `json:"total_users"`
			UsersByRole    map[string]int         `json:"users_by_role"`
			RequestMetrics map[string]interface{} `json:"request_metrics"`
		}, error) {
			// Calculate user statistics
			usersByRole := make(map[string]int)
			for _, user := range users {
				for _, role := range user.Roles {
					usersByRole[role]++
				}
			}

			return &struct {
				TotalUsers     int                    `json:"total_users"`
				UsersByRole    map[string]int         `json:"users_by_role"`
				RequestMetrics map[string]interface{} `json:"request_metrics"`
			}{
				TotalUsers:     len(users),
				UsersByRole:    usersByRole,
				RequestMetrics: metrics.GetStats(),
			}, nil
		},
			steel.WithSummary("Get admin statistics"),
			steel.WithDescription("Get comprehensive statistics (admin only)"),
			steel.WithTags("admin", "metrics"),
		)
	})

	// ==========================================================================
	// Enable Documentation
	// ==========================================================================

	router.EnableOpenAPI()

	// Print middleware information for debugging
	if os.Getenv("DEBUG") == "true" {
		router.PrintMiddlewareInfo()

		// Validate middleware configuration
		if err := router.ValidateMiddleware(); err != nil {
			log.Fatalf("Middleware validation failed: %v", err)
		}
	}

	// ==========================================================================
	// Start Server
	// ==========================================================================

	port := 8080
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = 8080 // Simple fallback, use proper parsing in production
	}

	fmt.Printf("🚀 Server starting on :%d\n", port)
	fmt.Printf("📚 OpenAPI Docs: http://localhost:%d/openapi/docs\n", port)
	fmt.Printf("🔧 API Spec: http://localhost:%d/openapi.json\n", port)
	fmt.Printf("📈 Health Check: http://localhost:%d/health\n", port)
	fmt.Printf("📊 Metrics: http://localhost:%d/metrics\n", port)
	fmt.Println()
	fmt.Println("Example requests:")
	fmt.Printf("  Login: curl -X POST http://localhost:%d/api/auth/login -H 'Content-Type: application/json' -d '{\"username\":\"admin\",\"password\":\"password123\"}'\n", port)
	fmt.Printf("  Get Users: curl -H 'Authorization: Bearer <token>' http://localhost:%d/api/users\n", port)
	fmt.Printf("  Admin Stats: curl -H 'Authorization: Bearer <token>' http://localhost:%d/api/admin/stats\n", port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}

// Helper function to generate user ID
func generateUserID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("user_%d", time.Now().UnixNano())
	}
	return "user_" + hex.EncodeToString(bytes)
}
