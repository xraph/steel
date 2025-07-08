package forgerouter

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ExampleTestUsage demonstrates how to use the test utilities
func ExampleTestUsage(t *testing.T) {
	// Create a test router with middleware
	router := NewTestRouter().
		WithMiddleware(Logger).
		WithRecovery().
		WithCORS().
		WithAuth("test-token").
		WithRoute("GET", "/health", MockHandler(http.StatusOK, "OK")).
		WithOpinionatedRoute("GET", "/users/:id", func(ctx *FastContext, req struct {
			ID int `path:"id"`
		}) (*map[string]interface{}, error) {
			user := map[string]interface{}{
				"id":   req.ID,
				"name": fmt.Sprintf("User %d", req.ID),
			}
			return &user, nil
		}).
		WithOpenAPI().
		Build()

	// Example 1: Simple request testing
	t.Run("Simple Request", func(t *testing.T) {
		response := NewRequest("GET", "/health").
			WithAuth("test-token").
			Execute(router)

		AssertResponse(t, response).
			Status(http.StatusOK).
			Body("OK")
	})

	// Example 2: JSON API testing
	t.Run("JSON API", func(t *testing.T) {
		response := NewRequest("GET", "/users/123").
			WithAuth("test-token").
			Execute(router)

		AssertResponse(t, response).
			Status(http.StatusOK).
			IsJSON().
			JSON("id", float64(123)). // JSON numbers are float64
			JSON("name", "User 123")
	})

	// Example 3: Testing with POST body
	t.Run("POST Request", func(t *testing.T) {
		router.OpinionatedPOST("/users", func(ctx *FastContext, req struct {
			Name  string `json:"name" body:"body"`
			Email string `json:"email" body:"body"`
		}) (*map[string]interface{}, error) {
			user := map[string]interface{}{
				"id":    999,
				"name":  req.Name,
				"email": req.Email,
			}
			return &user, nil
		})

		requestBody := map[string]string{
			"name":  "John Doe",
			"email": "john@example.com",
		}

		response := NewRequest("POST", "/users").
			WithAuth("test-token").
			WithJSON(requestBody).
			Execute(router)

		AssertResponse(t, response).
			Status(http.StatusOK).
			IsJSON().
			JSON("id", float64(999)).
			JSON("name", "John Doe").
			JSON("email", "john@example.com")
	})

	// Example 4: Error handling
	t.Run("Error Handling", func(t *testing.T) {
		router.OpinionatedGET("/error", func(ctx *FastContext, req struct{}) (*struct{}, error) {
			return nil, BadRequest("Something went wrong")
		})

		response := NewRequest("GET", "/error").
			WithAuth("test-token").
			Execute(router)

		AssertResponse(t, response).
			Status(http.StatusBadRequest).
			IsJSON().
			JSONExists("error")
	})

	// Example 5: Testing unauthorized access
	t.Run("Unauthorized", func(t *testing.T) {
		response := NewRequest("GET", "/users/123").
			Execute(router) // No auth token

		AssertResponse(t, response).
			Status(http.StatusUnauthorized).
			IsJSON().
			JSON("error", "Unauthorized")
	})

	// Example 6: Testing CORS
	t.Run("CORS", func(t *testing.T) {
		response := NewRequest("OPTIONS", "/users/123").
			WithHeader("Origin", "https://example.com").
			Execute(router)

		AssertResponse(t, response).
			Status(http.StatusOK).
			Header("Access-Control-Allow-Origin", "*").
			HeaderExists("Access-Control-Allow-Methods")
	})
}

// ExampleBenchmarkUsage demonstrates how to use benchmark utilities
func ExampleBenchmarkUsage(b *testing.B) {
	// Create a benchmark setup
	setup := NewBenchmarkSetup().
		AddStaticRoutes(100).
		AddParameterRoutes(50).
		AddRoute("GET", "/complex/:id/nested/:nested", MockHandler(http.StatusOK, "OK"))

	router := setup.Setup()

	// Run benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/route%d", i%100)
		req, _ := NewRequest("GET", path).Build()

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// ExampleLoadTestUsage demonstrates how to use load testing utilities
func ExampleLoadTestUsage(t *testing.T) {
	// Create a router for load testing
	router := NewTestRouter().
		WithRoute("GET", "/fast", MockHandler(http.StatusOK, "OK")).
		WithRoute("GET", "/slow", func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}).
		Build()

	// Configure load test
	config := LoadTestConfig{
		Concurrency: 10,
		Requests:    100,
		Timeout:     5 * time.Second,
		Paths:       []string{"/fast", "/slow"},
	}

	// Run load test
	result := RunLoadTest(router, config)

	// Assert results
	AssertTrue(t, result.SuccessRequests > 0)
	AssertTrue(t, result.FailedRequests == 0)
	AssertTrue(t, result.RequestsPerSecond > 0)

	t.Logf("Load test results:")
	t.Logf("  Total requests: %d", result.TotalRequests)
	t.Logf("  Success requests: %d", result.SuccessRequests)
	t.Logf("  Failed requests: %d", result.FailedRequests)
	t.Logf("  Average latency: %v", result.AverageLatency)
	t.Logf("  Max latency: %v", result.MaxLatency)
	t.Logf("  Min latency: %v", result.MinLatency)
	t.Logf("  Requests per second: %.2f", result.RequestsPerSecond)
	t.Logf("  Duration: %v", result.Duration)
}

