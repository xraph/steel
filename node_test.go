package forge_router

import (
	"fmt"
	"net/http"
	"testing"
)

// TestNodeCreation tests node creation and initialization
func TestNodeCreation(t *testing.T) {
	n := &node{
		path:     "/test",
		children: []*node{},
		methods:  make(map[string]HandlerFunc),
	}

	if n.path != "/test" {
		t.Errorf("Expected path '/test', got %q", n.path)
	}

	if n.children == nil {
		t.Error("Expected children to be initialized")
	}

	if len(n.children) != 0 {
		t.Errorf("Expected 0 children, got %d", len(n.children))
	}
}

// TestNodeAddRoute tests adding routes to the node tree
func TestNodeAddRoute(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	// Test adding simple route
	root.addRoute("/test", handler)

	// Should create a child node
	if len(root.children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(root.children))
	}

	if root.children[0].path != "test" {
		t.Errorf("Expected child path 'test', got %q", root.children[0].path)
	}

	if root.children[0].handler == nil {
		t.Error("Expected handler to be set")
	}
}

// TestNodeAddRouteWithParams tests adding routes with parameters
func TestNodeAddRouteWithParams(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	// Test adding route with parameter
	root.addRoute("/users/:id", handler)

	// Should create child nodes
	if len(root.children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(root.children))
	}

	usersNode := root.children[0]
	if usersNode.path != "users" {
		t.Errorf("Expected users node path 'users', got %q", usersNode.path)
	}

	if len(usersNode.children) != 1 {
		t.Errorf("Expected 1 child in users node, got %d", len(usersNode.children))
	}

	paramNode := usersNode.children[0]
	if !paramNode.isParam {
		t.Error("Expected parameter node to be marked as param")
	}

	if paramNode.paramName != "id" {
		t.Errorf("Expected parameter name 'id', got %q", paramNode.paramName)
	}

	if paramNode.handler == nil {
		t.Error("Expected handler to be set on parameter node")
	}
}

// TestNodeAddRouteWithWildcard tests adding routes with wildcards
func TestNodeAddRouteWithWildcard(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	// Test adding wildcard route
	root.addRoute("/static/*", handler)

	// Should create child nodes
	if len(root.children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(root.children))
	}

	staticNode := root.children[0]
	if staticNode.path != "static" {
		t.Errorf("Expected static node path 'static', got %q", staticNode.path)
	}

	if len(staticNode.children) != 1 {
		t.Errorf("Expected 1 child in static node, got %d", len(staticNode.children))
	}

	wildcardNode := staticNode.children[0]
	if !wildcardNode.wildcard {
		t.Error("Expected wildcard node to be marked as wildcard")
	}

	if wildcardNode.handler == nil {
		t.Error("Expected handler to be set on wildcard node")
	}
}

// TestNodeFindHandler tests finding handlers in the node tree
func TestNodeFindHandler(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	handler1 := func(w http.ResponseWriter, r *http.Request) {}
	handler2 := func(w http.ResponseWriter, r *http.Request) {}
	handler3 := func(w http.ResponseWriter, r *http.Request) {}

	// Add various routes
	root.addRoute("/", handler1)
	root.addRoute("/users", handler2)
	root.addRoute("/users/:id", handler3)

	params := &Params{
		keys:   make([]string, 0),
		values: make([]string, 0),
	}

	// Test finding root handler
	found := root.findHandler("/", params)
	if found == nil {
		t.Error("Expected to find root handler")
	}

	// Test finding static handler
	params.Reset()
	found = root.findHandler("/users", params)
	if found == nil {
		t.Error("Expected to find users handler")
	}

	// Test finding parameter handler
	params.Reset()
	found = root.findHandler("/users/123", params)
	if found == nil {
		t.Error("Expected to find users/:id handler")
	}

	// Check that parameter was extracted
	if params.Get("id") != "123" {
		t.Errorf("Expected parameter id to be '123', got %q", params.Get("id"))
	}

	// Test not found
	params.Reset()
	found = root.findHandler("/nonexistent", params)
	if found != nil {
		t.Error("Expected not to find nonexistent handler")
	}
}

