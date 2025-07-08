package forge_router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Test message types for WebSocket testing
type WSTestMessage struct {
	Text string `json:"text"`
	ID   int    `json:"id"`
}

type WSTestResponse struct {
	Echo      string `json:"echo"`
	Timestamp int64  `json:"timestamp"`
}

// SSE test parameter types
type SSETestParams struct {
	Channel string `query:"channel"`
	UserID  int    `path:"userId"`
}

// TestWSConnectionCreation tests WebSocket connection creation
func TestWSConnectionCreation(t *testing.T) {
	router := NewRouter()

	// Test WebSocket handler registration
	router.WebSocket("/ws", func(conn *WSConnection, message WSTestMessage) (*WSTestResponse, error) {
		return &WSTestResponse{
			Echo:      message.Text,
			Timestamp: time.Now().Unix(),
		}, nil
	}, WithAsyncSummary("Test WebSocket"), WithAsyncDescription("Test WebSocket endpoint"))

	// Check that handler was registered
	if len(router.wsHandlers) != 1 {
		t.Errorf("Expected 1 WebSocket handler, got %d", len(router.wsHandlers))
	}

	if handler, exists := router.wsHandlers["/ws"]; !exists {
		t.Error("Expected WebSocket handler to be registered")
	} else {
		if handler.Summary != "Test WebSocket" {
			t.Errorf("Expected summary 'Test WebSocket', got %q", handler.Summary)
		}
		if handler.Description != "Test WebSocket endpoint" {
			t.Errorf("Expected description 'Test WebSocket endpoint', got %q", handler.Description)
		}
	}
}

// TestWSConnectionManager tests WebSocket connection management
func TestWSConnectionManager(t *testing.T) {
	cm := NewConnectionManager()

	// Test initial state
	if len(cm.WSConnections()) != 0 {
		t.Errorf("Expected 0 WebSocket connections, got %d", len(cm.WSConnections()))
	}

	// Mock WebSocket connection
	conn := &WSConnection{
		clientID: "test-client-1",
		metadata: make(map[string]interface{}),
	}

	// Test adding connection
	cm.AddWSConnection("test-client-1", conn)

	if len(cm.WSConnections()) != 1 {
		t.Errorf("Expected 1 WebSocket connection, got %d", len(cm.WSConnections()))
	}

	// Test removing connection
	cm.RemoveWSConnection("test-client-1")

	if len(cm.WSConnections()) != 0 {
		t.Errorf("Expected 0 WebSocket connections after removal, got %d", len(cm.WSConnections()))
	}
}

// TestWSMessage tests WebSocket message structure
func TestWSMessage(t *testing.T) {
	msg := WSMessage{
		Type:    "test",
		Payload: map[string]string{"key": "value"},
		ID:      "msg-123",
	}

	// Test JSON marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal WebSocket message: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled WSMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal WebSocket message: %v", err)
	}

	if unmarshaled.Type != "test" {
		t.Errorf("Expected type 'test', got %q", unmarshaled.Type)
	}

	if unmarshaled.ID != "msg-123" {
		t.Errorf("Expected ID 'msg-123', got %q", unmarshaled.ID)
	}
}