// ExampleWebSocketTesting demonstrates WebSocket testing
func ExampleWebSocketTesting(t *testing.T) {
	router := NewRouter()

	// Add WebSocket handler
	router.WebSocket("/ws", func(conn *WSConnection, message struct {
		Text string `json:"text"`
	}) (*struct {
		Echo string `json:"echo"`
	}, error) {
		return &struct {
			Echo string `json:"echo"`
		}{
			Echo: "Echo: " + message.Text,
		}, nil
	})

	// Test WebSocket route registration
	if len(router.wsHandlers) != 1 {
		t.Errorf("Expected 1 WebSocket handler, got %d", len(router.wsHandlers))
	}

	if handler, exists := router.wsHandlers["/ws"]; !exists {
		t.Error("Expected WebSocket handler to be registered")
	} else {
		AssertEqual(t, "/ws", handler.Path)
		AssertNotEqual(t, nil, handler.Handler)
	}
}

// ExampleSSETesting demonstrates SSE testing
func ExampleSSETesting(t *testing.T) {
	router := NewRouter()

	// Add SSE handler
	router.SSE("/events/:userId", func(conn *SSEConnection, params struct {
		UserID int `path:"userId"`
	}) error {
		return conn.SendMessage(SSEMessage{
			Event: "user_event",
			Data:  map[string]interface{}{"user_id": params.UserID},
		})
	})

	// Test SSE route registration
	if len(router.sseHandlers) != 1 {
		t.Errorf("Expected 1 SSE handler, got %d", len(router.sseHandlers))
	}

	if handler, exists := router.sseHandlers["/events/:userId"]; !exists {
		t.Error("Expected SSE handler to be registered")
	} else {
		AssertEqual(t, "/events/:userId", handler.Path)
		AssertNotEqual(t, nil, handler.Handler)
	}
}

// ExampleErrorTesting demonstrates comprehensive error testing
func ExampleErrorTesting(t *testing.T) {
	router := NewRouter()

	// Add error handlers
	router.OpinionatedGET("/bad-request", func(ctx *FastContext, req struct{}) (*struct{}, error) {
		return nil, BadRequest("Bad request error")
	})

	router.OpinionatedGET("/validation-error", func(ctx *FastContext, req struct{}) (*struct{}, error) {
		fields := []FieldError{
			NewFieldError("email", "Invalid email", "invalid", "INVALID_EMAIL"),
		}
		return nil, UnprocessableEntity("Validation failed", fields...)
	})

	router.OpinionatedGET("/custom-error", func(ctx *FastContext, req struct{}) (*struct{}, error) {
		return nil, NewBusinessError(http.StatusConflict, "BUSINESS_RULE", "Business rule violation", nil)
	})

	// Test different error types
	errorTests := []struct {
		name         string
		path         string
		expectedCode int
		expectedType string
	}{
		{"Bad Request", "/bad-request", http.StatusBadRequest, "BAD_REQUEST"},
		{"Validation Error", "/validation-error", http.StatusUnprocessableEntity, "VALIDATION_FAILED"},
		{"Custom Error", "/custom-error", http.StatusConflict, "BUSINESS_ERROR"},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			response := NewRequest("GET", tt.path).Execute(router)

			AssertResponse(t, response).
				Status(tt.expectedCode).
				IsJSON().
				JSONExists("error")

			// Check error structure
			if response.JSON != nil {
				if errorData, ok := response.JSON["error"].(map[string]interface{}); ok {
					if code, ok := errorData["code"].(string); ok {
						AssertEqual(t, tt.expectedType, code)
					}
				}
			}
		})
	}
}

