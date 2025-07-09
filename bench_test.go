package forgerouter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Benchmark data structures
type BenchmarkRequest struct {
	ID   int           `path:"id"`
	Name string        `query:"name"`
	Body BenchmarkBody `body:"body"`
}

type BenchmarkBody struct {
	Content string `json:"content"`
	Count   int    `json:"count"`
}

type BenchmarkResponse struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Count   int    `json:"count"`
}

// BenchmarkStaticRoutes benchmarks static route performance
func BenchmarkStaticRoutes(b *testing.B) {
	router := NewRouter()

	// Add static routes
	routes := []string{
		"/",
		"/about",
		"/contact",
		"/products",
		"/services",
		"/blog",
		"/api/v1/health",
		"/api/v1/status",
		"/api/v1/version",
		"/api/v1/metrics",
	}

	for _, route := range routes {
		router.GET(route, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		route := routes[i%len(routes)]
		req := httptest.NewRequest("GET", route, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkParameterRoutes benchmarks parameter route performance
func BenchmarkParameterRoutes(b *testing.B) {
	router := NewRouter()

	// Add parameter routes
	routes := []string{
		"/users/:id",
		"/users/:id/posts",
		"/users/:id/posts/:postId",
		"/users/:id/posts/:postId/comments",
		"/users/:id/posts/:postId/comments/:commentId",
		"/api/v1/users/:userId",
		"/api/v1/users/:userId/profile",
		"/api/v1/users/:userId/settings",
		"/api/v1/users/:userId/notifications",
		"/api/v1/users/:userId/subscriptions",
	}

	for _, route := range routes {
		router.GET(route, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
	}

	// Test paths
	testPaths := []string{
		"/users/123",
		"/users/123/posts",
		"/users/123/posts/456",
		"/users/123/posts/456/comments",
		"/users/123/posts/456/comments/789",
		"/api/v1/users/123",
		"/api/v1/users/123/profile",
		"/api/v1/users/123/settings",
		"/api/v1/users/123/notifications",
		"/api/v1/users/123/subscriptions",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := testPaths[i%len(testPaths)]
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkWildcardRoutes benchmarks wildcard route performance
func BenchmarkWildcardRoutes(b *testing.B) {
	router := NewRouter()

	// Add wildcard routes
	router.GET("/static/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Static file"))
	})

	router.GET("/files/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("File"))
	})

	router.GET("/assets/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Asset"))
	})

	// Test paths
	testPaths := []string{
		"/static/css/style.css",
		"/static/js/app.js",
		"/static/images/logo.png",
		"/files/documents/report.pdf",
		"/files/uploads/image.jpg",
		"/assets/fonts/font.woff",
		"/assets/icons/icon.svg",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := testPaths[i%len(testPaths)]
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkMixedRoutes benchmarks mixed route types
func BenchmarkMixedRoutes(b *testing.B) {
	router := NewRouter()

	// Add mixed routes
	router.GET("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Home"))
	})

	router.GET("/about", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("About"))
	})

	router.GET("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("User"))
	})

	router.GET("/users/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Post"))
	})

	router.GET("/static/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Static"))
	})

	// Test paths
	testPaths := []string{
		"/",
		"/about",
		"/users/123",
		"/users/123/posts/456",
		"/static/file.txt",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := testPaths[i%len(testPaths)]
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkOpinionatedHandlers benchmarks opinionated handler performance
func BenchmarkOpinionatedHandlers(b *testing.B) {
	router := NewRouter()

	// Add opinionated handlers
	router.OpinionatedGET("/users/:id", func(ctx *ForgeContext, req BenchmarkRequest) (*BenchmarkResponse, error) {
		return &BenchmarkResponse{
			ID:      req.ID,
			Name:    req.Name,
			Content: req.Body.Content,
			Count:   req.Body.Count,
		}, nil
	})

	router.OpinionatedPOST("/users", func(ctx *ForgeContext, req BenchmarkRequest) (*BenchmarkResponse, error) {
		return &BenchmarkResponse{
			ID:      999,
			Name:    req.Name,
			Content: req.Body.Content,
			Count:   req.Body.Count,
		}, nil
	})

	// Prepare request body
	body := BenchmarkBody{
		Content: "Test content",
		Count:   42,
	}
	bodyBytes, _ := json.Marshal(body)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			// GET request
			req := httptest.NewRequest("GET", "/users/123?name=John", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}
		} else {
			// POST request
			req := httptest.NewRequest("POST", "/users?name=Jane", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}
		}
	}
}

// BenchmarkMiddleware benchmarks middleware performance
func BenchmarkMiddleware(b *testing.B) {
	router := NewRouter()

	// Add middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "middleware")
			next.ServeHTTP(w, r)
		})
	})

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			duration := time.Since(start)
			w.Header().Set("X-Duration", duration.String())
		})
	})

	// Add routes
	router.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkRouteGroups benchmarks route group performance