// TestWSError tests WebSocket error structure
func TestWSError(t *testing.T) {
	wsErr := &WSError{
		Code:    "INVALID_MESSAGE",
		Message: "Invalid message format",
		Details: map[string]string{"field": "text"},
	}

	// Test JSON marshaling
	data, err := json.Marshal(wsErr)
	if err != nil {
		t.Fatalf("Failed to marshal WebSocket error: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled WSError
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal WebSocket error: %v", err)
	}

	if unmarshaled.Code != "INVALID_MESSAGE" {
		t.Errorf("Expected code 'INVALID_MESSAGE', got %q", unmarshaled.Code)
	}

	if unmarshaled.Message != "Invalid message format" {
		t.Errorf("Expected message 'Invalid message format', got %q", unmarshaled.Message)
	}
}

// TestWSConnectionMetadata tests WebSocket connection metadata
func TestWSConnectionMetadata(t *testing.T) {
	conn := &WSConnection{
		metadata: make(map[string]interface{}),
	}

	// Test setting metadata
	conn.SetMetadata("user_id", 123)
	conn.SetMetadata("channel", "general")

	// Test getting metadata
	if userID, ok := conn.GetMetadata("user_id"); !ok || userID != 123 {
		t.Errorf("Expected user_id to be 123, got %v", userID)
	}

	if channel, ok := conn.GetMetadata("channel"); !ok || channel != "general" {
		t.Errorf("Expected channel to be 'general', got %v", channel)
	}

	// Test getting non-existent metadata
	if _, ok := conn.GetMetadata("nonexistent"); ok {
		t.Error("Expected non-existent metadata to return false")
	}
}

// TestWSConnectionParams tests WebSocket connection parameter handling
func TestWSConnectionParams(t *testing.T) {
	params := &Params{
		keys:   []string{"id", "channel"},
		values: []string{"123", "general"},
	}

	conn := &WSConnection{
		params: params,
	}

	// Test parameter retrieval
	if id := conn.Param("id"); id != "123" {
		t.Errorf("Expected id parameter to be '123', got %q", id)
	}

	if channel := conn.Param("channel"); channel != "general" {
		t.Errorf("Expected channel parameter to be 'general', got %q", channel)
	}

	// Test non-existent parameter
	if nonexistent := conn.Param("nonexistent"); nonexistent != "" {
		t.Errorf("Expected non-existent parameter to be empty, got %q", nonexistent)
	}
}

// TestSSEConnectionCreation tests SSE connection creation
func TestSSEConnectionCreation(t *testing.T) {
	router := NewRouter()

	// Test SSE handler registration
	router.SSE("/events/:userId", func(conn *SSEConnection, params SSETestParams) error {
		// Send a test message
		return conn.SendMessage(SSEMessage{
			Event: "test",
			Data:  map[string]interface{}{"user_id": params.UserID, "channel": params.Channel},
		})
	}, WithAsyncSummary("Test SSE"), WithAsyncDescription("Test SSE endpoint"))

	// Check that handler was registered
	if len(router.sseHandlers) != 1 {
		t.Errorf("Expected 1 SSE handler, got %d", len(router.sseHandlers))
	}

	if handler, exists := router.sseHandlers["/events/:userId"]; !exists {
		t.Error("Expected SSE handler to be registered")
	} else {
		if handler.Summary != "Test SSE" {
			t.Errorf("Expected summary 'Test SSE', got %q", handler.Summary)
		}
		if handler.Description != "Test SSE endpoint" {
			t.Errorf("Expected description 'Test SSE endpoint', got %q", handler.Description)
		}
	}
}

// TestSSEConnectionManager tests SSE connection management
func TestSSEConnectionManager(t *testing.T) {
	cm := NewConnectionManager()

	// Test initial state
	if len(cm.SSEConnections()) != 0 {
		t.Errorf("Expected 0 SSE connections, got %d", len(cm.SSEConnections()))
	}

	// Mock SSE connection
	conn := &SSEConnection{
		clientID: "test-client-1",
		metadata: make(map[string]interface{}),
	}

	// Test adding connection
	cm.AddSSEConnection("test-client-1", conn)

	if len(cm.SSEConnections()) != 1 {
		t.Errorf("Expected 1 SSE connection, got %d", len(cm.SSEConnections()))
	}

	// Test removing connection
	cm.RemoveSSEConnection("test-client-1")

	if len(cm.SSEConnections()) != 0 {
		t.Errorf("Expected 0 SSE connections after removal, got %d", len(cm.SSEConnections()))
	}
}

// TestSSEMessage tests SSE message structure
func TestSSEMessage(t *testing.T) {
	msg := SSEMessage{
		ID:    "msg-123",
		Event: "user_update",
		Data:  map[string]interface{}{"user_id": 123, "name": "John"},
		Retry: 5000,
	}

	// Test JSON marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal SSE message: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled SSEMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal SSE message: %v", err)
	}

	if unmarshaled.ID != "msg-123" {
		t.Errorf("Expected ID 'msg-123', got %q", unmarshaled.ID)
	}

	if unmarshaled.Event != "user_update" {
		t.Errorf("Expected event 'user_update', got %q", unmarshaled.Event)
	}

	if unmarshaled.Retry != 5000 {
		t.Errorf("Expected retry 5000, got %d", unmarshaled.Retry)
	}
}

