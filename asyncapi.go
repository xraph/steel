package forgerouter

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// AsyncAPISpec represents the structure of an AsyncAPI specification document.
type AsyncAPISpec struct {
	AsyncAPI   string                     `json:"asyncapi"`
	Info       AsyncAPIInfo               `json:"info"`
	Servers    map[string]AsyncAPIServer  `json:"servers"`
	Channels   map[string]AsyncAPIChannel `json:"channels"`
	Components AsyncAPIComponents         `json:"components"`
}

// AsyncAPIInfo represents metadata about an AsyncAPI specification.
// It includes the title, version, and a description of the API.
type AsyncAPIInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// AsyncAPIServer represents a server object in an AsyncAPI specification, containing its URL, protocol, and other details.
type AsyncAPIServer struct {
	URL         string                 `json:"url"`
	Protocol    string                 `json:"protocol"`
	Description string                 `json:"description,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
}

// AsyncAPIChannel represents a communication channel in an AsyncAPI specification.
// It defines operations for subscribing and publishing, associated parameters, and protocol-specific bindings.
type AsyncAPIChannel struct {
	Description string                   `json:"description,omitempty"`
	Subscribe   *AsyncAPIOperation       `json:"subscribe,omitempty"`
	Publish     *AsyncAPIOperation       `json:"publish,omitempty"`
	Parameters  map[string]AsyncAPIParam `json:"parameters,omitempty"`
	Bindings    map[string]interface{}   `json:"bindings,omitempty"`
}

// AsyncAPIOperation represents an operation in an AsyncAPI definition, such as publish or subscribe in a channel description.
type AsyncAPIOperation struct {
	OperationID string                 `json:"operationId,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []AsyncAPITag          `json:"tags,omitempty"`
	Message     AsyncAPIMessage        `json:"message"`
	Bindings    map[string]interface{} `json:"bindings,omitempty"`
}