func BenchmarkRouteGroups(b *testing.B) {
	router := NewRouter()

	// Add route groups
	router.Route("/api", func(r Router) {
		r.Route("/v1", func(r Router) {
			r.GET("/users", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Users"))
			})

			r.GET("/posts", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Posts"))
			})
		})

		r.Route("/v2", func(r Router) {
			r.GET("/users", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Users v2"))
			})

			r.GET("/posts", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Posts v2"))
			})
		})
	})

	// Test paths
	testPaths := []string{
		"/api/v1/users",
		"/api/v1/posts",
		"/api/v2/users",
		"/api/v2/posts",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := testPaths[i%len(testPaths)]
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkParameterExtraction benchmarks parameter extraction performance
func BenchmarkParameterExtraction(b *testing.B) {
	router := NewRouter()

	router.GET("/users/:id/posts/:postId/comments/:commentId", func(w http.ResponseWriter, r *http.Request) {
		params := ParamsFromContext(r.Context())
		userID := params.Get("id")
		postID := params.Get("postId")
		commentID := params.Get("commentId")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("User: %s, Post: %s, Comment: %s", userID, postID, commentID)))
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/users/%d/posts/%d/comments/%d", i, i*2, i*3)
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkJSONHandling benchmarks JSON request/response handling
func BenchmarkJSONHandling(b *testing.B) {
	router := NewRouter()

	router.OpinionatedPOST("/data", func(ctx *ForgeContext, req BenchmarkRequest) (*BenchmarkResponse, error) {
		return &BenchmarkResponse{
			ID:      req.ID,
			Name:    req.Name,
			Content: req.Body.Content,
			Count:   req.Body.Count,
		}, nil
	})

	// Prepare request body
	body := BenchmarkBody{
		Content: "Test content for benchmarking JSON handling performance",
		Count:   12345,
	}
	bodyBytes, _ := json.Marshal(body)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/data?name=TestUser", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkErrorHandling benchmarks error handling performance
func BenchmarkErrorHandling(b *testing.B) {
	router := NewRouter()

	router.OpinionatedGET("/error", func(ctx *ForgeContext, req struct{}) (*struct{}, error) {
		return nil, BadRequest("Test error for benchmarking")
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/error", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			b.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	}
}

// BenchmarkConcurrentRequests benchmarks concurrent request handling
func BenchmarkConcurrentRequests(b *testing.B) {
	router := NewRouter()

	router.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}
		}
	})
}

// BenchmarkLargeRouteTree benchmarks performance with large route trees
func BenchmarkLargeRouteTree(b *testing.B) {
	router := NewRouter()

	// Add many routes
	for i := 0; i < 1000; i++ {
		path := fmt.Sprintf("/route%d", i)
		router.GET(path, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
	}

	// Add parameter routes
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("/param%d/:id", i)
		router.GET(path, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var path string
		if i%10 == 0 {
			// Parameter route
			path = fmt.Sprintf("/param%d/123", i%100)
		} else {
			// Static route
			path = fmt.Sprintf("/route%d", i%1000)
		}

		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	router := NewRouter()

	router.GET("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		params := ParamsFromContext(r.Context())
		id := params.Get("id")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(id))
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkTypeConversion benchmarks type conversion in parameter binding
func BenchmarkTypeConversion(b *testing.B) {
	router := NewRouter()

	type TypeConversionRequest struct {
		ID     int     `path:"id"`
		Count  int     `query:"count"`
		Rate   float64 `query:"rate"`
		Active bool    `query:"active"`
	}

	router.OpinionatedGET("/convert/:id", func(ctx *ForgeContext, req TypeConversionRequest) (*TypeConversionRequest, error) {
		return &req, nil
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/convert/%d?count=%d&rate=%.2f&active=%t", i, i*2, float64(i)*1.5, i%2 == 0)
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkComplexRouting benchmarks complex routing scenarios
func BenchmarkComplexRouting(b *testing.B) {
	router := NewRouter()

	// Add complex routing structure
	router.Route("/api", func(r Router) {
		r.Route("/v1", func(r Router) {
			r.GET("/users", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Users"))
			})

			r.GET("/users/:id", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("User"))
			})

			r.Route("/users/:id", func(r Router) {
				r.GET("/posts", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("User posts"))
				})

				r.GET("/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("User post"))
				})

				r.GET("/posts/:postId/comments", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("Post comments"))
				})
			})
		})

		r.GET("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
	})

	router.GET("/static/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Static"))
	})

	// Test paths
	testPaths := []string{
		"/api/v1/users",
		"/api/v1/users/123",
		"/api/v1/users/123/posts",
		"/api/v1/users/123/posts/456",
		"/api/v1/users/123/posts/456/comments",
		"/api/health",
		"/static/css/style.css",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := testPaths[i%len(testPaths)]
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkRequestParsing benchmarks request parsing performance
func BenchmarkRequestParsing(b *testing.B) {
	router := NewRouter()

	type ParseRequest struct {
		ID      int    `path:"id"`
		Name    string `query:"name"`
		Email   string `query:"email"`
		Age     int    `query:"age"`
		Active  bool   `query:"active"`
		Headers string `header:"Authorization"`
	}

	router.OpinionatedGET("/parse/:id", func(ctx *ForgeContext, req ParseRequest) (*ParseRequest, error) {
		return &req, nil
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/parse/%d?name=User%d&email=user%d@example.com&age=%d&active=%t",
			i, i, i, 20+i%50, i%2 == 0)
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer token123")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkResponseSerialization benchmarks response serialization performance
func BenchmarkResponseSerialization(b *testing.B) {
	router := NewRouter()

	type LargeResponse struct {
		ID          int                    `json:"id"`
		Name        string                 `json:"name"`
		Email       string                 `json:"email"`
		Tags        []string               `json:"tags"`
		Metadata    map[string]interface{} `json:"metadata"`
		Items       []map[string]string    `json:"items"`
		Timestamp   int64                  `json:"timestamp"`
		Description string                 `json:"description"`
	}

	router.OpinionatedGET("/large", func(ctx *ForgeContext, req struct{}) (*LargeResponse, error) {
		return &LargeResponse{
			ID:          12345,
			Name:        "Test User",
			Email:       "test@example.com",
			Tags:        []string{"tag1", "tag2", "tag3", "tag4", "tag5"},
			Metadata:    map[string]interface{}{"key1": "value1", "key2": 42, "key3": true},
			Items:       []map[string]string{{"item1": "value1"}, {"item2": "value2"}},
			Timestamp:   time.Now().Unix(),
			Description: "This is a long description field that contains a lot of text to simulate a real-world response with substantial content that needs to be serialized efficiently.",
		}, nil
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/large", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkTrailingSlashRedirect benchmarks trailing slash redirect performance
func BenchmarkTrailingSlashRedirect(b *testing.B) {
	router := NewRouter()

	router.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Half requests with trailing slash, half without
		path := "/test"
		if i%2 == 0 {
			path = "/test/"
		}

		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should either be OK or redirect
		if w.Code != http.StatusOK && w.Code != http.StatusMovedPermanently {
			b.Errorf("Expected status %d or %d, got %d", http.StatusOK, http.StatusMovedPermanently, w.Code)
		}
	}
}

// BenchmarkNotFoundHandling benchmarks 404 handling performance
func BenchmarkNotFoundHandling(b *testing.B) {
	router := NewRouter()

	// Add some routes
	router.GET("/exists", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/notfound%d", i)
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			b.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	}
}

// BenchmarkPool benchmarks parameter pool performance
func BenchmarkPool(b *testing.B) {
	router := NewRouter()

	router.GET("/users/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/users/%d/posts/%d", i, i*2)
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkComplexParameters benchmarks complex parameter scenarios
func BenchmarkComplexParameters(b *testing.B) {
	router := NewRouter()

	type ComplexParams struct {
		UserID    int    `path:"userId"`
		PostID    int    `path:"postId"`
		CommentID int    `path:"commentId"`
		Limit     int    `query:"limit"`
		Offset    int    `query:"offset"`
		Sort      string `query:"sort"`
		Filter    string `query:"filter"`
		Include   string `query:"include"`
		Format    string `query:"format"`
		Version   string `header:"API-Version"`
		Auth      string `header:"Authorization"`
		UserAgent string `header:"User-Agent"`
	}

	router.OpinionatedGET("/users/:userId/posts/:postId/comments/:commentId",
		func(ctx *ForgeContext, req ComplexParams) (*ComplexParams, error) {
			return &req, nil
		})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/users/%d/posts/%d/comments/%d?limit=%d&offset=%d&sort=created&filter=active&include=user&format=json",
			i, i*2, i*3, 10, i*10)
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("API-Version", "v1")
		req.Header.Set("Authorization", "Bearer token123")
		req.Header.Set("User-Agent", "BenchmarkClient/1.0")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// BenchmarkHTTPMethods benchmarks different HTTP methods
func BenchmarkHTTPMethods(b *testing.B) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}

	router.GET("/test", handler)
	router.POST("/test", handler)
	router.PUT("/test", handler)
	router.DELETE("/test", handler)
	router.PATCH("/test", handler)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		method := methods[i%len(methods)]
		req := httptest.NewRequest(method, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// Comparative benchmarks against standard library
func BenchmarkStdLibRouter(b *testing.B) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("About"))
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := "/"
		if i%2 == 0 {
			path = "/about"
		}

		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

func BenchmarkForgeRouterVsStdLib(b *testing.B) {
	forgeRouter := NewRouter()
	forgeRouter.GET("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	forgeRouter.GET("/about", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("About"))
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := "/"
		if i%2 == 0 {
			path = "/about"
		}

		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		forgeRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}