// TestSSEConnectionMetadata tests SSE connection metadata
func TestSSEConnectionMetadata(t *testing.T) {
	conn := &SSEConnection{
		metadata: make(map[string]interface{}),
	}

	// Test setting metadata
	conn.SetMetadata("user_id", 123)
	conn.SetMetadata("subscriptions", []string{"news", "updates"})

	// Test getting metadata
	if userID, ok := conn.GetMetadata("user_id"); !ok || userID != 123 {
		t.Errorf("Expected user_id to be 123, got %v", userID)
	}

	if subs, ok := conn.GetMetadata("subscriptions"); !ok {
		t.Error("Expected subscriptions metadata to exist")
	} else if subsSlice, ok := subs.([]string); !ok || len(subsSlice) != 2 {
		t.Errorf("Expected subscriptions to be slice of 2 strings, got %v", subs)
	}
}

// TestSSEConnectionParams tests SSE connection parameter handling
func TestSSEConnectionParams(t *testing.T) {
	params := &Params{
		keys:   []string{"userId", "channel"},
		values: []string{"456", "notifications"},
	}

	conn := &SSEConnection{
		params: params,
	}

	// Test parameter retrieval
	if userID := conn.Param("userId"); userID != "456" {
		t.Errorf("Expected userId parameter to be '456', got %q", userID)
	}

	if channel := conn.Param("channel"); channel != "notifications" {
		t.Errorf("Expected channel parameter to be 'notifications', got %q", channel)
	}
}

// TestSSEConnectionClose tests SSE connection close functionality
func TestSSEConnectionClose(t *testing.T) {
	conn := &SSEConnection{
		closed: false,
	}

	// Test initial state
	if conn.IsClosed() {
		t.Error("Expected connection to be open initially")
	}

	// Test closing connection
	conn.Close()

	if !conn.IsClosed() {
		t.Error("Expected connection to be closed after Close()")
	}

	// Test sending message to closed connection
	err := conn.SendMessage(SSEMessage{
		Event: "test",
		Data:  "test data",
	})

	if err == nil {
		t.Error("Expected error when sending message to closed connection")
	}
}

// TestAsyncAPIGeneration tests AsyncAPI specification generation
func TestAsyncAPIGeneration(t *testing.T) {
	router := NewRouter()

	// Add WebSocket handler
	router.WebSocket("/ws/chat", func(conn *WSConnection, message WSTestMessage) (*WSTestResponse, error) {
		return &WSTestResponse{
			Echo:      message.Text,
			Timestamp: time.Now().Unix(),
		}, nil
	}, WithAsyncSummary("Chat WebSocket"), WithAsyncDescription("Real-time chat"), WithAsyncTags("chat", "websocket"))

	// Add SSE handler
	router.SSE("/events/:userId", func(conn *SSEConnection, params SSETestParams) error {
		return conn.SendMessage(SSEMessage{
			Event: "notification",
			Data:  map[string]interface{}{"user_id": params.UserID},
		})
	}, WithAsyncSummary("User Events"), WithAsyncDescription("User-specific events"), WithAsyncTags("events", "sse"))

	// Enable AsyncAPI
	router.EnableAsyncAPI()

	// Test AsyncAPI spec endpoint
	req := httptest.NewRequest("GET", "/asyncapi", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
	}

	// Parse AsyncAPI spec
	var spec map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("Failed to decode AsyncAPI spec: %v", err)
	}

	// Check AsyncAPI version
	if asyncapi, ok := spec["asyncapi"]; !ok || asyncapi != "2.6.0" {
		t.Errorf("Expected AsyncAPI version 2.6.0, got %v", asyncapi)
	}

	// Check channels
	if channels, ok := spec["channels"]; !ok {
		t.Error("Expected channels in AsyncAPI spec")
	} else if channelsMap, ok := channels.(map[string]interface{}); !ok {
		t.Error("Expected channels to be a map")
	} else {
		if _, ok := channelsMap["/ws/chat"]; !ok {
			t.Error("Expected /ws/chat channel in AsyncAPI spec")
		}
		if _, ok := channelsMap["/events/:userId"]; !ok {
			t.Error("Expected /events/:userId channel in AsyncAPI spec")
		}
	}
}

