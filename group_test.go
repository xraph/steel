package forge_router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test for the new Group functionality
func TestGroupMethods(t *testing.T) {
	router := NewRouter()

	// Test Group() method
	t.Run("Group() method", func(t *testing.T) {
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
