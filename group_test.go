package steel

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test for the new Group functionality
func TestGroupMethods(t *testing.T) {
	// Test Group() method
	t.Run("Group() method", func(t *testing.T) {
		router := NewRouter()
		group := router.Group()
		if group == nil {
			t.Error("Group() should return a non-nil Router")
		}

		// Test that we can add routes to the group
		group.GET("/test", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("test"))
		})

		// Test the route works
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		if w.Body.String() != "test" {
			t.Errorf("Expected body 'test', got %q", w.Body.String())
		}
	})

	// Test GroupFunc() method
	t.Run("GroupFunc() method", func(t *testing.T) {
		router := NewRouter()
		router.GroupFunc(func(r Router) {
			r.GET("/groupfunc", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("groupfunc"))
			})
		})

		// Test the route works
		req := httptest.NewRequest("GET", "/groupfunc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		if w.Body.String() != "groupfunc" {
			t.Errorf("Expected body 'groupfunc', got %q", w.Body.String())
		}
	})

	// Test nested groups
	t.Run("Nested groups", func(t *testing.T) {
		router := NewRouter()
		parentGroup := router.Group()
		childGroup := parentGroup.Group()

		childGroup.GET("/nested", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("nested"))
		})

		// Test the route works
		req := httptest.NewRequest("GET", "/nested", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		if w.Body.String() != "nested" {
			t.Errorf("Expected body 'nested', got %q", w.Body.String())
		}
	})

	// Test middleware inheritance
	t.Run("Middleware inheritance", func(t *testing.T) {
		router := NewRouter()
		middlewareCalled := false

		parentGroup := router.Group()
		parentGroup.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				middlewareCalled = true
				next.ServeHTTP(w, r)
			})
		})

		childGroup := parentGroup.Group()
		childGroup.GET("/middleware-test", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("middleware"))
		})

		// Test the route works and middleware is called
		req := httptest.NewRequest("GET", "/middleware-test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if !middlewareCalled {
			t.Error("Expected middleware to be called")
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	// Test Route() method with prefix
	t.Run("Route() method with prefix", func(t *testing.T) {
		router := NewRouter()

		router.Route("/api/v1", func(r Router) {
			r.GET("/users", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("users"))
			})
		})

		// Test the route works with prefix
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		if w.Body.String() != "users" {
			t.Errorf("Expected body 'users', got %q", w.Body.String())
		}
	})

	// Test multiple middleware layers
	t.Run("Multiple middleware layers", func(t *testing.T) {
		router := NewRouter()
		var callOrder []string

		// Router-level middleware
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callOrder = append(callOrder, "router")
				next.ServeHTTP(w, r)
			})
		})

		// Group-level middleware
		group := router.Group()
		group.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callOrder = append(callOrder, "group")
				next.ServeHTTP(w, r)
			})
		})

		// Nested group middleware
		nestedGroup := group.Group()
		nestedGroup.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callOrder = append(callOrder, "nested")
				next.ServeHTTP(w, r)
			})
		})

		nestedGroup.GET("/test-order", func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "handler")
			w.Write([]byte("ok"))
		})

		// Test middleware execution order
		req := httptest.NewRequest("GET", "/test-order", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		expectedOrder := []string{"router", "group", "nested", "handler"}
		if len(callOrder) != len(expectedOrder) {
			t.Errorf("Expected %d middleware calls, got %d", len(expectedOrder), len(callOrder))
		}

		for i, expected := range expectedOrder {
			if i >= len(callOrder) || callOrder[i] != expected {
				t.Errorf("Expected middleware call %d to be '%s', got '%s'", i, expected, callOrder[i])
			}
		}
	})
}

// Test for opinionated handlers in groups
func TestGroupOpinionatedHandlers(t *testing.T) {
	t.Run("Opinionated handlers in groups", func(t *testing.T) {
		router := NewRouter()

		type TestRequest struct {
			ID int `path:"id"`
		}

		type TestResponse struct {
			ID      int    `json:"id"`
			Message string `json:"message"`
		}

		group := router.Group()
		group.OpinionatedGET("/users/:id", func(ctx *Context, req TestRequest) (*TestResponse, error) {
			return &TestResponse{
				ID:      req.ID,
				Message: "Hello from group",
			}, nil
		})

		// Test the opinionated route works
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Check content type
		if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
		}
	})
}

// Benchmark comparison between Group() and GroupFunc()
func BenchmarkGroupCreation(b *testing.B) {
	router := NewRouter()

	b.Run("Group()", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			group := router.Group()
			group.GET("/test", func(w http.ResponseWriter, r *http.Request) {})
		}
	})

	b.Run("GroupFunc()", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			router.GroupFunc(func(r Router) {
				r.GET("/test", func(w http.ResponseWriter, r *http.Request) {})
			})
		}
	})
}