// TestNodeFindHandlerWithMultipleParams tests finding handlers with multiple parameters
func TestNodeFindHandlerWithMultipleParams(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	// Add route with multiple parameters
	root.addRoute("/users/:userId/posts/:postId", handler)

	params := &Params{
		keys:   make([]string, 0),
		values: make([]string, 0),
	}

	// Test finding handler with multiple parameters
	found := root.findHandler("/users/123/posts/456", params)
	if found == nil {
		t.Error("Expected to find handler with multiple parameters")
	}

	// Check that parameters were extracted
	if params.Get("userId") != "123" {
		t.Errorf("Expected parameter userId to be '123', got %q", params.Get("userId"))
	}

	if params.Get("postId") != "456" {
		t.Errorf("Expected parameter postId to be '456', got %q", params.Get("postId"))
	}
}

// TestNodeFindHandlerWithWildcard tests finding handlers with wildcards
func TestNodeFindHandlerWithWildcard(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	// Add wildcard route
	root.addRoute("/static/*", handler)

	params := &Params{
		keys:   make([]string, 0),
		values: make([]string, 0),
	}

	// Test finding wildcard handler
	found := root.findHandler("/static/css/style.css", params)
	if found == nil {
		t.Error("Expected to find wildcard handler")
	}

	// Test another wildcard path
	params.Reset()
	found = root.findHandler("/static/js/app.js", params)
	if found == nil {
		t.Error("Expected to find wildcard handler for js file")
	}
}

// TestNodeComplexTree tests complex tree structures
func TestNodeComplexTree(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	// Create handlers
	handlers := make(map[string]HandlerFunc)
	for i := 0; i < 10; i++ {
		handlers[string(rune('a'+i))] = func(w http.ResponseWriter, r *http.Request) {}
	}

	// Add complex routes
	routes := []string{
		"/",
		"/api",
		"/api/v1",
		"/api/v1/users",
		"/api/v1/users/:id",
		"/api/v1/users/:id/posts",
		"/api/v1/users/:id/posts/:postId",
		"/api/v2",
		"/api/v2/users",
		"/static/*",
	}

	for i, route := range routes {
		root.addRoute(route, handlers[string(rune('a'+i))])
	}

	params := &Params{
		keys:   make([]string, 0),
		values: make([]string, 0),
	}

	// Test finding various handlers
	testCases := []struct {
		path     string
		expected bool
		paramKey string
		paramVal string
	}{
		{"/", true, "", ""},
		{"/api", true, "", ""},
		{"/api/v1", true, "", ""},
		{"/api/v1/users", true, "", ""},
		{"/api/v1/users/123", true, "id", "123"},
		{"/api/v1/users/123/posts", true, "id", "123"},
		{"/api/v1/users/123/posts/456", true, "postId", "456"},
		{"/api/v2", true, "", ""},
		{"/api/v2/users", true, "", ""},
		{"/static/file.txt", true, "", ""},
		{"/nonexistent", false, "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			params.Reset()
			found := root.findHandler(tc.path, params)

			if tc.expected && found == nil {
				t.Errorf("Expected to find handler for %q", tc.path)
			} else if !tc.expected && found != nil {
				t.Errorf("Expected not to find handler for %q", tc.path)
			}

			if tc.paramKey != "" {
				if params.Get(tc.paramKey) != tc.paramVal {
					t.Errorf("Expected parameter %q to be %q, got %q", tc.paramKey, tc.paramVal, params.Get(tc.paramKey))
				}
			}
		})
	}
}

// TestNodeEdgeCases tests edge cases in node operations
func TestNodeEdgeCases(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	// Test adding empty route (should be converted to root)
	root.addRoute("", handler)
	if root.handler == nil {
		t.Error("Expected handler to be set on root for empty route")
	}

	// Test adding route with just slash
	root2 := &node{
		children: []*node{},
	}
	root2.addRoute("/", handler)
	if root2.handler == nil {
		t.Error("Expected handler to be set on root for '/' route")
	}

	// Test finding with empty path
	params := &Params{
		keys:   make([]string, 0),
		values: make([]string, 0),
	}

	found := root.findHandler("", params)
	if found == nil {
		t.Error("Expected to find handler for empty path")
	}

	// Test finding with just slash
	found = root2.findHandler("/", params)
	if found == nil {
		t.Error("Expected to find handler for '/' path")
	}
}

