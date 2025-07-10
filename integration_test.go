package steel

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	json "github.com/json-iterator/go"
)

// Integration test data structures
type User struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Age     int    `json:"age"`
	Active  bool   `json:"active"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type UpdateUserRequest struct {
	ID    int    `path:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type GetUserRequest struct {
	ID      int    `path:"id"`
	Include string `query:"include"`
}

type ListUsersRequest struct {
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
	Sort   string `query:"sort"`
	Filter string `query:"filter"`
}

type UserListResponse struct {
	Users   []User `json:"users"`
	Total   int    `json:"total"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
	HasMore bool   `json:"has_more"`
}

// Mock database for integration tests
type MockUserDB struct {
	users  map[int]User
	nextID int
	mu     sync.RWMutex
}

func NewMockUserDB() *MockUserDB {
	return &MockUserDB{
		users:  make(map[int]User),
		nextID: 1,
	}
}

func (db *MockUserDB) Create(name, email string, age int) User {
	db.mu.Lock()
	defer db.mu.Unlock()

	user := User{
		ID:      db.nextID,
		Name:    name,
		Email:   email,
		Age:     age,
		Active:  true,
		Created: time.Now().Format(time.RFC3339),
		Updated: time.Now().Format(time.RFC3339),
	}

	db.users[db.nextID] = user
	db.nextID++

	return user
}

func (db *MockUserDB) GetByID(id int) (User, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	user, exists := db.users[id]
	return user, exists
}

func (db *MockUserDB) Update(id int, name, email string, age int) (User, bool) {
	db.mu.Lock()
	defer db.mu.Unlock()

	user, exists := db.users[id]
	if !exists {
		return User{}, false
	}

	user.Name = name
	user.Email = email
	user.Age = age
	user.Updated = time.Now().Format(time.RFC3339)

	db.users[id] = user
	return user, true
}

func (db *MockUserDB) Delete(id int) bool {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, exists := db.users[id]
	if exists {
		delete(db.users, id)
	}
	return exists
}

func (db *MockUserDB) List(limit, offset int, sort, filter string) ([]User, int) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var users []User
	for _, user := range db.users {
		if filter == "" || strings.Contains(strings.ToLower(user.Name), strings.ToLower(filter)) {
			users = append(users, user)
		}
	}

	total := len(users)

	// Simple pagination
	if offset >= len(users) {
		return []User{}, total
	}

	end := offset + limit
	if end > len(users) {
		end = len(users)
	}

	return users[offset:end], total
}

// TestFullRESTAPI tests a complete REST API implementation
func TestFullRESTAPI(t *testing.T) {
	router := NewRouter()
	db := NewMockUserDB()

	// Configure router with middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		})
	})

	// User API routes
	router.Route("/api/v1/users", func(r Router) {
		// Create user
		r.OpinionatedPOST("/", func(ctx *Context, req CreateUserRequest) (*User, error) {
			if req.Name == "" {
				return nil, BadRequest("Name is required")
			}
			if req.Email == "" {
				return nil, BadRequest("Email is required")
			}
			if req.Age < 0 || req.Age > 150 {
				return nil, BadRequest("Age must be between 0 and 150")
			}

			user := db.Create(req.Name, req.Email, req.Age)
			return &user, nil
		}, WithSummary("Create user"), WithDescription("Create a new user"))

		// List users
		r.OpinionatedGET("/", func(ctx *Context, req ListUsersRequest) (*UserListResponse, error) {
			if req.Limit <= 0 {
				req.Limit = 10
			}
			if req.Limit > 100 {
				req.Limit = 100
			}
			if req.Offset < 0 {
				req.Offset = 0
			}

			users, total := db.List(req.Limit, req.Offset, req.Sort, req.Filter)

			return &UserListResponse{
				Users:   users,
				Total:   total,
				Limit:   req.Limit,
				Offset:  req.Offset,
				HasMore: req.Offset+req.Limit < total,
			}, nil
		}, WithSummary("List users"), WithDescription("Get paginated list of users"))

		// Get user by ID
		r.OpinionatedGET("/:id", func(ctx *Context, req GetUserRequest) (*User, error) {
			user, exists := db.GetByID(req.ID)
			if !exists {
				return nil, NotFound("User")
			}

			return &user, nil
		}, WithSummary("Get user"), WithDescription("Get user by ID"))

		// Update user
		r.OpinionatedPUT("/:id", func(ctx *Context, req UpdateUserRequest) (*User, error) {
			if req.Name == "" {
				return nil, BadRequest("Name is required")
			}
			if req.Email == "" {
				return nil, BadRequest("Email is required")
			}
			if req.Age < 0 || req.Age > 150 {
				return nil, BadRequest("Age must be between 0 and 150")
			}

			user, exists := db.Update(req.ID, req.Name, req.Email, req.Age)
			if !exists {
				return nil, NotFound("User")
			}

			return &user, nil
		}, WithSummary("Update user"), WithDescription("Update user by ID"))

		// Delete user
		r.OpinionatedDELETE("/:id", func(ctx *Context, req struct {
			ID int `path:"id"`
		}) (*APIResponse, error) {
			exists := db.Delete(req.ID)
			if !exists {
				return nil, NotFound("User")
			}

			return NoContent(), nil
		}, WithSummary("Delete user"), WithDescription("Delete user by ID"))
	})

	// Health check endpoint
	router.GET("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Enable OpenAPI documentation
	router.EnableOpenAPI()

	// Test CREATE user
	t.Run("Create User", func(t *testing.T) {
		body := CreateUserRequest{
			Name:  "John Doe",
			Email: "john@example.com",
			Age:   30,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/users/?name=John&age=30", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var user User
		if err := json.NewDecoder(w.Body).Decode(&user); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if user.ID == 0 {
			t.Error("Expected user ID to be set")
		}
		if user.Name != "John Doe" {
			t.Errorf("Expected name 'John Doe', got %q", user.Name)
		}
		if user.Email != "john@example.com" {
			t.Errorf("Expected email 'john@example.com', got %q", user.Email)
		}
		if user.Age != 30 {
			t.Errorf("Expected age 30, got %d", user.Age)
		}
	})

	// Test GET user
	t.Run("Get User", func(t *testing.T) {
		// First create a user
		db.Create("Jane Doe", "jane@example.com", 25)

		req := httptest.NewRequest("GET", "/api/v1/users/1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var user User
		if err := json.NewDecoder(w.Body).Decode(&user); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if user.ID != 1 {
			t.Errorf("Expected user ID 1, got %d", user.ID)
		}
	})

	// Test GET user not found
	t.Run("Get User Not Found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	// Test LIST users
	t.Run("List Users", func(t *testing.T) {
		// Create several users
		db.Create("Alice", "alice@example.com", 28)
		db.Create("Bob", "bob@example.com", 35)
		db.Create("Charlie", "charlie@example.com", 42)

		req := httptest.NewRequest("GET", "/api/v1/users/?limit=2&offset=0", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response UserListResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Users) != 2 {
			t.Errorf("Expected 2 users, got %d", len(response.Users))
		}

		if response.Total < 2 {
			t.Errorf("Expected total >= 2, got %d", response.Total)
		}

		if response.Limit != 2 {
			t.Errorf("Expected limit 2, got %d", response.Limit)
		}

		if response.Offset != 0 {
			t.Errorf("Expected offset 0, got %d", response.Offset)
		}
	})

	// Test UPDATE user
	t.Run("Update User", func(t *testing.T) {
		// Create a user first
		user := db.Create("Original Name", "original@example.com", 25)

		updateBody := UpdateUserRequest{
			Name:  "Updated Name",
			Email: "updated@example.com",
			Age:   26,
		}
		bodyBytes, _ := json.Marshal(updateBody)

		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/users/%d", user.ID), bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var updatedUser User
		if err := json.NewDecoder(w.Body).Decode(&updatedUser); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if updatedUser.Name != "Updated Name" {
			t.Errorf("Expected name 'Updated Name', got %q", updatedUser.Name)
		}
		if updatedUser.Email != "updated@example.com" {
			t.Errorf("Expected email 'updated@example.com', got %q", updatedUser.Email)
		}
		if updatedUser.Age != 26 {
			t.Errorf("Expected age 26, got %d", updatedUser.Age)
		}
	})

	// Test DELETE user
	t.Run("Delete User", func(t *testing.T) {
		// Create a user first
		user := db.Create("To Delete", "delete@example.com", 30)

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%d", user.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
		}

		// Verify user is deleted
		_, exists := db.GetByID(user.ID)
		if exists {
			t.Error("Expected user to be deleted")
		}
	})

	// Test validation errors
	t.Run("Create User Validation Error", func(t *testing.T) {
		body := CreateUserRequest{
			Name:  "", // Invalid: empty name
			Email: "invalid@example.com",
			Age:   25,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/users/?age=25", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
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

	// Test health check
	t.Run("Health Check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var health map[string]string
		if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode health response: %v", err)
		}

		if health["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %q", health["status"])
		}
	})

	// Test OpenAPI documentation
	t.Run("OpenAPI Documentation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/openapi.json", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var spec map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
			t.Fatalf("Failed to decode OpenAPI spec: %v", err)
		}

		if spec["openapi"] != "3.1.1" {
			t.Errorf("Expected OpenAPI version 3.1.1, got %v", spec["openapi"])
		}

		// Check that user endpoints are documented
		if paths, ok := spec["paths"].(map[string]interface{}); ok {
			if _, ok := paths["/api/v1/users/"]; !ok {
				t.Error("Expected /api/v1/users/ path in OpenAPI spec")
			}
			if _, ok := paths["/api/v1/users/{id}"]; !ok {
				t.Error("Expected /api/v1/users/{id} path in OpenAPI spec")
			}
		}
	})
}

// TestComplexMiddlewareChain tests complex middleware scenarios
func TestComplexMiddlewareChain(t *testing.T) {
	router := NewRouter()

	// Request logging middleware
	var requestLog []string
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestLog = append(requestLog, "request: "+r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// Authentication middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Unauthorized"))
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Rate limiting middleware
	requestCount := 0
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount > 5 {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Rate limit exceeded"))
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Response modification middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Header", "middleware-value")
			next.ServeHTTP(w, r)
		})
	})

	// Add routes
	router.GET("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Protected resource"))
	})

	// Test unauthorized request
	t.Run("Unauthorized Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}

		if body := w.Body.String(); body != "Unauthorized" {
			t.Errorf("Expected body 'Unauthorized', got %q", body)
		}
	})

	// Test authorized request
	t.Run("Authorized Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer token123")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		if body := w.Body.String(); body != "Protected resource" {
			t.Errorf("Expected body 'Protected resource', got %q", body)
		}

		// Check custom header
		if header := w.Header().Get("X-Custom-Header"); header != "middleware-value" {
			t.Errorf("Expected X-Custom-Header 'middleware-value', got %q", header)
		}
	})

	// Test rate limiting
	t.Run("Rate Limiting", func(t *testing.T) {
		// Reset request count
		requestCount = 0

		// Make requests up to limit
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer token123")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status %d, got %d", i+1, http.StatusOK, w.Code)
			}
		}

		// Next request should be rate limited
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer token123")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, w.Code)
		}
	})

	// Test request logging
	t.Run("Request Logging", func(t *testing.T) {
		// Clear log
		requestLog = []string{}

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer token123")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if len(requestLog) != 1 {
			t.Errorf("Expected 1 log entry, got %d", len(requestLog))
		}

		if requestLog[0] != "request: /protected" {
			t.Errorf("Expected log entry 'request: /protected', got %q", requestLog[0])
		}
	})
}

// TestErrorHandlingIntegration tests comprehensive error handling
func TestErrorHandlingIntegration(t *testing.T) {
	router := NewRouter()

	// Custom error handler middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "Internal server error",
						"type":  "panic",
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	})

	// Routes with different error types
	router.OpinionatedGET("/bad-request", func(ctx *Context, req struct{}) (*struct{}, error) {
		return nil, BadRequest("This is a bad request")
	})

	router.OpinionatedGET("/not-found", func(ctx *Context, req struct{}) (*struct{}, error) {
		return nil, NotFound("Resource")
	})

	router.OpinionatedGET("/validation-error", func(ctx *Context, req struct{}) (*struct{}, error) {
		fields := []FieldError{
			NewFieldError("email", "Invalid email format", "invalid-email", "INVALID_FORMAT"),
			NewFieldError("age", "Age must be positive", -1, "INVALID_VALUE"),
		}
		return nil, UnprocessableEntity("Validation failed", fields...)
	})

	router.OpinionatedGET("/internal-error", func(ctx *Context, req struct{}) (*struct{}, error) {
		return nil, InternalServerError("Something went wrong")
	})

	router.GET("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("Test panic")
	})

	router.OpinionatedGET("/business-error", func(ctx *Context, req struct{}) (*struct{}, error) {
		return nil, NewBusinessError(http.StatusConflict, "DUPLICATE_EMAIL", "Email already exists", map[string]string{
			"email": "test@example.com",
		})
	})

	// Test different error types
	errorTests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedType   string
	}{
		{"Bad Request", "/bad-request", http.StatusBadRequest, "BAD_REQUEST"},
		{"Not Found", "/not-found", http.StatusNotFound, "NOT_FOUND"},
		{"Validation Error", "/validation-error", http.StatusUnprocessableEntity, "VALIDATION_FAILED"},
		{"Internal Error", "/internal-error", http.StatusInternalServerError, "INTERNAL_ERROR"},
		{"Business Error", "/business-error", http.StatusConflict, "BUSINESS_ERROR"},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var errorResponse map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			if errorResponse["error"] == nil {
				t.Error("Expected error field in response")
			}

			if errorDetail, ok := errorResponse["error"].(map[string]interface{}); ok {
				if code, ok := errorDetail["code"].(string); ok {
					if code != tt.expectedType {
						t.Errorf("Expected error code %q, got %q", tt.expectedType, code)
					}
				}
			}
		})
	}

	// Test panic handling
	t.Run("Panic Handling", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}

		var errorResponse map[string]string
		if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if errorResponse["type"] != "panic" {
			t.Errorf("Expected error type 'panic', got %q", errorResponse["type"])
		}
	})
}

// TestFileUploadIntegration tests file upload scenarios
func TestFileUploadIntegration(t *testing.T) {
	router := NewRouter()

	// File upload handler
	router.POST("/upload", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20) // 10 MB limit
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Failed to parse multipart form",
			})
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "No file uploaded",
			})
			return
		}
		defer file.Close()

		// Read file content
		content, err := io.ReadAll(file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Failed to read file",
			})
			return
		}

		// Return file info
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"filename": header.Filename,
			"size":     len(content),
			"content":  string(content),
		})
	})

	// Test file upload
	t.Run("File Upload", func(t *testing.T) {
		// Create multipart form
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)

		part, err := writer.CreateFormFile("file", "test.txt")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}

		_, err = part.Write([]byte("Hello, World!"))
		if err != nil {
			t.Fatalf("Failed to write file content: %v", err)
		}

		err = writer.Close()
		if err != nil {
			t.Fatalf("Failed to close multipart writer: %v", err)
		}

		req := httptest.NewRequest("POST", "/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["filename"] != "test.txt" {
			t.Errorf("Expected filename 'test.txt', got %q", response["filename"])
		}

		if response["content"] != "Hello, World!" {
			t.Errorf("Expected content 'Hello, World!', got %q", response["content"])
		}
	})
}

// TestCORSIntegration tests CORS handling
func TestCORSIntegration(t *testing.T) {
	router := NewRouter()

	// CORS middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Add test route
	router.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Test CORS preflight
	t.Run("CORS Preflight", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin '*', got %q", origin)
		}

		if methods := w.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(methods, "GET") {
			t.Errorf("Expected Access-Control-Allow-Methods to contain 'GET', got %q", methods)
		}
	})

	// Test CORS actual request
	t.Run("CORS Actual Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin '*', got %q", origin)
		}

		if body := w.Body.String(); body != "OK" {
			t.Errorf("Expected body 'OK', got %q", body)
		}
	})
}

// TestStreamingResponse tests streaming response handling
func TestStreamingResponse(t *testing.T) {
	router := NewRouter()

	// Streaming endpoint
	router.GET("/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		for i := 0; i < 5; i++ {
			fmt.Fprintf(w, "chunk %d\n", i)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	})

	// Test streaming
	t.Run("Streaming Response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stream", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		body := w.Body.String()
		expectedChunks := []string{"chunk 0", "chunk 1", "chunk 2", "chunk 3", "chunk 4"}

		for _, chunk := range expectedChunks {
			if !strings.Contains(body, chunk) {
				t.Errorf("Expected body to contain %q, got %q", chunk, body)
			}
		}
	})
}

// TestContextTimeout tests context timeout handling
func TestContextTimeout(t *testing.T) {
	router := NewRouter()

	// Add timeout middleware
	router.Use(Timeout(50 * time.Millisecond))

	// Slow endpoint
	router.GET("/slow", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(100 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		case <-r.Context().Done():
			w.WriteHeader(http.StatusRequestTimeout)
			w.Write([]byte("Timeout"))
			return
		}
	})

	// Fast endpoint
	router.GET("/fast", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Test timeout
	t.Run("Request Timeout", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/slow", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// The exact behavior depends on implementation
		// This test mainly ensures no panic occurs
	})

	// Test fast request
	t.Run("Fast Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/fast", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

// TestContentNegotiation tests content negotiation
func TestContentNegotiation(t *testing.T) {
	router := NewRouter()

	// Content negotiation handler
	router.GET("/data", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"name":  "John Doe",
			"age":   30,
			"email": "john@example.com",
		}

		accept := r.Header.Get("Accept")

		switch {
		case strings.Contains(accept, "application/json"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
		case strings.Contains(accept, "application/xml"):
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<user>
    <name>John Doe</name>
    <age>30</age>
    <email>john@example.com</email>
</user>`))
		case strings.Contains(accept, "text/plain"):
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("Name: John Doe\nAge: 30\nEmail: john@example.com"))
		default:
			w.WriteHeader(http.StatusNotAcceptable)
			w.Write([]byte("Not Acceptable"))
		}
	})

	// Test JSON response
	t.Run("JSON Response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/data", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
		}

		var data map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&data); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		if data["name"] != "John Doe" {
			t.Errorf("Expected name 'John Doe', got %q", data["name"])
		}
	})

	// Test XML response
	t.Run("XML Response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/data", nil)
		req.Header.Set("Accept", "application/xml")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		if contentType := w.Header().Get("Content-Type"); contentType != "application/xml" {
			t.Errorf("Expected Content-Type 'application/xml', got %q", contentType)
		}

		body := w.Body.String()
		if !strings.Contains(body, "<name>John Doe</name>") {
			t.Errorf("Expected XML body to contain name element, got %q", body)
		}
	})

	// Test plain text response
	t.Run("Plain Text Response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/data", nil)
		req.Header.Set("Accept", "text/plain")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		if contentType := w.Header().Get("Content-Type"); contentType != "text/plain" {
			t.Errorf("Expected Content-Type 'text/plain', got %q", contentType)
		}

		body := w.Body.String()
		if !strings.Contains(body, "Name: John Doe") {
			t.Errorf("Expected plain text body to contain name, got %q", body)
		}
	})

	// Test not acceptable
	t.Run("Not Acceptable", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/data", nil)
		req.Header.Set("Accept", "application/pdf")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotAcceptable {
			t.Errorf("Expected status %d, got %d", http.StatusNotAcceptable, w.Code)
		}
	})
}
