package steel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestServer wraps httptest.Server with router-specific functionality
type TestServer struct {
	*httptest.Server
	Router *SteelRouter
}

// NewTestServer creates a new test server with the given router
func NewTestServer(router *SteelRouter) *TestServer {
	server := httptest.NewServer(router)
	return &TestServer{
		Server: server,
		Router: router,
	}
}

// NewTestServerTLS creates a new TLS test server with the given router
func NewTestServerTLS(router *SteelRouter) *TestServer {
	server := httptest.NewTLSServer(router)
	return &TestServer{
		Server: server,
		Router: router,
	}
}

// TestRequest represents a test HTTP request
type TestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// TestResponse represents a test HTTP response
type TestResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
	JSON       map[string]interface{}
}

// RequestBuilder helps build test requests
type RequestBuilder struct {
	request TestRequest
}

// NewRequest creates a new request builder
func NewRequest(method, path string) *RequestBuilder {
	return &RequestBuilder{
		request: TestRequest{
			Method:      method,
			Path:        path,
			Headers:     make(map[string]string),
			QueryParams: make(map[string]string),
		},
	}
}

// WithBody sets the request body
func (rb *RequestBuilder) WithBody(body interface{}) *RequestBuilder {
	rb.request.Body = body
	return rb
}

// WithHeader adds a header to the request
func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	rb.request.Headers[key] = value
	return rb
}

// WithQuery adds a query parameter to the request
func (rb *RequestBuilder) WithQuery(key, value string) *RequestBuilder {
	rb.request.QueryParams[key] = value
	return rb
}

// WithJSON sets the request body as JSON and adds Content-Type header
func (rb *RequestBuilder) WithJSON(body interface{}) *RequestBuilder {
	rb.request.Body = body
	rb.request.Headers["Content-Type"] = "application/json"
	return rb
}

// WithAuth adds Authorization header
func (rb *RequestBuilder) WithAuth(token string) *RequestBuilder {
	rb.request.Headers["Authorization"] = "Bearer " + token
	return rb
}

// Build creates an http.Request from the builder
func (rb *RequestBuilder) Build() (*http.Request, error) {
	// Build URL with query parameters
	url := rb.request.Path
	if len(rb.request.QueryParams) > 0 {
		params := make([]string, 0, len(rb.request.QueryParams))
		for k, v := range rb.request.QueryParams {
			params = append(params, fmt.Sprintf("%s=%s", k, v))
		}
		url += "?" + strings.Join(params, "&")
	}

	// Create request body
	var body io.Reader
	if rb.request.Body != nil {
		switch v := rb.request.Body.(type) {
		case string:
			body = strings.NewReader(v)
		case []byte:
			body = bytes.NewReader(v)
		default:
			// Assume JSON
			data, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %v", err)
			}
			body = bytes.NewReader(data)
		}
	}

	// Create request
	req, err := http.NewRequest(rb.request.Method, url, body)
	if err != nil {
		return nil, err
	}

	// Add headers
	for k, v := range rb.request.Headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

// Execute executes the request against the given router
func (rb *RequestBuilder) Execute(router *SteelRouter) *TestResponse {
	req, err := rb.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build request: %v", err))
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	response := &TestResponse{
		StatusCode: w.Code,
		Body:       w.Body.String(),
		Headers:    make(map[string]string),
	}

	// Copy headers
	for k, v := range w.Header() {
		if len(v) > 0 {
			response.Headers[k] = v[0]
		}
	}

	// Try to parse JSON
	if strings.Contains(response.Headers["Content-Type"], "application/json") {
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(response.Body), &jsonData); err == nil {
			response.JSON = jsonData
		}
	}

	return response
}

// ResponseAssertion helps assert response properties
type ResponseAssertion struct {
	t        *testing.T
	response *TestResponse
}

// AssertResponse creates a new response assertion
func AssertResponse(t *testing.T, response *TestResponse) *ResponseAssertion {
	return &ResponseAssertion{
		t:        t,
		response: response,
	}
}