// TestAsyncHandlerOptions tests async handler options
func TestAsyncHandlerOptions(t *testing.T) {
	router := NewRouter()

	// Test WebSocket handler with options
	router.WebSocket("/ws/test", func(conn *WSConnection, message WSTestMessage) (*WSTestResponse, error) {
		return &WSTestResponse{}, nil
	}, WithAsyncSummary("Test Summary"), WithAsyncDescription("Test Description"), WithAsyncTags("test", "websocket"))

	// Test SSE handler with options
	router.SSE("/sse/test", func(conn *SSEConnection, params SSETestParams) error {
		return nil
	}, WithAsyncSummary("SSE Summary"), WithAsyncDescription("SSE Description"), WithAsyncTags("test", "sse"))

	// Check WebSocket handler options
	if handler, exists := router.wsHandlers["/ws/test"]; !exists {
		t.Error("Expected WebSocket handler to be registered")
	} else {
		if handler.Summary != "Test Summary" {
			t.Errorf("Expected summary 'Test Summary', got %q", handler.Summary)
		}
		if handler.Description != "Test Description" {
			t.Errorf("Expected description 'Test Description', got %q", handler.Description)
		}
		if len(handler.Tags) != 2 || handler.Tags[0] != "test" || handler.Tags[1] != "websocket" {
			t.Errorf("Expected tags ['test', 'websocket'], got %v", handler.Tags)
		}
	}

	// Check SSE handler options
	if handler, exists := router.sseHandlers["/sse/test"]; !exists {
		t.Error("Expected SSE handler to be registered")
	} else {
		if handler.Summary != "SSE Summary" {
			t.Errorf("Expected summary 'SSE Summary', got %q", handler.Summary)
		}
		if handler.Description != "SSE Description" {
			t.Errorf("Expected description 'SSE Description', got %q", handler.Description)
		}
		if len(handler.Tags) != 2 || handler.Tags[0] != "test" || handler.Tags[1] != "sse" {
			t.Errorf("Expected tags ['test', 'sse'], got %v", handler.Tags)
		}
	}
}

// TestConnectionManagerBroadcast tests connection manager broadcast functionality
func TestConnectionManagerBroadcast(t *testing.T) {
	cm := NewConnectionManager()

	// Create mock connections
	wsConn1 := &WSConnection{
		clientID: "ws-client-1",
		metadata: make(map[string]interface{}),
	}

	wsConn2 := &WSConnection{
		clientID: "ws-client-2",
		metadata: make(map[string]interface{}),
	}

	sseConn1 := &SSEConnection{
		clientID: "sse-client-1",
		metadata: make(map[string]interface{}),
	}

	// Add connections
	cm.AddWSConnection("ws-client-1", wsConn1)
	cm.AddWSConnection("ws-client-2", wsConn2)
	cm.AddSSEConnection("sse-client-1", sseConn1)

	// Test broadcast (this tests the broadcast methods exist and don't panic)
	wsMessage := WSMessage{
		Type:    "broadcast",
		Payload: "test message",
	}

	sseMessage := SSEMessage{
		Event: "broadcast",
		Data:  "test message",
	}

	// These should not panic
	cm.BroadcastWS(wsMessage)
	cm.BroadcastSSE(sseMessage)

	// Verify connections are still registered
	if len(cm.WSConnections()) != 2 {
		t.Errorf("Expected 2 WebSocket connections, got %d", len(cm.WSConnections()))
	}

	if len(cm.SSEConnections()) != 1 {
		t.Errorf("Expected 1 SSE connection, got %d", len(cm.SSEConnections()))
	}
}

// TestGenerateClientID tests client ID generation
func TestGenerateClientID(t *testing.T) {
	id1 := generateClientID()
	id2 := generateClientID()

	// IDs should be different
	if id1 == id2 {
		t.Error("Expected different client IDs")
	}

	// IDs should have expected prefix
	if !strings.HasPrefix(id1, "client_") {
		t.Errorf("Expected client ID to start with 'client_', got %q", id1)
	}

	if !strings.HasPrefix(id2, "client_") {
		t.Errorf("Expected client ID to start with 'client_', got %q", id2)
	}
}