// ExampleAsyncAPITesting demonstrates AsyncAPI testing
func ExampleAsyncAPITesting(t *testing.T) {
	router := NewRouter()

	// Add WebSocket and SSE handlers
	router.WebSocket("/ws/chat", func(conn *WSConnection, message struct {
		Text string `json:"text"`
	}) (*struct {
		Reply string `json:"reply"`
	}, error) {
		return &struct {
			Reply string `json:"reply"`
		}{
			Reply: "Reply: " + message.Text,
		}, nil
	}, WithAsyncSummary("Chat WebSocket"))

	router.SSE("/events", func(conn *SSEConnection, params struct{}) error {
		return conn.SendMessage(SSEMessage{
			Event: "test",
			Data:  "test data",
		})
	}, WithAsyncSummary("Event Stream"))

	// Enable AsyncAPI
	router.EnableAsyncAPI()

	// Test AsyncAPI spec generation
	response := NewRequest("GET", "/asyncapi").Execute(router)

	AssertResponse(t, response).
		Status(http.StatusOK).
		IsJSON().
		JSON("asyncapi", "2.6.0").
		JSONExists("channels")

	// Check that channels are documented
	if response.JSON != nil {
		if channels, ok := response.JSON["channels"].(map[string]interface{}); ok {
			AssertTrue(t, len(channels) >= 2)
		}
	}
}

// ExampleMiddlewareTesting demonstrates middleware testing
func ExampleMiddlewareTesting(t *testing.T) {
	router := NewRouter()

	// Add custom middleware
	var requestCount int
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("X-Request-Count", fmt.Sprintf("%d", requestCount))
			next.ServeHTTP(w, r)
		})
	})

	// Add test route
	router.GET("/test", MockHandler(http.StatusOK, "OK"))

	// Test middleware functionality
	t.Run("Middleware", func(t *testing.T) {
		response := NewRequest("GET", "/test").Execute(router)

		AssertResponse(t, response).
			Status(http.StatusOK).
			Body("OK").
			Header("X-Request-Count", "1")

		// Second request should increment counter
		response2 := NewRequest("GET", "/test").Execute(router)

		AssertResponse(t, response2).
			Status(http.StatusOK).
			Header("X-Request-Count", "2")
	})
}

// ExampleParameterTesting demonstrates parameter testing
func ExampleParameterTesting(t *testing.T) {
	router := NewRouter()

	// Add route with complex parameters
	router.OpinionatedGET("/users/:id", func(ctx *FastContext, req struct {
		ID       int    `path:"id"`
		Include  string `query:"include"`
		Format   string `query:"format"`
		AuthUser string `header:"X-User-ID"`
	}) (*map[string]interface{}, error) {
		return &map[string]interface{}{
			"id":        req.ID,
			"include":   req.Include,
			"format":    req.Format,
			"auth_user": req.AuthUser,
		}, nil
	})

	// Test parameter binding
	response := NewRequest("GET", "/users/123").
		WithQuery("include", "profile").
		WithQuery("format", "json").
		WithHeader("X-User-ID", "admin").
		Execute(router)

	AssertResponse(t, response).
		Status(http.StatusOK).
		IsJSON().
		JSON("id", float64(123)).
		JSON("include", "profile").
		JSON("format", "json").
		JSON("auth_user", "admin")
}

// ExampleConcurrencyTesting demonstrates concurrency testing
func ExampleConcurrencyTesting(t *testing.T) {
	router := NewRouter()

	// Add route
	router.GET("/test", MockHandler(http.StatusOK, "OK"))

	// Test concurrent requests
	const numRequests = 100
	results := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			response := NewRequest("GET", "/test").Execute(router)
			results <- response.StatusCode == http.StatusOK
		}()
	}

	// Wait for all requests to complete
	successCount := 0
	for i := 0; i < numRequests; i++ {
		if <-results {
			successCount++
		}
	}

	AssertEqual(t, numRequests, successCount)
}

// ExampleOpenAPITesting demonstrates OpenAPI testing
func ExampleOpenAPITesting(t *testing.T) {
	router := NewRouter()

	// Add opinionated routes
	router.OpinionatedGET("/users/:id", func(ctx *FastContext, req struct {
		ID int `path:"id"`
	}) (*struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}, error) {
		return &struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}{
			ID:   req.ID,
			Name: fmt.Sprintf("User %d", req.ID),
		}, nil
	}, WithSummary("Get User"), WithDescription("Get user by ID"), WithTags("users"))

	router.OpinionatedPOST("/users", func(ctx *FastContext, req struct {
		Name  string `json:"name" body:"body"`
		Email string `json:"email" body:"body"`
	}) (*struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}, error) {
		return &struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			ID:    999,
			Name:  req.Name,
			Email: req.Email,
		}, nil
	}, WithSummary("Create User"), WithDescription("Create a new user"), WithTags("users"))

	// Enable OpenAPI
	router.EnableOpenAPI()

	// Test OpenAPI spec
	response := NewRequest("GET", "/openapi").Execute(router)

	AssertResponse(t, response).
		Status(http.StatusOK).
		IsJSON().
		JSON("openapi", "3.0.0").
		JSONExists("info").
		JSONExists("paths").
		JSONExists("components")

	// Test OpenAPI documentation endpoints
	docEndpoints := []string{
		"/openapi/docs",
		"/openapi/swagger",
		"/openapi/redoc",
		"/openapi/scalar",
	}

	for _, endpoint := range docEndpoints {
		t.Run(endpoint, func(t *testing.T) {
			response := NewRequest("GET", endpoint).Execute(router)

			AssertResponse(t, response).
				Status(http.StatusOK).
				Header("Content-Type", "text/html")
		})
	}
}