// Status asserts the response status code
func (ra *ResponseAssertion) Status(expected int) *ResponseAssertion {
	ra.t.Helper()
	if ra.response.StatusCode != expected {
		ra.t.Errorf("Expected status %d, got %d", expected, ra.response.StatusCode)
	}
	return ra
}

// Body asserts the response body
func (ra *ResponseAssertion) Body(expected string) *ResponseAssertion {
	ra.t.Helper()
	if ra.response.Body != expected {
		ra.t.Errorf("Expected body %q, got %q", expected, ra.response.Body)
	}
	return ra
}

// BodyContains asserts the response body contains a substring
func (ra *ResponseAssertion) BodyContains(expected string) *ResponseAssertion {
	ra.t.Helper()
	if !strings.Contains(ra.response.Body, expected) {
		ra.t.Errorf("Expected body to contain %q, got %q", expected, ra.response.Body)
	}
	return ra
}

// Header asserts a response header value
func (ra *ResponseAssertion) Header(key, expected string) *ResponseAssertion {
	ra.t.Helper()
	if actual := ra.response.Headers[key]; actual != expected {
		ra.t.Errorf("Expected header %s to be %q, got %q", key, expected, actual)
	}
	return ra
}

// HeaderExists asserts a response header exists
func (ra *ResponseAssertion) HeaderExists(key string) *ResponseAssertion {
	ra.t.Helper()
	if _, exists := ra.response.Headers[key]; !exists {
		ra.t.Errorf("Expected header %s to exist", key)
	}
	return ra
}

// JSON asserts a JSON field value
func (ra *ResponseAssertion) JSON(path string, expected interface{}) *ResponseAssertion {
	ra.t.Helper()
	if ra.response.JSON == nil {
		ra.t.Error("Response is not JSON")
		return ra
	}

	actual, exists := ra.response.JSON[path]
	if !exists {
		ra.t.Errorf("JSON field %q does not exist", path)
		return ra
	}

	if !reflect.DeepEqual(actual, expected) {
		ra.t.Errorf("Expected JSON field %q to be %v, got %v", path, expected, actual)
	}

	return ra
}

// JSONExists asserts a JSON field exists
func (ra *ResponseAssertion) JSONExists(path string) *ResponseAssertion {
	ra.t.Helper()
	if ra.response.JSON == nil {
		ra.t.Error("Response is not JSON")
		return ra
	}

	if _, exists := ra.response.JSON[path]; !exists {
		ra.t.Errorf("JSON field %q does not exist", path)
	}

	return ra
}

// IsJSON asserts the response is valid JSON
func (ra *ResponseAssertion) IsJSON() *ResponseAssertion {
	ra.t.Helper()
	if ra.response.JSON == nil {
		ra.t.Error("Response is not valid JSON")
	}
	return ra
}

// MockHandler creates a mock handler for testing
func MockHandler(statusCode int, body string) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}
}

// MockJSONHandler creates a mock JSON handler
func MockJSONHandler(statusCode int, data interface{}) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(data)
	}
}

// MockErrorHandler creates a mock error handler
func MockErrorHandler(err error) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if apiErr, ok := err.(APIError); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(apiErr.StatusCode())
			json.NewEncoder(w).Encode(apiErr.ToResponse())
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

// BenchmarkSetup helps set up benchmarks
type BenchmarkSetup struct {
	Router *SteelRouter
	Routes []BenchmarkRoute
}

// BenchmarkRoute represents a route for benchmarking
type BenchmarkRoute struct {
	Method  string
	Path    string
	Handler HandlerFunc
}

// NewBenchmarkSetup creates a new benchmark setup
func NewBenchmarkSetup() *BenchmarkSetup {
	return &BenchmarkSetup{
		Router: NewRouter(),
		Routes: make([]BenchmarkRoute, 0),
	}
}

// AddRoute adds a route to the benchmark setup
func (bs *BenchmarkSetup) AddRoute(method, path string, handler HandlerFunc) *BenchmarkSetup {
	bs.Routes = append(bs.Routes, BenchmarkRoute{
		Method:  method,
		Path:    path,
		Handler: handler,
	})
	return bs
}

