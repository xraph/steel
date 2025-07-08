package forgerouter

import (
	"fmt"
	"net/http"
	"reflect"
	"sync"

	json "github.com/json-iterator/go"
)

// SSEConnection represents a Server-Sent Events (SSE) connection, enabling bi-directional communication with a client.
// It manages request-specific data, connection status, and metadata for each client.
// The type is thread-safe, allowing concurrent operations across multiple goroutines.
type SSEConnection struct {
	writer   http.ResponseWriter
	router   *FastRouter
	params   *Params
	request  *http.Request
	clientID string
	metadata map[string]interface{}
	mu       sync.RWMutex
	closed   bool
}

// SSEMessage represents a single message sent over a Server-Sent Events (SSE) connection.
type SSEMessage struct {
	ID    string      `json:"id,omitempty"`
	Event string      `json:"event,omitempty"`
	Data  interface{} `json:"data"`
	Retry int         `json:"retry,omitempty"`
}

// SSEHandler defines a function type for handling SSE connections with typed parameters.
// Takes an SSEConnection and a parameter of any type, and returns an error.
type SSEHandler[TParams any] func(conn *SSEConnection, params TParams) error

// SSEHandlerInfo represents metadata and configuration for an SSE (Server-Sent Events) endpoint handler.
type SSEHandlerInfo struct {
	Path        string
	ParamsType  reflect.Type
	Handler     interface{}
	Summary     string
	Description string
	Tags        []string
}

// SendMessage sends a Server-Sent Event (SSE) message to the client through the current connection.
// It includes optional fields such as ID, event name, retry interval, and data payload.
// Returns an error if the connection is closed or data serialization fails.
func (sse *SSEConnection) SendMessage(message SSEMessage) error {
	sse.mu.Lock()
	defer sse.mu.Unlock()

	if sse.writer == nil {
		return fmt.Errorf("http writer is nil")
	}

	if sse.closed {
		return fmt.Errorf("connection closed")
	}

	if message.ID != "" {
		fmt.Fprintf(sse.writer, "id: %s\n", message.ID)
	}
	if message.Event != "" {
		fmt.Fprintf(sse.writer, "event: %s\n", message.Event)
	}
	if message.Retry > 0 {
		fmt.Fprintf(sse.writer, "retry: %d\n", message.Retry)
	}

	data, err := json.Marshal(message.Data)
	if err != nil {
		return err
	}

	fmt.Fprintf(sse.writer, "data: %s\n\n", string(data))

	if flusher, ok := sse.writer.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// Close gracefully shuts down the SSE connection by marking it as closed and ensuring thread-safety with a mutex lock.
func (sse *SSEConnection) Close() {
	sse.mu.Lock()
	defer sse.mu.Unlock()
	sse.closed = true
}

// IsClosed checks whether the SSE connection has been closed. Returns true if the connection is closed, otherwise false.
func (sse *SSEConnection) IsClosed() bool {
	sse.mu.RLock()
	defer sse.mu.RUnlock()
	return sse.closed
}

// Param retrieves the value associated with the given key from the connection's query parameters.
func (sse *SSEConnection) Param(key string) string {
	return sse.params.Get(key)
}

// SetMetadata sets a metadata key-value pair on the SSEConnection, initializing the map if it is nil.
func (sse *SSEConnection) SetMetadata(key string, value interface{}) {
	sse.mu.Lock()
	defer sse.mu.Unlock()
	if sse.metadata == nil {
		sse.metadata = make(map[string]interface{})
	}
	sse.metadata[key] = value
}

// GetMetadata retrieves the metadata value associated with the specified key if it exists. It returns the value and a boolean.
func (sse *SSEConnection) GetMetadata(key string) (interface{}, bool) {
	sse.mu.RLock()
	defer sse.mu.RUnlock()
	if sse.metadata == nil {
		return nil, false
	}
	val, ok := sse.metadata[key]
	return val, ok
}

func (sse *SSEConnection) Request() *http.Request {
	return sse.request
}