// RunAllExampleTests runs all example tests
func RunAllExampleTests(t *testing.T) {
	t.Run("TestUsage", ExampleTestUsage)
	t.Run("LoadTestUsage", ExampleLoadTestUsage)
	t.Run("WebSocketTesting", ExampleWebSocketTesting)
	t.Run("SSETesting", ExampleSSETesting)
	t.Run("ErrorTesting", ExampleErrorTesting)
	t.Run("AsyncAPITesting", ExampleAsyncAPITesting)
	t.Run("MiddlewareTesting", ExampleMiddlewareTesting)
	t.Run("ParameterTesting", ExampleParameterTesting)
	t.Run("ConcurrencyTesting", ExampleConcurrencyTesting)
	t.Run("OpenAPITesting", ExampleOpenAPITesting)
}

// Additional utility functions for specific test scenarios

// TestTableDriven demonstrates table-driven tests
func TestTableDriven(t *testing.T) {
	router := NewRouter()

	// Add route with validation
	router.OpinionatedPOST("/validate", func(ctx *FastContext, req struct {
		Age int `json:"age" body:"body"`
	}) (*struct {
		Valid bool `json:"valid"`
	}, error) {
		if req.Age < 0 || req.Age > 150 {
			return nil, BadRequest("Invalid age")
		}

		return &struct {
			Valid bool `json:"valid"`
		}{
			Valid: true,
		}, nil
	})

	// Table-driven test
	tests := []struct {
		name         string
		age          int
		expectedCode int
		expectedMsg  string
	}{
		{"Valid Age", 25, http.StatusOK, ""},
		{"Negative Age", -1, http.StatusBadRequest, "Invalid age"},
		{"Too Old", 200, http.StatusBadRequest, "Invalid age"},
		{"Zero Age", 0, http.StatusOK, ""},
		{"Max Age", 150, http.StatusOK, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]int{"age": tt.age}
			response := NewRequest("POST", "/validate").
				WithJSON(body).
				Execute(router)

			AssertResponse(t, response).
				Status(tt.expectedCode)

			if tt.expectedCode == http.StatusOK {
				AssertResponse(t, response).
					IsJSON().
					JSON("valid", true)
			}
		})
	}
}

// TestSubtests demonstrates subtests
func TestSubtests(t *testing.T) {
	router := NewRouter()

	// Add routes for different HTTP methods
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		router.Handle(method, "/test", MockHandler(http.StatusOK, method))
	}

	// Test each method
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			response := NewRequest(method, "/test").Execute(router)

			AssertResponse(t, response).
				Status(http.StatusOK).
				Body(method)
		})
	}
}

// TestCleanup demonstrates test cleanup
func TestCleanup(t *testing.T) {
	router := NewRouter()

	// Add cleanup function
	t.Cleanup(func() {
		// Clean up resources
		t.Log("Cleaning up test resources")
	})

	// Add route
	router.GET("/test", MockHandler(http.StatusOK, "OK"))

	// Test
	response := NewRequest("GET", "/test").Execute(router)
	AssertResponse(t, response).Status(http.StatusOK)
}

// TestHelpers demonstrates helper functions
func TestHelpers(t *testing.T) {
	// Test utility functions
	t.Run("MustMarshalJSON", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		result := MustMarshalJSON(data)
		AssertContains(t, string(result), "key")
		AssertContains(t, string(result), "value")
	})

	t.Run("CreateTestUsers", func(t *testing.T) {
		users := CreateTestUsers(5)
		AssertEqual(t, 5, len(users))
		AssertEqual(t, "User 1", users[0]["name"])
		AssertEqual(t, "user1@example.com", users[0]["email"])
	})

	t.Run("AssertFunctions", func(t *testing.T) {
		AssertEqual(t, 1, 1)
		AssertNotEqual(t, 1, 2)
		AssertTrue(t, true)
		AssertFalse(t, false)
		AssertContains(t, "hello world", "world")
		AssertNotContains(t, "hello world", "test")
	})
}