// AddStaticRoutes adds multiple static routes for benchmarking
func (bs *BenchmarkSetup) AddStaticRoutes(count int) *BenchmarkSetup {
	handler := MockHandler(http.StatusOK, "OK")
	for i := 0; i < count; i++ {
		path := fmt.Sprintf("/route%d", i)
		bs.AddRoute("GET", path, handler)
	}
	return bs
}

// AddParameterRoutes adds multiple parameter routes for benchmarking
func (bs *BenchmarkSetup) AddParameterRoutes(count int) *BenchmarkSetup {
	handler := MockHandler(http.StatusOK, "OK")
	for i := 0; i < count; i++ {
		path := fmt.Sprintf("/param%d/:id", i)
		bs.AddRoute("GET", path, handler)
	}
	return bs
}

// Setup registers all routes with the router
func (bs *BenchmarkSetup) Setup() *SteelRouter {
	for _, route := range bs.Routes {
		bs.Router.Handle(route.Method, route.Path, route.Handler)
	}
	return bs.Router
}

// LoadTestConfig represents configuration for load testing
type LoadTestConfig struct {
	Concurrency int
	Requests    int
	Timeout     time.Duration
	Paths       []string
}

// LoadTestResult represents the result of a load test
type LoadTestResult struct {
	TotalRequests     int
	SuccessRequests   int
	FailedRequests    int
	AverageLatency    time.Duration
	MaxLatency        time.Duration
	MinLatency        time.Duration
	RequestsPerSecond float64
	Duration          time.Duration
}

// RunLoadTest runs a simple load test against the router
func RunLoadTest(router *SteelRouter, config LoadTestConfig) LoadTestResult {
	start := time.Now()
	results := make(chan time.Duration, config.Requests)
	errors := make(chan error, config.Requests)

	// Create a semaphore to limit concurrency
	sem := make(chan struct{}, config.Concurrency)

	// Launch requests
	for i := 0; i < config.Requests; i++ {
		go func(i int) {
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			path := config.Paths[i%len(config.Paths)]

			reqStart := time.Now()
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			latency := time.Since(reqStart)

			if w.Code >= 400 {
				errors <- fmt.Errorf("HTTP %d", w.Code)
			} else {
				results <- latency
			}
		}(i)
	}

	// Collect results
	var totalLatency time.Duration
	var minLatency, maxLatency time.Duration
	successCount := 0
	errorCount := 0

	for i := 0; i < config.Requests; i++ {
		select {
		case latency := <-results:
			successCount++
			totalLatency += latency
			if minLatency == 0 || latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}
		case <-errors:
			errorCount++
		case <-time.After(config.Timeout):
			errorCount++
		}
	}

	totalDuration := time.Since(start)

	var averageLatency time.Duration
	if successCount > 0 {
		averageLatency = totalLatency / time.Duration(successCount)
	}

	rps := float64(successCount) / totalDuration.Seconds()

	return LoadTestResult{
		TotalRequests:     config.Requests,
		SuccessRequests:   successCount,
		FailedRequests:    errorCount,
		AverageLatency:    averageLatency,
		MaxLatency:        maxLatency,
		MinLatency:        minLatency,
		RequestsPerSecond: rps,
		Duration:          totalDuration,
	}
}

// TestRouterBuilder helps build routers for testing
type TestRouterBuilder struct {
	router *SteelRouter
}

// NewTestRouter creates a new test router builder
func NewTestRouter() *TestRouterBuilder {
	return &TestRouterBuilder{
		router: NewRouter(),
	}
}

// WithMiddleware adds middleware to the router
func (trb *TestRouterBuilder) WithMiddleware(middleware ...MiddlewareFunc) *TestRouterBuilder {
	trb.router.Use(middleware...)
	return trb
}

// WithRoute adds a route to the router
func (trb *TestRouterBuilder) WithRoute(method, path string, handler HandlerFunc) *TestRouterBuilder {
	trb.router.Handle(method, path, handler)
	return trb
}

