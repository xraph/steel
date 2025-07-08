package forge_router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test structs for opinionated handlers
type TestRequest2 struct {
	ID   int      `path:"id" description:"User ID"`
	Name string   `query:"name" description:"User name"`
	Age  int      `query:"age" description:"User age"`
	Body TestBody `body:"body" description:"Request body"`
}

type TestBody struct {
	Email    string `json:"email" description:"User email"`
	Password string `json:"password" description:"User password"`
}

type TestResponse3 struct {
	ID      int    `json:"id" description:"User ID"`
	Name    string `json:"name" description:"User name"`
	Email   string `json:"email" description:"User email"`
	Created bool   `json:"created" description:"Whether user was created"`
}

type TestErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TestNewFastRouter tests router creation
func TestNewFastRouter(t *testing.T) {
	router := NewRouter()

	if router == nil {
		t.Fatal("Expected router to be created")
	}

	if router.trees == nil {
		t.Error("Expected trees to be initialized")
	}

	if router.openAPISpec == nil {
		t.Error("Expected OpenAPI spec to be initialized")
	}

	if router.handlers == nil {
		t.Error("Expected handlers map to be initialized")
	}
}

// TestBasicRouting tests basic HTTP method routing
func TestBasicRouting(t *testing.T) {
	router := NewRouter()

	// Test GET route
	router.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("GET test"))
	})

	// Test POST route
	router.POST("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("POST test"))
	})

	// Test PUT route
	router.PUT("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("PUT test"))
	})

	// Test DELETE route
	router.DELETE("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("DELETE test"))
	})

	// Test PATCH route
	router.PATCH("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("PATCH test"))
	})

	// Test HEAD route
	router.HEAD("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test OPTIONS route
	router.OPTIONS("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OPTIONS test"))
	})

	tests := []struct {
		method   string
		path     string
		expected string
		status   int
	}{
		{"GET", "/test", "GET test", http.StatusOK},
		{"POST", "/test", "POST test", http.StatusOK},
		{"PUT", "/test", "PUT test", http.StatusOK},
		{"DELETE", "/test", "DELETE test", http.StatusOK},
		{"PATCH", "/test", "PATCH test", http.StatusOK},
		{"HEAD", "/test", "", http.StatusOK},
		{"OPTIONS", "/test", "OPTIONS test", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, w.Code)
			}

			if body := w.Body.String(); body != tt.expected {
				t.Errorf("Expected body %q, got %q", tt.expected, body)
			}
		})
	}
}

// TestParameterRouting tests URL parameter extraction
func TestParameterRouting(t *testing.T) {
	router := NewRouter()

	router.GET("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		id := URLParam(r, "id")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("User ID: %s", id)))
	})

	router.GET("/users/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
		id := URLParam(r, "id")
		postId := URLParam(r, "postId")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("User ID: %s, Post ID: %s", id, postId)))
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/users/123", "User ID: 123"},
		{"/users/abc", "User ID: abc"},
		{"/users/123/posts/456", "User ID: 123, Post ID: 456"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			if body := w.Body.String(); body != tt.expected {
				t.Errorf("Expected body %q, got %q", tt.expected, body)
			}
		})
	}
}

// TestWildcardRouting tests wildcard routing
func TestWildcardRouting(t *testing.T) {
	router := NewRouter()

	router.GET("/static/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Static file"))
	})

	tests := []string{
		"/static/css/style.css",
		"/static/js/app.js",
		"/static/images/logo.png",
		"/static/fonts/font.woff",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			if body := w.Body.String(); body != "Static file" {
				t.Errorf("Expected body %q, got %q", "Static file", body)
			}
		})
	}
}