// TestAsyncAPISchemaGeneration tests AsyncAPI schema generation
func TestAsyncAPISchemaGeneration(t *testing.T) {
	router := NewRouter()

	type ComplexMessage struct {
		ID       int               `json:"id" description:"Message ID"`
		Content  string            `json:"content" description:"Message content"`
		Tags     []string          `json:"tags" description:"Message tags"`
		Metadata map[string]string `json:"metadata" description:"Message metadata"`
	}

	type ComplexResponse struct {
		Success   bool  `json:"success" description:"Operation success"`
		MessageID int   `json:"message_id" description:"Created message ID"`
		Timestamp int64 `json:"timestamp" description:"Creation timestamp"`
	}

	// Add WebSocket handler with complex types
	router.WebSocket("/ws/complex", func(conn *WSConnection, message ComplexMessage) (*ComplexResponse, error) {
		return &ComplexResponse{
			Success:   true,
			MessageID: message.ID,
			Timestamp: time.Now().Unix(),
		}, nil
	})

	// Check that schemas were generated
	if router.asyncAPISpec.Components.Schemas == nil {
		t.Error("Expected schemas to be generated")
	}

	// Check for ComplexMessage schema
	if _, exists := router.asyncAPISpec.Components.Schemas["ComplexMessage"]; !exists {
		t.Error("Expected ComplexMessage schema to be generated")
	}

	// Check for ComplexResponse schema
	if _, exists := router.asyncAPISpec.Components.Schemas["ComplexResponse"]; !exists {
		t.Error("Expected ComplexResponse schema to be generated")
	}
}

// TestAsyncAPIDocumentation tests AsyncAPI documentation endpoints
func TestAsyncAPIDocumentation(t *testing.T) {
	router := NewRouter()

	// Add a simple handler
	router.WebSocket("/ws/test", func(conn *WSConnection, message WSTestMessage) (*WSTestResponse, error) {
		return &WSTestResponse{}, nil
	})

	router.EnableAsyncAPI()

	// Test documentation endpoints
	endpoints := []string{
		"/asyncapi/docs",
		"/asyncapi/simple",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d for %s, got %d", http.StatusOK, endpoint, w.Code)
			}

			if contentType := w.Header().Get("Content-Type"); contentType != "text/html" {
				t.Errorf("Expected Content-Type 'text/html' for %s, got %q", endpoint, contentType)
			}

			if w.Body.Len() == 0 {
				t.Errorf("Expected non-empty body for %s", endpoint)
			}
		})
	}
}

// TestConcurrentConnectionManagement tests concurrent connection management
func TestConcurrentConnectionManagement(t *testing.T) {
	cm := NewConnectionManager()

	const numGoroutines = 50
	results := make(chan bool, numGoroutines)

	// Test concurrent connection operations
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer func() {
				if r := recover(); r != nil {
					results <- false
					return
				}
			}()

			clientID := fmt.Sprintf("client-%d", i)

			// Add WebSocket connection
			wsConn := &WSConnection{
				clientID: clientID,
				metadata: make(map[string]interface{}),
			}
			cm.AddWSConnection(clientID, wsConn)

			// Add SSE connection
			sseConn := &SSEConnection{
				clientID: clientID,
				metadata: make(map[string]interface{}),
			}
			cm.AddSSEConnection(clientID, sseConn)

			// Set metadata
			wsConn.SetMetadata("test", i)
			sseConn.SetMetadata("test", i)

			// Get metadata
			if val, ok := wsConn.GetMetadata("test"); !ok || val != i {
				results <- false
				return
			}

			if val, ok := sseConn.GetMetadata("test"); !ok || val != i {
				results <- false
				return
			}

			// Remove connections
			cm.RemoveWSConnection(clientID)
			cm.RemoveSSEConnection(clientID)

			results <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		if !<-results {
			t.Error("Concurrent connection management failed")
		}
	}
}