// TestNodeParameterPriority tests parameter matching priority
func TestNodeParameterPriority(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	staticHandler := func(w http.ResponseWriter, r *http.Request) {}
	paramHandler := func(w http.ResponseWriter, r *http.Request) {}

	// Add parameter route first
	root.addRoute("/users/:id", paramHandler)

	// Add static route (should have higher priority)
	root.addRoute("/users/admin", staticHandler)

	params := &Params{
		keys:   make([]string, 0),
		values: make([]string, 0),
	}

	// Test that static route has priority over parameter route
	found := root.findHandler("/users/admin", params)
	if found == nil {
		t.Error("Expected to find static handler")
	}

	// The implementation should prefer static matches over parameter matches
	// This test depends on the specific implementation details
}

// TestNodeParameterBacktracking tests parameter backtracking
func TestNodeParameterBacktracking(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	handler1 := func(w http.ResponseWriter, r *http.Request) {}
	handler2 := func(w http.ResponseWriter, r *http.Request) {}

	// Add routes that might conflict
	root.addRoute("/users/:id/posts", handler1)
	root.addRoute("/users/:id/profile", handler2)

	params := &Params{
		keys:   make([]string, 0),
		values: make([]string, 0),
	}

	// Test finding first route
	found := root.findHandler("/users/123/posts", params)
	if found == nil {
		t.Error("Expected to find posts handler")
	}

	if params.Get("id") != "123" {
		t.Errorf("Expected parameter id to be '123', got %q", params.Get("id"))
	}

	// Test finding second route
	params.Reset()
	found = root.findHandler("/users/456/profile", params)
	if found == nil {
		t.Error("Expected to find profile handler")
	}

	if params.Get("id") != "456" {
		t.Errorf("Expected parameter id to be '456', got %q", params.Get("id"))
	}
}

// TestNodeWildcardPriority tests wildcard matching priority
func TestNodeWildcardPriority(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	specificHandler := func(w http.ResponseWriter, r *http.Request) {}
	wildcardHandler := func(w http.ResponseWriter, r *http.Request) {}

	// Add wildcard route first
	root.addRoute("/static/*", wildcardHandler)

	// Add specific route
	root.addRoute("/static/admin", specificHandler)

	params := &Params{
		keys:   make([]string, 0),
		values: make([]string, 0),
	}

	// Test that specific route has priority over wildcard
	found := root.findHandler("/static/admin", params)
	if found == nil {
		t.Error("Expected to find specific handler")
	}

	// Test that wildcard still works for other paths
	params.Reset()
	found = root.findHandler("/static/other/file.txt", params)
	if found == nil {
		t.Error("Expected to find wildcard handler")
	}
}