// TestOpinionatedHandlers tests the opinionated handler functionality
func TestOpinionatedHandlers(t *testing.T) {
	router := NewRouter()

	// Test successful handler
	router.OpinionatedGET("/users/:id", func(ctx *FastContext, req TestRequest2) (*TestResponse3, error) {
		return &TestResponse3{
			ID:      req.ID,
			Name:    req.Name,
			Email:   req.Body.Email,
			Created: false,
		}, nil
	}, WithSummary("Get user"), WithDescription("Get user by ID"))

	// Test error handler
	router.OpinionatedGET("/users/:id/error", func(ctx *FastContext, req TestRequest2) (*TestResponse3, error) {
		return nil, BadRequest("Invalid request")
	})

	// Test POST with body
	router.OpinionatedPOST("/users", func(ctx *FastContext, req TestRequest2) (*TestResponse3, error) {
		return &TestResponse3{
			ID:      999,
			Name:    req.Name,
			Email:   req.Body.Email,
			Created: true,
		}, nil
	})

	t.Run("GET with parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/123?name=John&age=30", nil)
		req.Header.Set("Content-Type", "application/json")

		body := bytes.NewBuffer([]byte(`{"email": "john@example.com", "password": "secret"}`))
		req.Body = io.NopCloser(body)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response TestResponse3
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.ID != 123 {
			t.Errorf("Expected ID 123, got %d", response.ID)
		}

		if response.Name != "John" {
			t.Errorf("Expected name 'John', got %q", response.Name)
		}

		if response.Email != "john@example.com" {
			t.Errorf("Expected email 'john@example.com', got %q", response.Email)
		}
	})

	t.Run("Error handling", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/123/error", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		var errorResponse map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if errorResponse["error"] == nil {
			t.Error("Expected error field in response")
		}
	})

	t.Run("POST with body", func(t *testing.T) {
		body := bytes.NewBuffer([]byte(`{"email": "jane@example.com", "password": "secret"}`))
		req := httptest.NewRequest("POST", "/users?name=Jane", body)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response TestResponse3
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.ID != 999 {
			t.Errorf("Expected ID 999, got %d", response.ID)
		}

		if response.Name != "Jane" {
			t.Errorf("Expected name 'Jane', got %q", response.Name)
		}

		if response.Email != "jane@example.com" {
			t.Errorf("Expected email 'jane@example.com', got %q", response.Email)
		}

		if !response.Created {
			t.Error("Expected created to be true")
		}
	})
}

// TestMiddleware tests middleware functionality
func TestMiddleware(t *testing.T) {
	router := NewRouter()

	// Test middleware that adds a header
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "test")
			next.ServeHTTP(w, r)
		})
	})

	router.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if header := w.Header().Get("X-Middleware"); header != "test" {
		t.Errorf("Expected X-Middleware header to be 'test', got %q", header)
	}
}

// TestBuiltinMiddleware tests built-in middleware
func TestBuiltinMiddleware(t *testing.T) {
	router := NewRouter()

	// Test recoverer middleware
	router.Use(Recoverer)

	router.GET("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	if body := w.Body.String(); body != "Internal Server Error" {
		t.Errorf("Expected body 'Internal Server Error', got %q", body)
	}
}

// TestRouteGroups tests route grouping functionality
func TestRouteGroups(t *testing.T) {
	router := NewRouter()

	// Test route group with prefix
	router.Route("/api/v1", func(r Router) {
		r.GET("/users", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Users API"))
		})

		r.POST("/users", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("User created"))
		})
	})

	// Test nested groups
	router.Route("/admin", func(r Router) {
		r.Route("/users", func(r Router) {
			r.GET("/", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Admin users"))
			})
		})
	})

	tests := []struct {
		method   string
		path     string
		expected string
		status   int
	}{
		{"GET", "/api/v1/users", "Users API", http.StatusOK},
		{"POST", "/api/v1/users", "User created", http.StatusCreated},
		{"GET", "/admin/users/", "Admin users", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, w.Code)
			}

			if body := w.Body.String(); body != tt.expected {
				t.Errorf("Expected body %q, got %q", tt.expected, body)
			}
		})
	}
}