// WithOpinionatedRoute adds an opinionated route to the router
func (trb *TestRouterBuilder) WithOpinionatedRoute(method, path string, handler interface{}, opts ...HandlerOption) *TestRouterBuilder {
	switch method {
	case "GET":
		trb.router.OpinionatedGET(path, handler, opts...)
	case "POST":
		trb.router.OpinionatedPOST(path, handler, opts...)
	case "PUT":
		trb.router.OpinionatedPUT(path, handler, opts...)
	case "DELETE":
		trb.router.OpinionatedDELETE(path, handler, opts...)
	case "PATCH":
		trb.router.OpinionatedPATCH(path, handler, opts...)
	}
	return trb
}

// WithCORS adds CORS middleware
func (trb *TestRouterBuilder) WithCORS() *TestRouterBuilder {
	trb.router.Use(func(next http.Handler) http.Handler {
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
	return trb
}

// WithAuth adds authentication middleware
func (trb *TestRouterBuilder) WithAuth(validToken string) *TestRouterBuilder {
	trb.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+validToken {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Unauthorized",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	return trb
}

// WithLogging adds logging middleware
func (trb *TestRouterBuilder) WithLogging() *TestRouterBuilder {
	trb.router.Use(Logger)
	return trb
}

// WithTimeout adds timeout middleware
func (trb *TestRouterBuilder) WithTimeout(timeout time.Duration) *TestRouterBuilder {
	trb.router.Use(Timeout(timeout))
	return trb
}

// WithRecovery adds recovery middleware
func (trb *TestRouterBuilder) WithRecovery() *TestRouterBuilder {
	trb.router.Use(Recoverer)
	return trb
}

// WithOpenAPI enables OpenAPI documentation
func (trb *TestRouterBuilder) WithOpenAPI() *TestRouterBuilder {
	trb.router.EnableOpenAPI()
	return trb
}

// Build returns the configured router
func (trb *TestRouterBuilder) Build() *SteelRouter {
	return trb.router
}

// Utility functions for testing

// MustMarshalJSON marshals data to JSON or panics
func MustMarshalJSON(data interface{}) []byte {
	result, err := json.Marshal(data)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}
	return result
}

// MustUnmarshalJSON unmarshals JSON or panics
func MustUnmarshalJSON(data []byte, v interface{}) {
	err := json.Unmarshal(data, v)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal JSON: %v", err))
	}
}

// CreateTestUsers creates test users for testing
func CreateTestUsers(count int) []map[string]interface{} {
	users := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		users[i] = map[string]interface{}{
			"id":    i + 1,
			"name":  fmt.Sprintf("User %d", i+1),
			"email": fmt.Sprintf("user%d@example.com", i+1),
			"age":   20 + (i % 50),
		}
	}
	return users
}

// AssertNoError fails the test if error is not nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// AssertError fails the test if error is nil
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// AssertEqual fails the test if values are not equal
func AssertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotEqual fails the test if values are equal
func AssertNotEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected values to be different, but both were %v", expected)
	}
}

// AssertTrue fails the test if condition is false
func AssertTrue(t *testing.T, condition bool) {
	t.Helper()
	if !condition {
		t.Fatal("Expected condition to be true")
	}
}

// AssertFalse fails the test if condition is true
func AssertFalse(t *testing.T, condition bool) {
	t.Helper()
	if condition {
		t.Fatal("Expected condition to be false")
	}
}

// AssertContains fails the test if string doesn't contain substring
func AssertContains(t *testing.T, str, substr string) {
	t.Helper()
	if !strings.Contains(str, substr) {
		t.Fatalf("Expected %q to contain %q", str, substr)
	}
}

// AssertNotContains fails the test if string contains substring
func AssertNotContains(t *testing.T, str, substr string) {
	t.Helper()
	if strings.Contains(str, substr) {
		t.Fatalf("Expected %q to not contain %q", str, substr)
	}
}

// AssertPanic fails the test if function doesn't panic
func AssertPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected function to panic")
		}
	}()
	fn()
}

// AssertNoPanic fails the test if function panics
func AssertNoPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Expected function to not panic, but it panicked with: %v", r)
		}
	}()
	fn()
}