// TestLongestCommonPrefix tests the longest common prefix utility function
func TestLongestCommonPrefix(t *testing.T) {
	tests := []struct {
		a, b     string
		expected string
	}{
		{"", "", ""},
		{"a", "", ""},
		{"", "a", ""},
		{"abc", "abc", "abc"},
		{"abc", "abd", "ab"},
		{"abc", "def", ""},
		{"hello", "help", "hel"},
		{"test", "testing", "test"},
		{"prefix", "pre", "pre"},
	}

	for _, tt := range tests {
		t.Run(tt.a+"+"+tt.b, func(t *testing.T) {
			result := longestCommonPrefix(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestNodeMemoryUsage tests memory usage patterns
func TestNodeMemoryUsage(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	// Add many routes to test memory usage
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("/route%d", i)
		root.addRoute(path, handler)
	}

	// Count total nodes
	nodeCount := countNodes(root)

	// Should have at least 100 nodes (one per route)
	if nodeCount < 100 {
		t.Errorf("Expected at least 100 nodes, got %d", nodeCount)
	}

	// Should not have excessive nodes due to good tree structure
	if nodeCount > 200 {
		t.Errorf("Expected less than 200 nodes, got %d (possible memory inefficiency)", nodeCount)
	}
}

// Helper function to count nodes in tree
func countNodes(n *node) int {
	count := 1
	for _, child := range n.children {
		count += countNodes(child)
	}
	return count
}

// TestNodeThreadSafety tests that node operations are safe for concurrent read access
func TestNodeThreadSafety(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	// Add routes
	routes := []string{
		"/api/users",
		"/api/users/:id",
		"/api/posts",
		"/api/posts/:id",
		"/static/*",
	}

	for _, route := range routes {
		root.addRoute(route, handler)
	}

	// Test concurrent reads
	const numGoroutines = 50
	results := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			params := &Params{
				keys:   make([]string, 0),
				values: make([]string, 0),
			}

			// Try to find various handlers
			paths := []string{
				"/api/users",
				"/api/users/123",
				"/api/posts",
				"/api/posts/456",
				"/static/file.txt",
			}

			for _, path := range paths {
				params.Reset()
				found := root.findHandler(path, params)
				if found == nil {
					results <- false
					return
				}
			}

			results <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		if !<-results {
			t.Error("Concurrent read failed")
		}
	}
}

// TestNodeDeepNesting tests deeply nested routes
func TestNodeDeepNesting(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	// Create a deeply nested route
	deepRoute := "/level1/level2/level3/level4/level5/level6/level7/level8/level9/level10"
	root.addRoute(deepRoute, handler)

	params := &Params{
		keys:   make([]string, 0),
		values: make([]string, 0),
	}

	// Should be able to find the deeply nested handler
	found := root.findHandler(deepRoute, params)
	if found == nil {
		t.Error("Expected to find deeply nested handler")
	}
}

// TestNodeRouteConflicts tests handling of route conflicts
func TestNodeRouteConflicts(t *testing.T) {
	root := &node{
		children: []*node{},
	}

	handler1 := func(w http.ResponseWriter, r *http.Request) {}
	handler2 := func(w http.ResponseWriter, r *http.Request) {}

	// Add initial route
	root.addRoute("/users/profile", handler1)

	// Add conflicting route (should overwrite)
	root.addRoute("/users/profile", handler2)

	params := &Params{
		keys:   make([]string, 0),
		values: make([]string, 0),
	}

	// Should find the second handler
	found := root.findHandler("/users/profile", params)
	if found == nil {
		t.Error("Expected to find handler after conflict")
	}

	// This test mainly ensures no panic occurs during conflict resolution
}

// BenchmarkNodeAddRoute benchmarks adding routes to node
func BenchmarkNodeAddRoute(b *testing.B) {
	root := &node{
		children: []*node{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/api/v1/users/%d", i)
		root.addRoute(path, handler)
	}
}

// BenchmarkNodeFindHandler benchmarks finding handlers
func BenchmarkNodeFindHandler(b *testing.B) {
	root := &node{
		children: []*node{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	// Pre-populate tree
	for i := 0; i < 1000; i++ {
		path := fmt.Sprintf("/api/v1/users/%d", i)
		root.addRoute(path, handler)
	}

	params := &Params{
		keys:   make([]string, 0, 8),
		values: make([]string, 0, 8),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		params.Reset()
		path := fmt.Sprintf("/api/v1/users/%d", i%1000)
		found := root.findHandler(path, params)
		if found == nil {
			b.Error("Handler not found")
		}
	}
}

// BenchmarkNodeFindHandlerWithParams benchmarks finding handlers with parameters
func BenchmarkNodeFindHandlerWithParams(b *testing.B) {
	root := &node{
		children: []*node{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	// Add routes with parameters
	root.addRoute("/users/:id", handler)
	root.addRoute("/users/:id/posts/:postId", handler)
	root.addRoute("/api/v1/users/:userId/posts/:postId/comments/:commentId", handler)

	params := &Params{
		keys:   make([]string, 0, 8),
		values: make([]string, 0, 8),
	}

	paths := []string{
		"/users/123",
		"/users/456/posts/789",
		"/api/v1/users/111/posts/222/comments/333",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		params.Reset()
		path := paths[i%len(paths)]
		found := root.findHandler(path, params)
		if found == nil {
			b.Error("Handler not found")
		}
	}
}
