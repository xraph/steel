package forge_router

import (
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"github.com/gorilla/websocket"
)

// WSConnection represents a WebSocket connection with additional utilities like parameter handling and metadata storage.
type WSConnection struct {
	conn     *websocket.Conn
	router   *FastRouter
	params   *Params
	request  *http.Request
	clientID string
	metadata map[string]interface{}
	mu       sync.RWMutex
}

// WSMessage represents a WebSocket message containing a type, payload, optional ID, and optional error.
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	ID      string      `json:"id,omitempty"`
	Error   *WSError    `json:"error,omitempty"`
}

// WSError represents an error structure used in WebSocket communication.
// It contains an error code, a descriptive message, and optional details.
type WSError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// WSHandler defines a function type for handling WebSocket messages and generating responses.
// It processes incoming messages using the provided WSConnection and returns a response or an error.
type WSHandler[TMessage any, TResponse any] func(conn *WSConnection, message TMessage) (*TResponse, error)

// WSHandlerInfo defines metadata for a WebSocket handler, including path, message type, response type, and related details.
type WSHandlerInfo struct {
	Path         string
	MessageType  reflect.Type
	ResponseType reflect.Type
	Handler      interface{}
	Summary      string
	Description  string
	Tags         []string
}

// SendMessage sends a WSMessage to the WebSocket connection in a thread-safe manner using a mutex lock.
func (ws *WSConnection) SendMessage(message WSMessage) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if ws.conn == nil {
		return fmt.Errorf("websocket connection is nil")
	}
	return ws.conn.WriteJSON(message)
}

// ReadMessage reads a JSON-formatted message from the WebSocket connection and returns it as a WSMessage, or an error.
func (ws *WSConnection) ReadMessage() (WSMessage, error) {
	var message WSMessage
	err := ws.conn.ReadJSON(&message)
	return message, err
}

// Close safely closes the underlying WebSocket connection, ensuring thread-safety by locking the mutex during operation.
func (ws *WSConnection) Close() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.conn.Close()
}

// Param retrieves the value associated with the given key from the connection's parameters.
func (ws *WSConnection) Param(key string) string {
	return ws.params.Get(key)
}

// SetMetadata sets a key-value pair in the metadata map of the WSConnection instance in a thread-safe manner.
func (ws *WSConnection) SetMetadata(key string, value interface{}) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if ws.metadata == nil {
		ws.metadata = make(map[string]interface{})
	}
	ws.metadata[key] = value
}

// GetMetadata retrieves the value associated with the given key from the metadata.
// It returns the value and a boolean indicating if the key exists.
func (ws *WSConnection) GetMetadata(key string) (interface{}, bool) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	if ws.metadata == nil {
		return nil, false
	}
	val, ok := ws.metadata[key]
	return val, ok
}

func (ws *WSConnection) Request() *http.Request {
	return ws.request
}