// AsyncAPIMessage represents a message in an AsyncAPI specification, including metadata, payload, headers, and bindings.
type AsyncAPIMessage struct {
	Name        string                 `json:"name,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	ContentType string                 `json:"contentType,omitempty"`
	Payload     AsyncAPISchema         `json:"payload,omitempty"`
	Headers     AsyncAPISchema         `json:"headers,omitempty"`
	Examples    []AsyncAPIExample      `json:"examples,omitempty"`
	Bindings    map[string]interface{} `json:"bindings,omitempty"`
}

// AsyncAPISchema defines a schema object used to describe message payloads, including type, format, and properties.
type AsyncAPISchema struct {
	Type        string                    `json:"type,omitempty"`
	Format      string                    `json:"format,omitempty"`
	Description string                    `json:"description,omitempty"`
	Properties  map[string]AsyncAPISchema `json:"properties,omitempty"`
	Required    []string                  `json:"required,omitempty"`
	Items       *AsyncAPISchema           `json:"items,omitempty"`
	Ref         string                    `json:"$ref,omitempty"`
	Example     interface{}               `json:"example,omitempty"`
}

// AsyncAPIParam represents an AsyncAPI parameter with a description, associated schema, and its location in the API.
type AsyncAPIParam struct {
	Description string         `json:"description,omitempty"`
	Schema      AsyncAPISchema `json:"schema,omitempty"`
	Location    string         `json:"location,omitempty"`
}

// AsyncAPITag represents a tag used in the AsyncAPI specification, including a name and an optional description.
type AsyncAPITag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// AsyncAPIExample represents an example object used in AsyncAPI messages, containing a name, summary, and value.
type AsyncAPIExample struct {
	Name    string      `json:"name,omitempty"`
	Summary string      `json:"summary,omitempty"`
	Value   interface{} `json:"value,omitempty"`
}

// AsyncAPIComponents represents the components section of an AsyncAPI document, holding reusable definitions for schemas and messages.
type AsyncAPIComponents struct {
	Schemas  map[string]AsyncAPISchema  `json:"schemas,omitempty"`
	Messages map[string]AsyncAPIMessage `json:"messages,omitempty"`
}

// ConnectionManager manages WebSocket and Server-Sent Event connections concurrently.
// It provides methods to add, remove, retrieve, and broadcast messages to these connections.
// Thread-safety is ensured through a read-write mutex.
type ConnectionManager struct {
	wsConnections  map[string]*WSConnection
	sseConnections map[string]*SSEConnection
	mu             sync.RWMutex
}

// NewConnectionManager initializes a new instance of ConnectionManager with separate maps for WebSocket and SSE connections.
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		wsConnections:  make(map[string]*WSConnection),
		sseConnections: make(map[string]*SSEConnection),
	}
}

// AddWSConnection adds a WebSocket connection to the ConnectionManager, associating it with the given unique ID.
func (cm *ConnectionManager) AddWSConnection(id string, conn *WSConnection) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.wsConnections[id] = conn
}

// WSConnections returns the map of active WebSocket connections managed by the ConnectionManager.
func (cm *ConnectionManager) WSConnections() map[string]*WSConnection {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.wsConnections
}

// SSEConnections returns the map of all active Server-Sent Events (SSE) connections managed by the ConnectionManager.
func (cm *ConnectionManager) SSEConnections() map[string]*SSEConnection {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.sseConnections
}

// RemoveWSConnection removes a WebSocket connection from the connection manager using the provided connection ID.
func (cm *ConnectionManager) RemoveWSConnection(id string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.wsConnections, id)
}

// AddSSEConnection adds a Server-Sent Events (SSE) connection to the ConnectionManager with the specified ID.
func (cm *ConnectionManager) AddSSEConnection(id string, conn *SSEConnection) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.sseConnections[id] = conn
}

// RemoveSSEConnection removes an SSE connection from the connection manager using the specified connection ID.
func (cm *ConnectionManager) RemoveSSEConnection(id string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.sseConnections, id)
}

// BroadcastWS sends a WebSocket message to all active WebSocket connections managed by the ConnectionManager.
func (cm *ConnectionManager) BroadcastWS(message WSMessage) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	for _, conn := range cm.wsConnections {
		conn.SendMessage(message)
	}
}

// BroadcastSSE sends the given SSEMessage to all active SSE connections managed by the ConnectionManager.
func (cm *ConnectionManager) BroadcastSSE(message SSEMessage) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	for _, conn := range cm.sseConnections {
		conn.SendMessage(message)
	}
}

// initAsyncAPI initializes the AsyncAPI specification, WebSocket and SSE handlers, and the connection manager for the router.
func (r *FastRouter) initAsyncAPI() {
	if r.asyncAPISpec == nil {
		r.asyncAPISpec = &AsyncAPISpec{
			AsyncAPI: "2.6.0",
			Info: AsyncAPIInfo{
				Title:       r.options.OpenAPITitle,
				Version:     r.options.OpenAPIVersion,
				Description: r.options.OpenAPIDescription,
			},
			Servers: map[string]AsyncAPIServer{
				"production": {
					URL:         "ws://localhost:8080",
					Protocol:    "ws",
					Description: "WebSocket server",
				},
			},
			Channels: make(map[string]AsyncAPIChannel),
			Components: AsyncAPIComponents{
				Schemas:  make(map[string]AsyncAPISchema),
				Messages: make(map[string]AsyncAPIMessage),
			},
		}
	}

	if r.wsHandlers == nil {
		r.wsHandlers = make(map[string]*WSHandlerInfo)
	}

	if r.sseHandlers == nil {
		r.sseHandlers = make(map[string]*SSEHandlerInfo)
	}

	if r.connectionManager == nil {
		r.connectionManager = NewConnectionManager()
	}
}

// WebSocket sets up a WebSocket route with the specified pattern and handler function.
// The handler can process incoming WebSocket messages and send responses.
// Additional options can be applied using AsyncHandlerOption.
func (r *FastRouter) WebSocket(pattern string, handler interface{}, opts ...AsyncHandlerOption) {
	r.initAsyncAPI()
	r.registerWSHandler(pattern, handler, opts...)
}

// registerWSHandler registers a WebSocket handler for the given URL pattern with optional handler options.
// The handler must be a function with the signature func(*WSConnection, MessageType) (*ResponseType, error).
// It validates handler signatures, creates info structures, and sets up the WebSocket HTTP handler.
// Handlers are automatically managed under the router's connection manager for easy tracking and lifecycle handling.
// Upgrades HTTP connections to WebSocket, manages connections, and integrates with FastRouter's async API generation.
func (r *FastRouter) registerWSHandler(pattern string, handler interface{}, opts ...AsyncHandlerOption) {
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Func {
		panic("WebSocket handler must be a function")
	}

	if handlerType.NumIn() != 2 || handlerType.NumOut() != 2 {
		panic("WebSocket handler must have signature func(*WSConnection, MessageType) (*ResponseType, error)")
	}

	messageType := handlerType.In(1)
	responseType := handlerType.Out(0)

	if responseType.Kind() == reflect.Ptr {
		responseType = responseType.Elem()
	}

	info := &WSHandlerInfo{
		Path:         pattern,
		MessageType:  messageType,
		ResponseType: responseType,
		Handler:      handler,
	}

	for _, opt := range opts {
		opt.ApplyToWS(info)
	}

	r.wsHandlers[pattern] = info
	r.generateAsyncAPIForWS(info)

	// Create HTTP handler for WebSocket upgrade
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Configure as needed
		},
	}

	httpHandler := func(w http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		params := r.pool.Get().(*Params)
		params.Reset()
		defer r.pool.Put(params)

		r.extractURLParams(req.URL.Path, req.Method, params)

		clientID := generateClientID()
		wsConn := &WSConnection{
			conn:     conn,
			router:   r,
			params:   params,
			request:  req,
			clientID: clientID,
			metadata: make(map[string]interface{}),
		}

		r.connectionManager.AddWSConnection(clientID, wsConn)
		defer r.connectionManager.RemoveWSConnection(clientID)

		r.handleWebSocketConnection(wsConn, handler, messageType, responseType)
	}

	r.GET(pattern, httpHandler)
}

// handleWebSocketConnection manages communication with a WebSocket client, handling incoming messages and sending responses.
// wsConn represents the WebSocket connection instance for the client.
// handler is the function invoked to process incoming WebSocket messages.
// messageType specifies the type of the message payload expected by the handler.
// responseType defines the type of response payload returned by the handler.
func (r *FastRouter) handleWebSocketConnection(wsConn *WSConnection, handler interface{}, messageType, responseType reflect.Type) {
	handlerValue := reflect.ValueOf(handler)
	defer wsConn.Close()

	for {
		var rawMessage WSMessage
		if err := wsConn.conn.ReadJSON(&rawMessage); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse message payload into expected type
		payloadData, _ := json.Marshal(rawMessage.Payload)
		message := reflect.New(messageType).Interface()

		if err := json.Unmarshal(payloadData, message); err != nil {
			wsConn.SendMessage(WSMessage{
				Type: "error",
				Error: &WSError{
					Code:    "INVALID_MESSAGE",
					Message: "Failed to parse message",
					Details: err.Error(),
				},
			})
			continue
		}

		// Call handler
		results := handlerValue.Call([]reflect.Value{
			reflect.ValueOf(wsConn),
			reflect.ValueOf(message).Elem(),
		})

		response := results[0]
		err := results[1]

		if !err.IsNil() {
			wsConn.SendMessage(WSMessage{
				Type: "error",
				Error: &WSError{
					Code:    "HANDLER_ERROR",
					Message: err.Interface().(error).Error(),
				},
			})
			continue
		}

		if !response.IsNil() {
			wsConn.SendMessage(WSMessage{
				Type:    "response",
				Payload: response.Interface(),
				ID:      rawMessage.ID,
			})
		}
	}
}

// SSE registers a server-sent events (SSE) handler for the specified pattern with optional configuration options.
func (r *FastRouter) SSE(pattern string, handler interface{}, opts ...AsyncHandlerOption) {
	r.initAsyncAPI()
	r.registerSSEHandler(pattern, handler, opts...)
}

// registerSSEHandler registers an SSE handler with a specified URL pattern and handler function.
// The handler must follow the signature func(*SSEConnection, ParamsType) error.
// Additional options can be applied through AsyncHandlerOption arguments.
// An HTTP GET route is added to handle SSE requests, setting required headers and managing connections.
func (r *FastRouter) registerSSEHandler(pattern string, handler interface{}, opts ...AsyncHandlerOption) {
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Func {
		panic("SSE handler must be a function")
	}

	if handlerType.NumIn() != 2 || handlerType.NumOut() != 1 {
		panic("SSE handler must have signature func(*SSEConnection, ParamsType) error")
	}

	paramsType := handlerType.In(1)

	info := &SSEHandlerInfo{
		Path:       pattern,
		ParamsType: paramsType,
		Handler:    handler,
	}

	for _, opt := range opts {
		opt.ApplyToSSE(info)
	}

	r.sseHandlers[pattern] = info
	r.generateAsyncAPIForSSE(info)

	httpHandler := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		params := r.pool.Get().(*Params)
		params.Reset()
		defer r.pool.Put(params)

		r.extractURLParams(req.URL.Path, req.Method, params)

		clientID := generateClientID()
		sseConn := &SSEConnection{
			writer:   w,
			router:   r,
			params:   params,
			request:  req,
			clientID: clientID,
			metadata: make(map[string]interface{}),
		}

		r.connectionManager.AddSSEConnection(clientID, sseConn)
		defer r.connectionManager.RemoveSSEConnection(clientID)

		r.handleSSEConnection(sseConn, handler, paramsType)
	}

	r.GET(pattern, httpHandler)
}

// handleSSEConnection handles an SSE connection by binding parameters, invoking the handler, and processing the result.
// sseConn represents the server-sent events connection used within the handler.
// handler is the function to execute for handling the events.
// paramsType specifies the expected type for parameters the handler will use.
func (r *FastRouter) handleSSEConnection(sseConn *SSEConnection, handler interface{}, paramsType reflect.Type) {
	handlerValue := reflect.ValueOf(handler)

	// Create params instance
	params := reflect.New(paramsType).Interface()

	// Bind parameters (similar to regular handlers)
	if err := r.bindParameters(&FastContext{
		Request:  sseConn.request,
		Response: sseConn.writer,
		router:   r,
		params:   sseConn.params,
	}, params); err != nil {
		log.Printf("SSE parameter binding error: %v", err)
		return
	}

	// Call handler
	results := handlerValue.Call([]reflect.Value{
		reflect.ValueOf(sseConn),
		reflect.ValueOf(params).Elem(),
	})

	if !results[0].IsNil() {
		log.Printf("SSE handler error: %v", results[0].Interface().(error))
	}
}

// generateAsyncAPIForWS generates the AsyncAPI specification for a WebSocket route and updates the AsyncAPI spec.
// It creates a channel with subscribe and publish operations based on the provided WSHandlerInfo.
func (r *FastRouter) generateAsyncAPIForWS(info *WSHandlerInfo) {
	channel := AsyncAPIChannel{
		Description: info.Description,
		Subscribe: &AsyncAPIOperation{
			Summary:     info.Summary,
			Description: info.Description,
			Tags:        convertToAsyncAPITags(info.Tags),
			Message: AsyncAPIMessage{
				ContentType: "application/json",
				Payload:     r.typeToAsyncAPISchema(info.MessageType),
			},
		},
		Publish: &AsyncAPIOperation{
			Summary:     info.Summary + " Response",
			Description: info.Description,
			Tags:        convertToAsyncAPITags(info.Tags),
			Message: AsyncAPIMessage{
				ContentType: "application/json",
				Payload:     r.typeToAsyncAPISchema(info.ResponseType),
			},
		},
	}

	r.asyncAPISpec.Channels[info.Path] = channel
}

// generateAsyncAPIForSSE generates an AsyncAPI channel configuration for an SSE endpoint based on the provided handler info.
func (r *FastRouter) generateAsyncAPIForSSE(info *SSEHandlerInfo) {
	channel := AsyncAPIChannel{
		Description: info.Description,
		Subscribe: &AsyncAPIOperation{
			Summary:     info.Summary,
			Description: info.Description,
			Tags:        convertToAsyncAPITags(info.Tags),
			Message: AsyncAPIMessage{
				ContentType: "text/event-stream",
				Payload:     r.typeToAsyncAPISchema(reflect.TypeOf(SSEMessage{})),
			},
		},
	}

	r.asyncAPISpec.Channels[info.Path] = channel
}

// typeToAsyncAPISchema converts a given Go reflect.Type to its corresponding AsyncAPISchema representation.
func (r *FastRouter) typeToAsyncAPISchema(t reflect.Type) AsyncAPISchema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return AsyncAPISchema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return AsyncAPISchema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return AsyncAPISchema{Type: "number"}
	case reflect.Bool:
		return AsyncAPISchema{Type: "boolean"}
	case reflect.Slice, reflect.Array:
		return AsyncAPISchema{
			Type:  "array",
			Items: &AsyncAPISchema{Type: "string"}, // Simplified
		}
	case reflect.Struct:
		if t.Name() != "" {
			schemaName := t.Name()
			if _, exists := r.asyncAPISpec.Components.Schemas[schemaName]; !exists {
				r.asyncAPISpec.Components.Schemas[schemaName] = r.generateAsyncAPIStructSchema(t)
			}
			return AsyncAPISchema{Ref: "#/components/schemas/" + schemaName}
		}
		return r.generateAsyncAPIStructSchema(t)
	default:
		return AsyncAPISchema{Type: "object"}
	}
}

// generateAsyncAPIStructSchema generates an AsyncAPISchema for a given Go struct type using reflection.
// It constructs schema properties, required fields, and handles JSON tags, including omitempty handling.
// Fields marked as unexported or explicitly omitted via JSON tags are excluded from the schema.
// The method processes field metadata such as type, description, and JSON serialization behavior.
func (r *FastRouter) generateAsyncAPIStructSchema(t reflect.Type) AsyncAPISchema {
	schema := AsyncAPISchema{
		Type:       "object",
		Properties: make(map[string]AsyncAPISchema),
		Required:   []string{},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		parts := strings.Split(jsonTag, ",")
		fieldName := parts[0]
		if fieldName == "" {
			fieldName = field.Name
		}

		fieldSchema := r.typeToAsyncAPISchema(field.Type)
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema.Description = desc
		}

		schema.Properties[fieldName] = fieldSchema

		omitEmpty := false
		for _, part := range parts[1:] {
			if part == "omitempty" {
				omitEmpty = true
				break
			}
		}

		if !omitEmpty && field.Type.Kind() != reflect.Ptr {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}

// convertToAsyncAPITags converts a slice of strings into a slice of AsyncAPITag structs with names matching the input strings.
func convertToAsyncAPITags(tags []string) []AsyncAPITag {
	var result []AsyncAPITag
	for _, tag := range tags {
		result = append(result, AsyncAPITag{Name: tag})
	}
	return result
}

// EnableAsyncAPI with embedded Studio support
func (r *FastRouter) EnableAsyncAPI() {
	r.initAsyncAPI()

	// JSON endpoint for the spec
	r.GET("/asyncapi", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		json.NewEncoder(w).Encode(r.asyncAPISpec)
	})

	// 	// Embedded AsyncAPI Studio
	// 	r.GET("/asyncapi/studio", func(w http.ResponseWriter, req *http.Request) {
	// 		// Get the current spec
	// 		specJSON, err := json.Marshal(r.asyncAPISpec)
	// 		if err != nil {
	// 			http.Error(w, "Failed to marshal AsyncAPI spec", http.StatusInternalServerError)
	// 			return
	// 		}
	//
	// 		html := fmt.Sprintf(`<!DOCTYPE html>
	// <html>
	// <head>
	//     <title>AsyncAPI Studio - Embedded</title>
	//     <meta charset="utf-8">
	//     <meta name="viewport" content="width=device-width, initial-scale=1">
	//     <style>
	//         * {
	//             margin: 0;
	//             padding: 0;
	//             box-sizing: border-box;
	//         }
	//
	//         body {
	//             font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
	//             background-color: #f5f6fa;
	//             height: 100vh;
	//             overflow: hidden;
	//         }
	//
	//         .header {
	//             background: #2c3e50;
	//             color: white;
	//             padding: 15px 20px;
	//             display: flex;
	//             justify-content: space-between;
	//             align-items: center;
	//             box-shadow: 0 2px 4px rgba(0,0,0,0.1);
	//             z-index: 1000;
	//             position: relative;
	//         }
	//
	//         .header h1 {
	//             font-size: 18px;
	//             font-weight: 600;
	//         }
	//
	//         .header-actions {
	//             display: flex;
	//             gap: 10px;
	//         }
	//
	//         .btn {
	//             padding: 8px 16px;
	//             border: none;
	//             border-radius: 4px;
	//             cursor: pointer;
	//             font-size: 14px;
	//             text-decoration: none;
	//             display: inline-flex;
	//             align-items: center;
	//             gap: 5px;
	//             transition: all 0.2s;
	//         }
	//
	//         .btn-primary {
	//             background: #3498db;
	//             color: white;
	//         }
	//
	//         .btn-primary:hover {
	//             background: #2980b9;
	//         }
	//
	//         .btn-secondary {
	//             background: #95a5a6;
	//             color: white;
	//         }
	//
	//         .btn-secondary:hover {
	//             background: #7f8c8d;
	//         }
	//
	//         .container {
	//             display: flex;
	//             height: calc(100vh - 60px);
	//         }
	//
	//         .sidebar {
	//             width: 300px;
	//             background: white;
	//             border-right: 1px solid #e1e8ed;
	//             overflow-y: auto;
	//             flex-shrink: 0;
	//         }
	//
	//         .main-content {
	//             flex: 1;
	//             background: white;
	//             overflow: hidden;
	//             position: relative;
	//         }
	//
	//         .sidebar-section {
	//             border-bottom: 1px solid #e1e8ed;
	//         }
	//
	//         .sidebar-header {
	//             padding: 15px 20px;
	//             background: #f8f9fa;
	//             font-weight: 600;
	//             font-size: 14px;
	//             color: #2c3e50;
	//         }
	//
	//         .sidebar-content {
	//             padding: 15px 20px;
	//         }
	//
	//         .info-item {
	//             margin-bottom: 12px;
	//         }
	//
	//         .info-label {
	//             font-weight: 600;
	//             color: #7f8c8d;
	//             font-size: 12px;
	//             text-transform: uppercase;
	//             margin-bottom: 4px;
	//         }
	//
	//         .info-value {
	//             color: #2c3e50;
	//             font-size: 14px;
	//         }
	//
	//         .channel-item {
	//             padding: 10px 0;
	//             border-bottom: 1px solid #ecf0f1;
	//         }
	//
	//         .channel-item:last-child {
	//             border-bottom: none;
	//         }
	//
	//         .channel-path {
	//             font-family: 'Courier New', monospace;
	//             font-size: 13px;
	//             color: #e74c3c;
	//             margin-bottom: 4px;
	//         }
	//
	//         .channel-desc {
	//             font-size: 12px;
	//             color: #7f8c8d;
	//         }
	//
	//         .operation-tag {
	//             display: inline-block;
	//             padding: 2px 6px;
	//             border-radius: 3px;
	//             font-size: 11px;
	//             font-weight: 600;
	//             margin-right: 4px;
	//             margin-bottom: 4px;
	//         }
	//
	//         .tag-subscribe {
	//             background: #e8f5e8;
	//             color: #27ae60;
	//         }
	//
	//         .tag-publish {
	//             background: #fff3cd;
	//             color: #f39c12;
	//         }
	//
	//         .asyncapi-viewer {
	//             height: 100%;
	//             border: none;
	//             background: white;
	//         }
	//
	//         .loading {
	//             display: flex;
	//             justify-content: center;
	//             align-items: center;
	//             height: 100%;
	//             color: #7f8c8d;
	//         }
	//
	//         .error {
	//             padding: 20px;
	//             background: #fadbd8;
	//             color: #c0392b;
	//             border-left: 4px solid #e74c3c;
	//         }
	//
	//         .toggle-sidebar {
	//             display: none;
	//         }
	//
	//         @media (max-width: 768px) {
	//             .sidebar {
	//                 position: absolute;
	//                 left: -300px;
	//                 z-index: 1000;
	//                 height: 100%;
	//                 transition: left 0.3s ease;
	//             }
	//
	//             .sidebar.open {
	//                 left: 0;
	//             }
	//
	//             .toggle-sidebar {
	//                 display: inline-flex;
	//             }
	//
	//             .main-content {
	//                 margin-left: 0;
	//             }
	//         }
	//     </style>
	// </head>
	// <body>
	//     <div class="header">
	//         <div style="display: flex; align-items: center; gap: 15px;">
	//             <button class="btn btn-secondary toggle-sidebar" onclick="toggleSidebar()">☰</button>
	//             <h1>AsyncAPI Studio</h1>
	//         </div>
	//         <div class="header-actions">
	//             <button class="btn btn-primary" onclick="refreshSpec()">🔄 Refresh</button>
	//             <a href="/asyncapi" class="btn btn-secondary" target="_blank">📋 View JSON</a>
	//             <a href="/asyncapi/docs" class="btn btn-secondary">📚 Docs</a>
	//         </div>
	//     </div>
	//
	//     <div class="container">
	//         <div class="sidebar" id="sidebar">
	//             <div class="sidebar-section">
	//                 <div class="sidebar-header">API Information</div>
	//                 <div class="sidebar-content" id="api-info">
	//                     <div class="loading">Loading...</div>
	//                 </div>
	//             </div>
	//
	//             <div class="sidebar-section">
	//                 <div class="sidebar-header">Servers</div>
	//                 <div class="sidebar-content" id="servers-info">
	//                     <div class="loading">Loading...</div>
	//                 </div>
	//             </div>
	//
	//             <div class="sidebar-section">
	//                 <div class="sidebar-header">Channels</div>
	//                 <div class="sidebar-content" id="channels-info">
	//                     <div class="loading">Loading...</div>
	//                 </div>
	//             </div>
	//         </div>
	//
	//         <div class="main-content">
	//             <div id="asyncapi-container" class="asyncapi-viewer">
	//                 <div class="loading">Loading AsyncAPI documentation...</div>
	//             </div>
	//         </div>
	//     </div>
	//
	//     <script>
	//         let currentSpec = null;
	//
	//         // Initialize the page
	//         window.onload = function() {
	//             loadSpec();
	//         };
	//
	//         function loadSpec() {
	//             fetch('/asyncapi')
	//                 .then(response => {
	//                     if (!response.ok) {
	//                         throw new Error('Failed to load AsyncAPI specification');
	//                     }
	//                     return response.json();
	//                 })
	//                 .then(spec => {
	//                     currentSpec = spec;
	//                     updateSidebar(spec);
	//                     loadAsyncAPIViewer(spec);
	//                 })
	//                 .catch(error => {
	//                     console.error('Error loading spec:', error);
	//                     document.getElementById('asyncapi-container').innerHTML =
	//                         '<div class="error">Failed to load AsyncAPI specification: ' + error.message + '</div>';
	//                 });
	//         }
	//
	//         function updateSidebar(spec) {
	//             // Update API info
	//             const apiInfo = document.getElementById('api-info');
	//             apiInfo.innerHTML = '';
	//
	//             if (spec.info) {
	//                 apiInfo.innerHTML = `+"`"+`
	//                     <div class="info-item">
	//                         <div class="info-label">Title</div>
	//                         <div class="info-value">${spec.info.title || 'N/A'}</div>
	//                     </div>
	//                     <div class="info-item">
	//                         <div class="info-label">Version</div>
	//                         <div class="info-value">${spec.info.version || 'N/A'}</div>
	//                     </div>
	//                     <div class="info-item">
	//                         <div class="info-label">AsyncAPI Version</div>
	//                         <div class="info-value">${spec.asyncapi || 'N/A'}</div>
	//                     </div>
	//                     ${spec.info.description ? `+"`"+`<div class="info-item">
	//                         <div class="info-label">Description</div>
	//                         <div class="info-value">${spec.info.description}</div>
	//                     </div>`+"`"+` : ''}
	//                 `+"`"+`;
	//             }
	//
	//             // Update servers info
	//             const serversInfo = document.getElementById('servers-info');
	//             serversInfo.innerHTML = '';
	//
	//             if (spec.servers && Object.keys(spec.servers).length > 0) {
	//                 for (const [name, server] of Object.entries(spec.servers)) {
	//                     serversInfo.innerHTML += `+"`"+`
	//                         <div class="info-item">
	//                             <div class="info-label">${name}</div>
	//                             <div class="info-value">${server.url} (${server.protocol})</div>
	//                             ${server.description ? `+"`"+`<div style="font-size: 12px; color: #7f8c8d; margin-top: 2px;">${server.description}</div>`+"`"+` : ''}
	//                         </div>
	//                     `+"`"+`;
	//                 }
	//             } else {
	//                 serversInfo.innerHTML = '<div class="info-value">No servers defined</div>';
	//             }
	//
	//             // Update channels info
	//             const channelsInfo = document.getElementById('channels-info');
	//             channelsInfo.innerHTML = '';
	//
	//             if (spec.channels && Object.keys(spec.channels).length > 0) {
	//                 for (const [path, channel] of Object.entries(spec.channels)) {
	//                     const operations = [];
	//                     if (channel.subscribe) operations.push('<span class="operation-tag tag-subscribe">SUB</span>');
	//                     if (channel.publish) operations.push('<span class="operation-tag tag-publish">PUB</span>');
	//
	//                     channelsInfo.innerHTML += `+"`"+`
	//                         <div class="channel-item">
	//                             <div class="channel-path">${path}</div>
	//                             <div style="margin-bottom: 4px;">${operations.join('')}</div>
	//                             ${channel.description ? `+"`"+`<div class="channel-desc">${channel.description}</div>`+"`"+` : ''}
	//                         </div>
	//                     `+"`"+`;
	//                 }
	//             } else {
	//                 channelsInfo.innerHTML = '<div class="info-value">No channels defined</div>';
	//             }
	//         }
	//
	//         function loadAsyncAPIViewer(spec) {
	//             const container = document.getElementById('asyncapi-container');
	//
	//             // Try to load the AsyncAPI web component
	//             if (window.customElements && window.customElements.define) {
	//                 loadWebComponent(spec, container);
	//             } else {
	//                 // Fallback to simple HTML rendering
	//                 container.innerHTML = generateDetailedHTML(spec);
	//             }
	//         }
	//
	//         function loadWebComponent(spec, container) {
	//             // Load the AsyncAPI web component
	//             const script = document.createElement('script');
	//             script.src = 'https://unpkg.com/@asyncapi/web-component@1.0.0-next.54/lib/asyncapi-web-component.js';
	//             script.onload = function() {
	//                 try {
	//                     container.innerHTML = `+"`"+`<asyncapi-component
	//                         schemaUrl="/asyncapi"
	//                         config='{"show": {"sidebar": false, "info": true, "servers": true, "operations": true, "messages": true, "schemas": true}, "expand": {"messageExamples": true}}'
	//                         cssImportPath="https://unpkg.com/@asyncapi/react-component@1.0.0-next.54/styles/default.min.css">
	//                     </asyncapi-component>`+"`"+`;
	//                 } catch (error) {
	//                     console.error('Web component failed:', error);
	//                     container.innerHTML = generateDetailedHTML(spec);
	//                 }
	//             };
	//             script.onerror = function() {
	//                 console.warn('Failed to load web component, using fallback');
	//                 container.innerHTML = generateDetailedHTML(spec);
	//             };
	//             document.head.appendChild(script);
	//         }
	//
	//         function generateDetailedHTML(spec) {
	//             let html = '<div style="padding: 20px; max-width: 800px; margin: 0 auto;">';
	//
	//             // Channels section with detailed information
	//             if (spec.channels && Object.keys(spec.channels).length > 0) {
	//                 html += '<h2>Channels</h2>';
	//
	//                 for (const [path, channel] of Object.entries(spec.channels)) {
	//                     html += `+"`"+`
	//                         <div style="margin-bottom: 30px; border: 1px solid #e1e8ed; border-radius: 8px; overflow: hidden;">
	//                             <div style="background: #f8f9fa; padding: 15px; border-bottom: 1px solid #e1e8ed;">
	//                                 <h3 style="margin: 0; color: #2c3e50; font-family: 'Courier New', monospace;">${path}</h3>
	//                                 ${channel.description ? `+"`"+`<p style="margin: 5px 0 0 0; color: #7f8c8d;">${channel.description}</p>`+"`"+` : ''}
	//                             </div>
	//                             <div style="padding: 15px;">
	//                     `+"`"+`;
	//
	//                     // Subscribe operation
	//                     if (channel.subscribe) {
	//                         html += `+"`"+`
	//                             <div style="margin-bottom: 20px;">
	//                                 <h4 style="color: #27ae60; margin: 0 0 10px 0;">
	//                                     <span style="background: #e8f5e8; padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: 600;">SUBSCRIBE</span>
	//                                     ${channel.subscribe.summary || 'Subscribe Operation'}
	//                                 </h4>
	//                                 ${channel.subscribe.description ? `+"`"+`<p style="color: #7f8c8d; margin: 5px 0;">${channel.subscribe.description}</p>`+"`"+` : ''}
	//                                 ${channel.subscribe.message ? renderMessage(channel.subscribe.message, 'Incoming Message') : ''}
	//                             </div>
	//                         `+"`"+`;
	//                     }
	//
	//                     // Publish operation
	//                     if (channel.publish) {
	//                         html += `+"`"+`
	//                             <div style="margin-bottom: 20px;">
	//                                 <h4 style="color: #f39c12; margin: 0 0 10px 0;">
	//                                     <span style="background: #fff3cd; padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: 600;">PUBLISH</span>
	//                                     ${channel.publish.summary || 'Publish Operation'}
	//                                 </h4>
	//                                 ${channel.publish.description ? `+"`"+`<p style="color: #7f8c8d; margin: 5px 0;">${channel.publish.description}</p>`+"`"+` : ''}
	//                                 ${channel.publish.message ? renderMessage(channel.publish.message, 'Outgoing Message') : ''}
	//                             </div>
	//                         `+"`"+`;
	//                     }
	//
	//                     html += '</div></div>';
	//                 }
	//             }
	//
	//             // Components section
	//             if (spec.components && spec.components.schemas && Object.keys(spec.components.schemas).length > 0) {
	//                 html += '<h2>Schemas</h2>';
	//                 for (const [name, schema] of Object.entries(spec.components.schemas)) {
	//                     html += `+"`"+`
	//                         <div style="margin-bottom: 20px; border: 1px solid #e1e8ed; border-radius: 8px; overflow: hidden;">
	//                             <div style="background: #f8f9fa; padding: 15px; border-bottom: 1px solid #e1e8ed;">
	//                                 <h3 style="margin: 0; color: #2c3e50;">${name}</h3>
	//                                 ${schema.description ? `+"`"+`<p style="margin: 5px 0 0 0; color: #7f8c8d;">${schema.description}</p>`+"`"+` : ''}
	//                             </div>
	//                             <div style="padding: 15px;">
	//                                 ${renderSchema(schema)}
	//                             </div>
	//                         </div>
	//                     `+"`"+`;
	//                 }
	//             }
	//
	//             html += '</div>';
	//             return html;
	//         }
	//
	//         function renderMessage(message, title) {
	//             let html = `+"`"+`<div style="background: #f8f9fa; padding: 15px; border-radius: 4px; margin-top: 10px;">
	//                 <h5 style="margin: 0 0 10px 0; color: #2c3e50;">${title}</h5>`+"`"+`;
	//
	//             if (message.payload) {
	//                 html += '<div style="margin-top: 10px;"><strong>Payload:</strong></div>';
	//                 html += renderSchema(message.payload);
	//             }
	//
	//             if (message.headers) {
	//                 html += '<div style="margin-top: 10px;"><strong>Headers:</strong></div>';
	//                 html += renderSchema(message.headers);
	//             }
	//
	//             html += '</div>';
	//             return html;
	//         }
	//
	//         function renderSchema(schema) {
	//             if (!schema) return '';
	//
	//             if (schema.$ref) {
	//                 return `+"`"+`<div style="font-family: 'Courier New', monospace; color: #e74c3c;">Reference: ${schema.$ref}</div>`+"`"+`;
	//             }
	//
	//             let html = '';
	//
	//             if (schema.type) {
	//                 html += `+"`"+`<div style="margin-bottom: 10px;"><strong>Type:</strong> <code>${schema.type}</code></div>`+"`"+`;
	//             }
	//
	//             if (schema.properties) {
	//                 html += '<div style="margin-bottom: 10px;"><strong>Properties:</strong></div>';
	//                 html += '<div style="margin-left: 20px;">';
	//
	//                 for (const [propName, propSchema] of Object.entries(schema.properties)) {
	//                     const isRequired = schema.required && schema.required.includes(propName);
	//                     html += `+"`"+`
	//                         <div style="margin-bottom: 8px; padding: 8px; border-left: 3px solid #3498db; background: #f8f9fa;">
	//                             <strong>${propName}</strong>
	//                             ${isRequired ? '<span style="color: #e74c3c;"> *</span>' : ''}
	//                             ${propSchema.type ? `+"`"+` (<code>${propSchema.type}</code>)`+"`"+` : ''}
	//                             ${propSchema.description ? `+"`"+`<br><span style="color: #7f8c8d; font-size: 14px;">${propSchema.description}</span>`+"`"+` : ''}
	//                         </div>
	//                     `+"`"+`;
	//                 }
	//
	//                 html += '</div>';
	//             }
	//
	//             return html;
	//         }
	//
	//         function refreshSpec() {
	//             document.getElementById('asyncapi-container').innerHTML = '<div class="loading">Refreshing...</div>';
	//             loadSpec();
	//         }
	//
	//         function toggleSidebar() {
	//             const sidebar = document.getElementById('sidebar');
	//             sidebar.classList.toggle('open');
	//         }
	//     </script>
	// </body>
	// </html>`, string(specJSON))
	//
	// 		w.Header().Set("Content-Type", "text/html")
	// 		w.Write([]byte(html))
	// 	})

	// Updated main documentation page with Studio link
	r.GET("/asyncapi/docs", func(w http.ResponseWriter, req *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>AsyncAPI Documentation</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 1px solid #eee;
        }
        .options {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .option {
            padding: 25px;
            border: 1px solid #ddd;
            border-radius: 8px;
            text-align: center;
            transition: all 0.3s ease;
        }
        .option:hover {
            box-shadow: 0 4px 8px rgba(0,0,0,0.1);
            transform: translateY(-2px);
        }
        .option.featured {
            border-color: #007bff;
            background: linear-gradient(135deg, #007bff 0%, #0056b3 100%);
            color: white;
        }
        .option.featured .btn {
            background: white;
            color: #007bff;
        }
        .option.featured .btn:hover {
            background: #f8f9fa;
        }
        .option h3 {
            margin-top: 0;
            color: inherit;
        }
        .option p {
            color: inherit;
            opacity: 0.8;
            margin-bottom: 15px;
        }
        .btn {
            display: inline-block;
            padding: 12px 24px;
            background-color: #007bff;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            transition: background-color 0.2s;
            border: none;
            cursor: pointer;
            font-size: 14px;
        }
        .btn:hover {
            background-color: #0056b3;
        }
        .btn-secondary {
            background-color: #6c757d;
        }
        .btn-secondary:hover {
            background-color: #545b62;
        }
        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 15px;
            margin-top: 30px;
        }
        .feature {
            padding: 15px;
            background: #f8f9fa;
            border-radius: 4px;
            border-left: 4px solid #007bff;
        }
        .feature h4 {
            margin: 0 0 8px 0;
            color: #2c3e50;
        }
        .feature p {
            margin: 0;
            color: #7f8c8d;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>AsyncAPI Documentation</h1>
            <p>Comprehensive documentation for WebSocket and Server-Sent Events APIs</p>
        </div>

        <div class="options">
            <div class="option featured">
                <h3>🎨 Embedded Studio</h3>
                <p>Full-featured AsyncAPI Studio embedded directly in your application with syntax highlighting, validation, and interactive documentation.</p>
                <a href="/asyncapi/studio" class="btn">Open Studio</a>
            </div>
            
            <div class="option">
                <h3>🌐 External Studio</h3>
                <p>Open your AsyncAPI specification in the official AsyncAPI Studio hosted online.</p>
                <a href="#" id="external-studio-link" class="btn" target="_blank">Open External Studio</a>
            </div>
            
            <div class="option">
                <h3>📋 Raw Specification</h3>
                <p>View or download the raw AsyncAPI specification in JSON format for integration with other tools.</p>
                <a href="/asyncapi" class="btn btn-secondary" target="_blank">View JSON</a>
            </div>
            
            <div class="option">
                <h3>📖 Simple Documentation</h3>
                <p>Basic HTML documentation with specification preview and debugging information.</p>
                <a href="/asyncapi/simple" class="btn btn-secondary">View Simple Docs</a>
            </div>
        </div>

        <div class="features">
            <div class="feature">
                <h4>🔄 Real-time Updates</h4>
                <p>Documentation automatically reflects changes to your API specification</p>
            </div>
            <div class="feature">
                <h4>🎯 Interactive</h4>
                <p>Test WebSocket and SSE endpoints directly from the documentation</p>
            </div>
            <div class="feature">
                <h4>📱 Responsive</h4>
                <p>Works perfectly on desktop, tablet, and mobile devices</p>
            </div>
            <div class="feature">
                <h4>🔧 Developer-Friendly</h4>
                <p>Multiple viewing options to suit different workflows and preferences</p>
            </div>
        </div>
    </div>

    <script>
        // Set up external studio link
        const externalLink = document.getElementById('external-studio-link');
        const specUrl = encodeURIComponent(window.location.origin + '/asyncapi');
        externalLink.href = 'https://studio.asyncapi.com/?url=' + specUrl;
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// Simple docs endpoint (keeping the existing one with minor updates)
	r.GET("/asyncapi/simple", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		// Get the spec
		specJSON, err := json.MarshalIndent(r.asyncAPISpec, "", "  ")
		if err != nil {
			http.Error(w, "Failed to marshal AsyncAPI spec", http.StatusInternalServerError)
			return
		}

		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>AsyncAPI Specification</title>
    <style>
        body { font-family: monospace; margin: 20px; }
        pre { background: #f5f5f5; padding: 20px; border-radius: 4px; overflow-x: auto; }
        .header { margin-bottom: 20px; }
        .links { margin-bottom: 20px; }
        .links a { margin-right: 15px; padding: 8px 16px; background: #007bff; color: white; text-decoration: none; border-radius: 4px; }
        .links a:hover { background: #0056b3; }
    </style>
</head>
<body>
    <div class="header">
        <h1>AsyncAPI Specification</h1>
        <div class="links">
            <a href="/asyncapi">JSON Format</a>
            <a href="/asyncapi/docs">Documentation Hub</a>
            <a href="/asyncapi/studio">Embedded Studio</a>
            <a href="https://studio.asyncapi.com/?url=%s" target="_blank">External Studio</a>
        </div>
    </div>
    <pre><code>%s</code></pre>
</body>
</html>`,
			req.Host+"/asyncapi",
			string(specJSON))

		w.Write([]byte(html))
	})
}

// AsyncHandlerOption defines an interface for applying settings to WebSocket and SSE handler information structs.
// ApplyToWS applies the option to a WSHandlerInfo instance.
// ApplyToSSE applies the option to an SSEHandlerInfo instance.
type AsyncHandlerOption interface {
	ApplyToWS(*WSHandlerInfo)
	ApplyToSSE(*SSEHandlerInfo)
}

// asyncSummaryOption is a type that implements AsyncHandlerOption to set a summary for WebSocket or SSE handlers.
type asyncSummaryOption struct {
	summary string
}

// ApplyToWS sets the summary field of the provided WSHandlerInfo instance to the value of the asyncSummaryOption summary.
func (o asyncSummaryOption) ApplyToWS(info *WSHandlerInfo) {
	info.Summary = o.summary
}

// ApplyToSSE sets the Summary field of the provided SSEHandlerInfo instance to the summary value of the asyncSummaryOption.
func (o asyncSummaryOption) ApplyToSSE(info *SSEHandlerInfo) {
	info.Summary = o.summary
}

// asyncDescriptionOption is a struct that holds a description used to configure asynchronous handler behavior.
type asyncDescriptionOption struct {
	description string
}

// ApplyToWS updates the Description field of the provided WSHandlerInfo with the value from asyncDescriptionOption.
func (o asyncDescriptionOption) ApplyToWS(info *WSHandlerInfo) {
	info.Description = o.description
}

// ApplyToSSE sets the description field of the given SSEHandlerInfo instance to the value of the asyncDescriptionOption.
func (o asyncDescriptionOption) ApplyToSSE(info *SSEHandlerInfo) {
	info.Description = o.description
}

// asyncTagsOption is a type that holds a list of tags to be applied to asynchronous handler metadata.
type asyncTagsOption struct {
	tags []string
}

// ApplyToWS applies the tags from asyncTagsOption to the WSHandlerInfo instance.
func (o asyncTagsOption) ApplyToWS(info *WSHandlerInfo) {
	info.Tags = o.tags
}

// ApplyToSSE applies the tags from asyncTagsOption to the SSEHandlerInfo instance.
func (o asyncTagsOption) ApplyToSSE(info *SSEHandlerInfo) {
	info.Tags = o.tags
}

// WithAsyncSummary sets a summary description for an asynchronous handler and returns an AsyncHandlerOption.
func WithAsyncSummary(summary string) AsyncHandlerOption {
	return asyncSummaryOption{summary: summary}
}

// WithAsyncDescription sets a description for an asynchronous handler, applied to WebSocket or SSE handlers.
func WithAsyncDescription(description string) AsyncHandlerOption {
	return asyncDescriptionOption{description: description}
}

// WithAsyncTags specifies tags to be applied to WebSocket (WS) and Server-Sent Events (SSE) handler information.
func WithAsyncTags(tags ...string) AsyncHandlerOption {
	return asyncTagsOption{tags: tags}
}

// generateClientID generates a unique client identifier based on the current timestamp in nanoseconds.
func generateClientID() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback for the rare case that crypto/rand fails
		return fmt.Sprintf("client_%d", time.Now().UnixNano())
	}
	return "client_" + hex.EncodeToString(b)
}

// ConnectionManager returns the ConnectionManager instance used to manage WebSocket and SSE connections.
func (r *FastRouter) ConnectionManager() *ConnectionManager {
	r.initAsyncAPI()
	return r.connectionManager
}