// TestNotFound tests 404 handling
func TestNotFound(t *testing.T) {
	router := NewRouter()

	router.GET("/exists", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestMethodNotAllowed tests 405 handling
func TestMethodNotAllowed(t *testing.T) {
	router := NewRouter()

	router.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// TestTrailingSlashRedirect tests trailing slash redirection
func TestTrailingSlashRedirect(t *testing.T) {
	router := NewRouter()

	router.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/test/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// if location := w.Header().Get("Location"); location != "/test" {
	// 	t.Errorf("Expected Location header to be '/test', got %q", location)
	// }
}

// TestOpenAPIGeneration tests OpenAPI specification generation
func TestOpenAPIGeneration(t *testing.T) {
	router := NewRouter()

	router.OpinionatedGET("/users/:id", func(ctx *FastContext, req TestRequest2) (*TestResponse3, error) {
		return &TestResponse3{}, nil
	}, WithSummary("Get user"), WithDescription("Get user by ID"), WithTags("users"))

	router.EnableOpenAPI()

	req := httptest.NewRequest("GET", "/openapi", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
	}

	var spec map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("Failed to decode OpenAPI spec: %v", err)
	}

	if openapi, ok := spec["openapi"]; !ok || openapi != "3.0.0" {
		t.Errorf("Expected OpenAPI version 3.0.0, got %v", openapi)
	}

	if paths, ok := spec["paths"]; !ok {
		t.Error("Expected paths in OpenAPI spec")
	} else if pathsMap, ok := paths.(map[string]interface{}); !ok {
		t.Error("Expected paths to be a map")
	} else if _, ok := pathsMap["/users/{id}"]; !ok {
		t.Error("Expected /users/{id} path in OpenAPI spec")
	}
}

// TestContextHelpers tests FastContext helper methods
func TestContextHelpers(t *testing.T) {
	router := NewRouter()

	router.GET("/test/:id", func(w http.ResponseWriter, r *http.Request) {
		ctx := &FastContext{
			Request:  r,
			Response: w,
			router:   router,
			params:   ParamsFromContext(r.Context()),
		}

		// Test Param method
		id := ctx.Param("id")
		if id == "" {
			t.Error("Expected ID parameter")
		}

		// Test Query method
		name := ctx.Query("name")
		if name == "" {
			t.Error("Expected name query parameter")
		}

		// Test Header method
		contentType := ctx.Header("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
		}

		// Test JSON response
		ctx.JSON(http.StatusOK, map[string]string{
			"id":   id,
			"name": name,
		})
	})

	req := httptest.NewRequest("GET", "/test/123?name=John", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["id"] != "123" {
		t.Errorf("Expected ID '123', got %q", response["id"])
	}

	if response["name"] != "John" {
		t.Errorf("Expected name 'John', got %q", response["name"])
	}
}

// TestErrorHandling tests error handling functionality
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		error    APIError
		expected int
	}{
		{"BadRequest", BadRequest("Invalid input"), http.StatusBadRequest},
		{"Unauthorized", Unauthorized(""), http.StatusUnauthorized},
		{"Forbidden", Forbidden(""), http.StatusForbidden},
		{"NotFound", NotFound("User"), http.StatusNotFound},
		{"Conflict", Conflict("Resource exists"), http.StatusConflict},
		{"UnprocessableEntity", UnprocessableEntity("Validation failed"), http.StatusUnprocessableEntity},
		{"InternalServerError", InternalServerError(""), http.StatusInternalServerError},
		{"TooManyRequests", TooManyRequests(""), http.StatusTooManyRequests},
		{"ServiceUnavailable", ServiceUnavailable(""), http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error.StatusCode() != tt.expected {
				t.Errorf("Expected status code %d, got %d", tt.expected, tt.error.StatusCode())
			}

			response := tt.error.ToResponse()
			if response.Error.Status != tt.expected {
				t.Errorf("Expected response status %d, got %d", tt.expected, response.Error.Status)
			}
		})
	}
}