// TestAsyncAPITypeConversion tests AsyncAPI type conversion
func TestAsyncAPITypeConversion(t *testing.T) {
	router := NewRouter()
	router.EnableAsyncAPI()

	type TestStruct struct {
		StringField  string            `json:"string_field" description:"String field"`
		IntField     int               `json:"int_field" description:"Integer field"`
		FloatField   float64           `json:"float_field" description:"Float field"`
		BoolField    bool              `json:"bool_field" description:"Boolean field"`
		SliceField   []string          `json:"slice_field" description:"Slice field"`
		MapField     map[string]string `json:"map_field" description:"Map field"`
		PointerField *string           `json:"pointer_field,omitempty" description:"Pointer field"`
	}

	// Trigger the schema generation by calling the function
	_ = router.typeToAsyncAPISchema(reflect.TypeOf(TestStruct{}))

	// Retrieve the actual schema from the components
	schema, exists := router.asyncAPISpec.Components.Schemas["TestStruct"]
	if !exists {
		t.Fatal("Expected TestStruct schema to be generated and registered in components")
	}

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got %q", schema.Type)
	}

	if schema.Properties == nil {
		t.Fatal("Expected properties to be set")
	}

	// Check specific field types
	if _, ok := schema.Properties["string_field"]; !ok {
		t.Error("Expected string_field in properties")
	} else if schema.Properties["string_field"].Type != "string" {
		t.Errorf("Expected string_field type 'string', got %q", schema.Properties["string_field"].Type)
	}

	if _, ok := schema.Properties["int_field"]; !ok {
		t.Error("Expected int_field in properties")
	} else if schema.Properties["int_field"].Type != "integer" {
		t.Errorf("Expected int_field type 'integer', got %q", schema.Properties["int_field"].Type)
	}

	if _, ok := schema.Properties["float_field"]; !ok {
		t.Error("Expected float_field in properties")
	} else if schema.Properties["float_field"].Type != "number" {
		t.Errorf("Expected float_field type 'number', got %q", schema.Properties["float_field"].Type)
	}

	if _, ok := schema.Properties["bool_field"]; !ok {
		t.Error("Expected bool_field in properties")
	} else if schema.Properties["bool_field"].Type != "boolean" {
		t.Errorf("Expected bool_field type 'boolean', got %q", schema.Properties["bool_field"].Type)
	}

	if _, ok := schema.Properties["slice_field"]; !ok {
		t.Error("Expected slice_field in properties")
	} else if schema.Properties["slice_field"].Type != "array" {
		t.Errorf("Expected slice_field type 'array', got %q", schema.Properties["slice_field"].Type)
	}
}

// TestRouterConnectionManager tests router connection manager integration
func TestRouterConnectionManager(t *testing.T) {
	router := NewRouter()

	// Test that connection manager is initialized
	cm := router.ConnectionManager()
	if cm == nil {
		t.Error("Expected connection manager to be initialized")
	}

	// Test that same instance is returned
	cm2 := router.ConnectionManager()
	if cm != cm2 {
		t.Error("Expected same connection manager instance")
	}
}

// BenchmarkWSConnectionMetadata benchmarks WebSocket connection metadata operations
func BenchmarkWSConnectionMetadata(b *testing.B) {
	conn := &WSConnection{
		metadata: make(map[string]interface{}),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%100)
		conn.SetMetadata(key, i)
		if _, ok := conn.GetMetadata(key); !ok {
			b.Error("Metadata not found")
		}
	}
}

// BenchmarkSSEConnectionMetadata benchmarks SSE connection metadata operations
func BenchmarkSSEConnectionMetadata(b *testing.B) {
	conn := &SSEConnection{
		metadata: make(map[string]interface{}),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%100)
		conn.SetMetadata(key, i)
		if _, ok := conn.GetMetadata(key); !ok {
			b.Error("Metadata not found")
		}
	}
}

// BenchmarkConnectionManagerOperations benchmarks connection manager operations
func BenchmarkConnectionManagerOperations(b *testing.B) {
	cm := NewConnectionManager()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		clientID := fmt.Sprintf("client-%d", i)

		// Add WebSocket connection
		wsConn := &WSConnection{
			clientID: clientID,
			metadata: make(map[string]interface{}),
		}
		cm.AddWSConnection(clientID, wsConn)

		// Add SSE connection
		sseConn := &SSEConnection{
			clientID: clientID,
			metadata: make(map[string]interface{}),
		}
		cm.AddSSEConnection(clientID, sseConn)

		// Remove connections
		cm.RemoveWSConnection(clientID)
		cm.RemoveSSEConnection(clientID)
	}
}