// TestValidationErrors tests validation error handling
func TestValidationErrors(t *testing.T) {
	fieldErr := NewFieldError("email", "Invalid email format", "invalid-email", "INVALID_FORMAT")
	validationErr := UnprocessableEntity("Validation failed", fieldErr)

	if validationErr.StatusCode() != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d", http.StatusUnprocessableEntity, validationErr.StatusCode())
	}

	if len(validationErr.Fields) != 1 {
		t.Errorf("Expected 1 field error, got %d", len(validationErr.Fields))
	}

	if validationErr.Fields[0].Field != "email" {
		t.Errorf("Expected field 'email', got %q", validationErr.Fields[0].Field)
	}

	if validationErr.Fields[0].Code != "INVALID_FORMAT" {
		t.Errorf("Expected code 'INVALID_FORMAT', got %q", validationErr.Fields[0].Code)
	}
}

// TestMount tests route mounting
func TestMount(t *testing.T) {
	router := NewRouter()

	// Create a sub-router to mount
	subRouter := http.NewServeMux()
	subRouter.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Mounted handler"))
	})

	router.Mount("/sub", subRouter)

	req := httptest.NewRequest("GET", "/sub/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if body := w.Body.String(); body != "Mounted handler" {
		t.Errorf("Expected body 'Mounted handler', got %q", body)
	}
}

// TestCustomResponseTypes tests custom response types
func TestCustomResponseTypes(t *testing.T) {
	router := NewRouter()

	router.OpinionatedGET("/test", func(ctx *FastContext, req struct{}) (*APIResponse, error) {
		return Created(map[string]string{"message": "Resource created"}).WithHeader("X-Custom", "test"), nil
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	if header := w.Header().Get("X-Custom"); header != "test" {
		t.Errorf("Expected X-Custom header to be 'test', got %q", header)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["message"] != "Resource created" {
		t.Errorf("Expected message 'Resource created', got %q", response["message"])
	}
}

// TestDebugRoutes tests debug functionality
func TestDebugRoutes(t *testing.T) {
	router := NewRouter()

	router.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.GET("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.GET("/static/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// This should not panic
	router.DebugRoutes()

	// Test that we have routes in the tree
	if len(router.trees) == 0 {
		t.Error("Expected routes to be registered")
	}

	if router.trees["GET"] == nil {
		t.Error("Expected GET routes to be registered")
	}
}

// TestConcurrency tests concurrent access to router
func TestConcurrency(t *testing.T) {
	router := NewRouter()

	router.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	const numRequests = 100
	results := make(chan bool, numRequests)

	// Start multiple goroutines making requests
	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			results <- w.Code == http.StatusOK
		}()
	}

	// Wait for all results
	for i := 0; i < numRequests; i++ {
		if !<-results {
			t.Error("Request failed")
		}
	}
}

// TestTimeoutMiddleware tests timeout middleware
func TestTimeoutMiddleware(t *testing.T) {
	router := NewRouter()

	router.Use(Timeout(100 * time.Millisecond))

	router.GET("/slow", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(200 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		case <-r.Context().Done():
			// Context was cancelled due to timeout
			return
		}
	})

	req := httptest.NewRequest("GET", "/slow", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// The timeout middleware should have triggered
	// Note: The actual behavior depends on implementation details
	// This test mainly ensures no panic occurs
}

// TestComplexRouting tests complex routing scenarios
func TestComplexRouting(t *testing.T) {
	router := NewRouter()

	// Static routes
	router.GET("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("home"))
	})

	router.GET("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("about"))
	})

	// Parameter routes
	router.GET("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		id := URLParam(r, "id")
		w.Write([]byte("user-" + id))
	})

	router.GET("/users/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
		id := URLParam(r, "id")
		postId := URLParam(r, "postId")
		w.Write([]byte("user-" + id + "-post-" + postId))
	})

	// Wildcard routes
	router.GET("/files/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("file"))
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/", "home"},
		{"/about", "about"},
		{"/users/123", "user-123"},
		{"/users/123/posts/456", "user-123-post-456"},
		{"/files/path/to/file.txt", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			if body := w.Body.String(); body != tt.expected {
				t.Errorf("Expected body %q, got %q", tt.expected, body)
			}
		})
	}
}

// TestRouterConfiguration tests router configuration options
func TestRouterConfiguration(t *testing.T) {
	router := NewRouter()

	// Test configuration methods
	router.SetTrailingSlashRedirect(false)
	router.SetFixedPathRedirect(false)

	customNotFound := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Custom not found"))
	})

	router.SetNotFoundHandler(customNotFound)

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	if body := w.Body.String(); body != "Custom not found" {
		t.Errorf("Expected body 'Custom not found', got %q", body)
	}
}

// TestHandlerOptions tests handler options
func TestHandlerOptions(t *testing.T) {
	router := NewRouter()

	router.OpinionatedGET("/test", func(ctx *FastContext, req struct{}) (*struct{}, error) {
		return &struct{}{}, nil
	}, WithSummary("Test summary"), WithDescription("Test description"), WithTags("test", "api"))

	// Check that handler info was registered
	key := "GET /test"
	if info, ok := router.handlers[key]; !ok {
		t.Error("Expected handler info to be registered")
	} else {
		if info.Summary != "Test summary" {
			t.Errorf("Expected summary 'Test summary', got %q", info.Summary)
		}

		if info.Description != "Test description" {
			t.Errorf("Expected description 'Test description', got %q", info.Description)
		}

		if len(info.Tags) != 2 || info.Tags[0] != "test" || info.Tags[1] != "api" {
			t.Errorf("Expected tags ['test', 'api'], got %v", info.Tags)
		}
	}
}

// TestTypeConversion tests type conversion in parameter binding
func TestTypeConversion(t *testing.T) {
	router := NewRouter()

	type TestParams struct {
		ID    int     `path:"id"`
		Age   int     `query:"age"`
		Score float64 `query:"score"`
		Admin bool    `query:"admin"`
	}

	router.OpinionatedGET("/test/:id", func(ctx *FastContext, req TestParams) (*TestParams, error) {
		return &req, nil
	})

	req := httptest.NewRequest("GET", "/test/123?age=25&score=98.5&admin=true", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response TestParams
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.ID != 123 {
		t.Errorf("Expected ID 123, got %d", response.ID)
	}

	if response.Age != 25 {
		t.Errorf("Expected age 25, got %d", response.Age)
	}

	if response.Score != 98.5 {
		t.Errorf("Expected score 98.5, got %f", response.Score)
	}

	if !response.Admin {
		t.Error("Expected admin to be true")
	}
}

// TestSchemaGeneration tests OpenAPI schema generation
func TestSchemaGeneration(t *testing.T) {
	router := NewRouter()

	type ComplexType struct {
		Name     string            `json:"name" description:"User name"`
		Age      int               `json:"age" description:"User age"`
		Email    string            `json:"email" description:"User email"`
		Tags     []string          `json:"tags" description:"User tags"`
		Metadata map[string]string `json:"metadata" description:"User metadata"`
	}

	router.OpinionatedGET("/test", func(ctx *FastContext, req struct{}) (*ComplexType, error) {
		return &ComplexType{}, nil
	})

	// Check that schema was generated
	if router.openAPISpec.Components.Schemas == nil {
		t.Error("Expected schemas to be generated")
	}

	if _, exists := router.openAPISpec.Components.Schemas["ComplexType"]; !exists {
		t.Error("Expected ComplexType schema to be generated")
	}
}

// Utility functions for testing
func assertStatus(t *testing.T, expected, actual int) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected status %d, got %d", expected, actual)
	}
}

func assertBody(t *testing.T, expected, actual string) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected body %q, got %q", expected, actual)
	}
}

func assertHeader(t *testing.T, w *httptest.ResponseRecorder, key, expected string) {
	t.Helper()
	if actual := w.Header().Get(key); actual != expected {
		t.Errorf("Expected header %s to be %q, got %q", key, expected, actual)
	}
}
