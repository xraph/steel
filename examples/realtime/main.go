package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/xraph/forgerouter"
)

// ChatMessage Example message types for WebSocket
type ChatMessage struct {
	UserID    string    `json:"user_id" description:"User identifier"`
	RoomID    string    `json:"room_id" description:"Chat room identifier"`
	Message   string    `json:"message" description:"Chat message content"`
	Timestamp time.Time `json:"timestamp" description:"Message timestamp"`
}

type ChatResponse struct {
	MessageID string    `json:"message_id" description:"Unique message identifier"`
	Status    string    `json:"status" description:"Message status"`
	Timestamp time.Time `json:"timestamp" description:"Response timestamp"`
}

// NotificationParams Example types for SSE
type NotificationParams struct {
	UserID string `path:"user_id" description:"User ID to send notifications to"`
	Topics string `query:"topics" description:"Comma-separated list of topics to subscribe to"`
}

type NotificationMessage struct {
	ID      string    `json:"id" description:"Notification ID"`
	Type    string    `json:"type" description:"Notification type"`
	Title   string    `json:"title" description:"Notification title"`
	Content string    `json:"content" description:"Notification content"`
	UserID  string    `json:"user_id" description:"Target user ID"`
	Created time.Time `json:"created" description:"Creation timestamp"`
}

func main() {
	router := forge_router.NewRouter()

	// Add debug logging to see what's happening
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("DEBUG: Request %s %s\n", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// Enable both OpenAPI and AsyncAPI documentation
	router.EnableOpenAPI()
	router.EnableAsyncAPI()

	// Standard middleware
	router.Use(forge_router.Logger)
	router.Use(forge_router.Recoverer)

	// WebSocket chat endpoint
	router.WebSocket("/ws/chat/{room_id}", func(conn *forge_router.WSConnection, message ChatMessage) (*ChatResponse, error) {
		log.Printf("Received chat message from %s in room %s: %s",
			message.UserID, message.RoomID, message.Message)

		// Set user metadata
		conn.SetMetadata("user_id", message.UserID)
		conn.SetMetadata("room_id", message.RoomID)

		// Broadcast message to all connections in the same room
		broadcastMessage := forge_router.WSMessage{
			Type: "chat_message",
			Payload: map[string]interface{}{
				"user_id":   message.UserID,
				"room_id":   message.RoomID,
				"message":   message.Message,
				"timestamp": message.Timestamp,
			},
		}

		// Get all connections and broadcast to same room
		cm := router.ConnectionManager()
		for _, wsConn := range cm.WSConnections() {
			if roomID, ok := wsConn.GetMetadata("room_id"); ok && roomID == message.RoomID {
				wsConn.SendMessage(broadcastMessage)
			}
		}

		return &ChatResponse{
			MessageID: fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			Status:    "delivered",
			Timestamp: time.Now(),
		}, nil
	},
		forge_router.WithAsyncSummary("Real-time chat messaging"),
		forge_router.WithAsyncDescription("WebSocket endpoint for real-time chat in rooms"),
		forge_router.WithAsyncTags("chat", "websocket", "real-time"),
	)

	// SSE notifications endpoint
	router.SSE("/sse/notifications/{user_id}", func(conn *forge_router.SSEConnection, params NotificationParams) error {
		log.Printf("Starting SSE for user %s, topics: %s", params.UserID, params.Topics)

		// Set user metadata
		conn.SetMetadata("user_id", params.UserID)
		conn.SetMetadata("topics", params.Topics)

		// Send initial connection message
		conn.SendMessage(forge_router.SSEMessage{
			Event: "connected",
			Data: map[string]interface{}{
				"message": "Connected to notification stream",
				"user_id": params.UserID,
				"topics":  params.Topics,
			},
		})

		// Keep connection alive and send periodic heartbeats
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		// Listen for connection close
		done := make(chan bool)
		go func() {
			select {
			case <-conn.Request().Context().Done():
				done <- true
			}
		}()

		for {
			select {
			case <-ticker.C:
				if conn.IsClosed() {
					return nil
				}
				conn.SendMessage(forge_router.SSEMessage{
					Event: "heartbeat",
					Data: map[string]interface{}{
						"timestamp": time.Now(),
					},
				})
			case <-done:
				log.Printf("SSE connection closed for user %s", params.UserID)
				return nil
			}
		}
	},
		forge_router.WithAsyncSummary("Real-time notifications"),
		forge_router.WithAsyncDescription("Server-sent events for real-time notifications"),
		forge_router.WithAsyncTags("notifications", "sse", "real-time"),
	)

	// REST API endpoint to send notifications
	router.OpinionatedPOST("/api/notifications", func(ctx *forge_router.FastContext, input struct {
		UserID  string `json:"user_id" description:"Target user ID"`
		Type    string `json:"type" description:"Notification type"`
		Title   string `json:"title" description:"Notification title"`
		Content string `json:"content" description:"Notification content"`
	}) (*struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}, error) {
		// Create notification message
		notification := NotificationMessage{
			ID:      fmt.Sprintf("notif_%d", time.Now().UnixNano()),
			Type:    input.Type,
			Title:   input.Title,
			Content: input.Content,
			UserID:  input.UserID,
			Created: time.Now(),
		}

		// Send to SSE connections for this user
		cm := router.ConnectionManager()
		sent := false
		for _, sseConn := range cm.SSEConnections() {
			if userID, ok := sseConn.GetMetadata("user_id"); ok && userID == input.UserID {
				sseConn.SendMessage(forge_router.SSEMessage{
					Event: "notification",
					Data:  notification,
				})
				sent = true
			}
		}

		if !sent {
			return nil, ctx.NotFound("User not connected or not found")
		}

		return &struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}{
			Success: true,
			Message: "Notification sent successfully",
		}, nil
	},
		forge_router.WithSummary("Send notification to user"),
		forge_router.WithDescription("Send a notification to a specific user via SSE"),
		forge_router.WithTags("notifications", "api"),
	)

	// WebSocket endpoint for system-wide broadcasts
	router.WebSocket("/ws/system", func(conn *forge_router.WSConnection, message struct {
		Type    string      `json:"type" description:"Message type"`
		Payload interface{} `json:"payload" description:"Message payload"`
	}) (*struct {
		Acknowledged bool      `json:"acknowledged"`
		Timestamp    time.Time `json:"timestamp"`
	}, error) {
		log.Printf("System message: %s", message.Type)

		// Broadcast to all connected clients
		broadcastMsg := forge_router.WSMessage{
			Type:    "system_broadcast",
			Payload: message.Payload,
		}

		router.ConnectionManager().BroadcastWS(broadcastMsg)

		return &struct {
			Acknowledged bool      `json:"acknowledged"`
			Timestamp    time.Time `json:"timestamp"`
		}{
			Acknowledged: true,
			Timestamp:    time.Now(),
		}, nil
	},
		forge_router.WithAsyncSummary("System-wide broadcasts"),
		forge_router.WithAsyncDescription("WebSocket for system-wide message broadcasting"),
		forge_router.WithAsyncTags("system", "broadcast", "websocket"),
	)

	// API endpoint for system broadcasts
	router.OpinionatedPOST("/api/system/broadcast", func(ctx *forge_router.FastContext, input struct {
		Type    string      `json:"type" description:"Broadcast type"`
		Message interface{} `json:"message" description:"Broadcast message"`
	}) (*struct {
		Success   bool `json:"success"`
		Receivers int  `json:"receivers"`
	}, error) {
		// Broadcast via WebSocket
		wsMessage := forge_router.WSMessage{
			Type:    "system_broadcast",
			Payload: input.Message,
		}
		router.ConnectionManager().BroadcastWS(wsMessage)

		// Broadcast via SSE
		sseMessage := forge_router.SSEMessage{
			Event: "system_broadcast",
			Data: map[string]interface{}{
				"type":    input.Type,
				"message": input.Message,
			},
		}
		router.ConnectionManager().BroadcastSSE(sseMessage)

		// Count receivers
		cm := router.ConnectionManager()
		receivers := len(cm.WSConnections()) + len(cm.SSEConnections())

		return &struct {
			Success   bool `json:"success"`
			Receivers int  `json:"receivers"`
		}{
			Success:   true,
			Receivers: receivers,
		}, nil
	},
		forge_router.WithSummary("System-wide broadcast"),
		forge_router.WithDescription("Send a message to all connected clients via WebSocket and SSE"),
		forge_router.WithTags("system", "broadcast", "api"),
	)

	// Health check endpoint
	router.OpinionatedGET("/health", func(ctx *forge_router.FastContext, input struct{}) (*struct {
		Status      string    `json:"status"`
		Timestamp   time.Time `json:"timestamp"`
		Connections struct {
			WebSocket int `json:"websocket"`
			SSE       int `json:"sse"`
		} `json:"connections"`
	}, error) {
		cm := router.ConnectionManager()
		wsCount := len(cm.WSConnections())
		sseCount := len(cm.SSEConnections())

		return &struct {
			Status      string    `json:"status"`
			Timestamp   time.Time `json:"timestamp"`
			Connections struct {
				WebSocket int `json:"websocket"`
				SSE       int `json:"sse"`
			} `json:"connections"`
		}{
			Status:    "healthy",
			Timestamp: time.Now(),
			Connections: struct {
				WebSocket int `json:"websocket"`
				SSE       int `json:"sse"`
			}{
				WebSocket: wsCount,
				SSE:       sseCount,
			},
		}, nil
	}, forge_router.WithSummary("Health check with connection stats"))

	// Serve the beautiful new interface
	router.GET("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>FastRouter Async Demo</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        primary: '#3B82F6',
                        secondary: '#10B981',
                        accent: '#F59E0B',
                        danger: '#EF4444',
                    }
                }
            }
        }
    </script>
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
        body { font-family: 'Inter', sans-serif; }
        
        .message-fade-in {
            animation: fadeIn 0.3s ease-in-out;
        }
        
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }
        
        .pulse-dot {
            animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
        }
        
        .status-indicator {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            margin-right: 8px;
            display: inline-block;
        }
        
        .status-connected { background-color: #10B981; }
        .status-disconnected { background-color: #6B7280; }
        .status-error { background-color: #EF4444; }
        .status-connecting { background-color: #F59E0B; }
        
        .glass-effect {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255, 255, 255, 0.2);
        }
        
        .custom-scrollbar::-webkit-scrollbar {
            width: 6px;
        }
        
        .custom-scrollbar::-webkit-scrollbar-track {
            background: #f1f1f1;
            border-radius: 10px;
        }
        
        .custom-scrollbar::-webkit-scrollbar-thumb {
            background: #c1c1c1;
            border-radius: 10px;
        }
        
        .custom-scrollbar::-webkit-scrollbar-thumb:hover {
            background: #a8a8a8;
        }
    </style>
</head>
<body class="bg-gradient-to-br from-blue-50 via-indigo-50 to-purple-50 min-h-screen">
    <!-- Header -->
    <header class="bg-white/80 backdrop-blur-sm border-b border-gray-200 sticky top-0 z-50">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div class="flex justify-between items-center h-16">
                <div class="flex items-center">
                    <div class="flex-shrink-0">
                        <h1 class="text-2xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                            FastRouter Demo
                        </h1>
                    </div>
                </div>
                <div class="flex items-center space-x-4">
                    <a href="/openapi/swagger" target="_blank" 
                       class="text-sm font-medium text-gray-700 hover:text-blue-600 transition-colors">
                        OpenAPI
                    </a>
                    <a href="/asyncapi/docs" target="_blank" 
                       class="text-sm font-medium text-gray-700 hover:text-blue-600 transition-colors">
                        AsyncAPI
                    </a>
                </div>
            </div>
        </div>
    </header>

    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <!-- Grid Layout -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
            <!-- WebSocket Chat Section -->
            <div class="bg-white rounded-xl shadow-lg border border-gray-200 overflow-hidden">
                <div class="bg-gradient-to-r from-blue-600 to-blue-700 px-6 py-4">
                    <h2 class="text-xl font-semibold text-white flex items-center">
                        <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"/>
                        </svg>
                        WebSocket Chat
                    </h2>
                </div>
                
                <div class="p-6">
                    <!-- Connection Controls -->
                    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
                        <div>
                            <label class="block text-sm font-medium text-gray-700 mb-2">Room ID</label>
                            <input type="text" id="roomId" value="general" 
                                   class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors">
                        </div>
                        <div>
                            <label class="block text-sm font-medium text-gray-700 mb-2">User ID</label>
                            <input type="text" id="userId" value="user1" 
                                   class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors">
                        </div>
                    </div>
                    
                    <!-- Connection Status & Buttons -->
                    <div class="flex items-center justify-between mb-4">
                        <div class="flex items-center">
                            <span class="status-indicator status-disconnected" id="wsStatusIndicator"></span>
                            <span class="text-sm font-medium text-gray-700" id="wsStatus">Disconnected</span>
                        </div>
                        <div class="flex space-x-2">
                            <button onclick="connectWebSocket()" 
                                    class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium">
                                Connect
                            </button>
                            <button onclick="disconnectWebSocket()" 
                                    class="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors text-sm font-medium">
                                Disconnect
                            </button>
                        </div>
                    </div>
                    
                    <!-- Messages Display -->
                    <div class="bg-gray-50 rounded-lg p-4 h-64 overflow-y-auto custom-scrollbar mb-4" id="wsMessages">
                        <div class="text-gray-500 text-sm text-center py-8">
                            No messages yet. Connect to start chatting!
                        </div>
                    </div>
                    
                    <!-- Message Input -->
                    <div class="flex space-x-2">
                        <textarea id="wsMessage" placeholder="Type your message..." 
                                  class="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none h-12 transition-colors"></textarea>
                        <button onclick="sendWebSocketMessage()" 
                                class="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium">
                            Send
                        </button>
                    </div>
                </div>
            </div>
            
            <!-- Server-Sent Events Section -->
            <div class="bg-white rounded-xl shadow-lg border border-gray-200 overflow-hidden">
                <div class="bg-gradient-to-r from-green-600 to-green-700 px-6 py-4">
                    <h2 class="text-xl font-semibold text-white flex items-center">
                        <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-5 5v-5zM12 17h1M8 17h1m-8 0h6m12-9v-1a2 2 0 00-2-2H5a2 2 0 00-2 2v1m16 0v6a2 2 0 01-2 2h-2"/>
                        </svg>
                        Server-Sent Events
                    </h2>
                </div>
                
                <div class="p-6">
                    <!-- SSE Controls -->
                    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
                        <div>
                            <label class="block text-sm font-medium text-gray-700 mb-2">User ID</label>
                            <input type="text" id="sseUserId" value="user1" 
                                   class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-transparent transition-colors">
                        </div>
                        <div>
                            <label class="block text-sm font-medium text-gray-700 mb-2">Topics</label>
                            <input type="text" id="sseTopics" value="general,alerts" 
                                   class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-transparent transition-colors">
                        </div>
                    </div>
                    
                    <!-- SSE Status & Buttons -->
                    <div class="flex items-center justify-between mb-4">
                        <div class="flex items-center">
                            <span class="status-indicator status-disconnected" id="sseStatusIndicator"></span>
                            <span class="text-sm font-medium text-gray-700" id="sseStatus">Disconnected</span>
                        </div>
                        <div class="flex space-x-2">
                            <button onclick="connectSSE()" 
                                    class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors text-sm font-medium">
                                Connect
                            </button>
                            <button onclick="disconnectSSE()" 
                                    class="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors text-sm font-medium">
                                Disconnect
                            </button>
                        </div>
                    </div>
                    
                    <!-- SSE Messages Display -->
                    <div class="bg-gray-50 rounded-lg p-4 h-64 overflow-y-auto custom-scrollbar mb-4" id="sseMessages">
                        <div class="text-gray-500 text-sm text-center py-8">
                            No events yet. Connect to start receiving notifications!
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- Notification Sender Section -->
        <div class="mt-8 bg-white rounded-xl shadow-lg border border-gray-200 overflow-hidden">
            <div class="bg-gradient-to-r from-yellow-500 to-orange-500 px-6 py-4">
                <h2 class="text-xl font-semibold text-white flex items-center">
                    <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-5 5v-5zM4 5h7m0 0v12M8 17l4-4-4-4"/>
                    </svg>
                    Send Notification
                </h2>
            </div>
            
            <div class="p-6">
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-4">
                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-2">User ID</label>
                        <input type="text" id="notifUserId" value="user1" 
                               class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-yellow-500 focus:border-transparent transition-colors">
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-2">Type</label>
                        <select id="notifType" class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-yellow-500 focus:border-transparent transition-colors">
                            <option value="info">Info</option>
                            <option value="success">Success</option>
                            <option value="warning">Warning</option>
                            <option value="error">Error</option>
                        </select>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-2">Title</label>
                        <input type="text" id="notifTitle" value="Test Notification" 
                               class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-yellow-500 focus:border-transparent transition-colors">
                    </div>
                    <div class="flex items-end">
                        <button onclick="sendNotification()" 
                                class="w-full px-4 py-2 bg-yellow-500 text-white rounded-lg hover:bg-yellow-600 transition-colors font-medium">
                            Send Notification
                        </button>
                    </div>
                </div>
                
                <div>
                    <label class="block text-sm font-medium text-gray-700 mb-2">Content</label>
                    <textarea id="notifContent" placeholder="Notification content..." 
                              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-yellow-500 focus:border-transparent resize-none h-20 transition-colors">This is a test notification</textarea>
                </div>
            </div>
        </div>
    </main>

    <script>
        let ws = null;
        let eventSource = null;

        // WebSocket Functions
        function connectWebSocket() {
            const roomId = document.getElementById('roomId').value;
            const userId = document.getElementById('userId').value;
            
            if (!roomId || !userId) {
                showAlert('Please enter both Room ID and User ID', 'error');
                return;
            }
            
            updateWSStatus('connecting', 'Connecting...');
            
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const host = window.location.host;
            const wsUrl = protocol + '//' + host + '/ws/chat/' + roomId;
            
            ws = new WebSocket(wsUrl);
            
            ws.onopen = function() {
                updateWSStatus('connected', 'Connected');
                addMessage('wsMessages', 'Connected to WebSocket', 'system');
                clearEmptyState('wsMessages');
            };
            
            ws.onmessage = function(event) {
                const data = JSON.parse(event.data);
                
                if (data.type === 'chat_message') {
                    addMessage('wsMessages', data.payload.user_id + ': ' + data.payload.message, 'chat');
                } else if (data.type === 'response') {
                    addMessage('wsMessages', 'Message delivered (' + data.payload.status + ')', 'success');
                } else {
                    addMessage('wsMessages', 'Received: ' + JSON.stringify(data), 'info');
                }
            };
            
            ws.onclose = function() {
                updateWSStatus('disconnected', 'Disconnected');
                addMessage('wsMessages', 'WebSocket connection closed', 'system');
            };
            
            ws.onerror = function(error) {
                updateWSStatus('error', 'Connection Error');
                addMessage('wsMessages', 'WebSocket error occurred', 'error');
            };
        }

        function disconnectWebSocket() {
            if (ws) {
                ws.close();
                ws = null;
            }
        }

        function sendWebSocketMessage() {
            if (!ws || ws.readyState !== WebSocket.OPEN) {
                showAlert('WebSocket is not connected', 'error');
                return;
            }
            
            const roomId = document.getElementById('roomId').value;
            const userId = document.getElementById('userId').value;
            const message = document.getElementById('wsMessage').value.trim();
            
            if (!message) {
                showAlert('Please enter a message', 'warning');
                return;
            }
            
            const wsMessage = {
                type: 'chat_message',
                payload: {
                    user_id: userId,
                    room_id: roomId,
                    message: message,
                    timestamp: new Date().toISOString()
                }
            };
            
            ws.send(JSON.stringify(wsMessage));
            document.getElementById('wsMessage').value = '';
            
            // Show sent message immediately
            addMessage('wsMessages', 'You: ' + message, 'sent');
        }

        // SSE Functions
        function connectSSE() {
            const userId = document.getElementById('sseUserId').value;
            const topics = document.getElementById('sseTopics').value;
            
            if (!userId) {
                showAlert('Please enter User ID', 'error');
                return;
            }
            
            updateSSEStatus('connecting', 'Connecting...');
            
            const sseUrl = '/sse/notifications/' + userId + '?topics=' + topics;
            eventSource = new EventSource(sseUrl);
            
            eventSource.onopen = function() {
                updateSSEStatus('connected', 'Connected');
                addMessage('sseMessages', 'Connected to SSE', 'system');
                clearEmptyState('sseMessages');
            };
            
            eventSource.onmessage = function(event) {
                const data = JSON.parse(event.data);
                addMessage('sseMessages', 'Message: ' + JSON.stringify(data), 'info');
            };
            
            eventSource.addEventListener('connected', function(event) {
                const data = JSON.parse(event.data);
                addMessage('sseMessages', 'Connected to topics: ' + data.topics, 'success');
            });
            
            eventSource.addEventListener('notification', function(event) {
                const data = JSON.parse(event.data);
                addMessage('sseMessages', '📢 ' + data.title + ': ' + data.content, 'notification');
            });
            
            eventSource.addEventListener('heartbeat', function(event) {
                addMessage('sseMessages', '💓 Heartbeat', 'heartbeat');
            });
            
            eventSource.onerror = function(event) {
                updateSSEStatus('error', 'Connection Error');
                addMessage('sseMessages', 'SSE connection error', 'error');
            };
        }

        function disconnectSSE() {
            if (eventSource) {
                eventSource.close();
                eventSource = null;
                updateSSEStatus('disconnected', 'Disconnected');
            }
        }

        function sendNotification() {
            const payload = {
                user_id: document.getElementById('notifUserId').value,
                type: document.getElementById('notifType').value,
                title: document.getElementById('notifTitle').value,
                content: document.getElementById('notifContent').value
            };
            
            if (!payload.user_id || !payload.title || !payload.content) {
                showAlert('Please fill in all fields', 'error');
                return;
            }
            
            fetch('/api/notifications', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(payload)
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error('HTTP ' + response.status);
                }
                return response.json();
            })
            .then(data => {
                showAlert('Notification sent successfully!', 'success');
            })
            .catch(error => {
                showAlert('Error sending notification: ' + error.message, 'error');
            });
        }

        // Helper Functions
        function updateWSStatus(status, text) {
            const indicator = document.getElementById('wsStatusIndicator');
            const statusText = document.getElementById('wsStatus');
            
            indicator.className = 'status-indicator status-' + status;
            if (status === 'connecting') {
                indicator.classList.add('pulse-dot');
            } else {
                indicator.classList.remove('pulse-dot');
            }
            
            statusText.textContent = text;
        }

        function updateSSEStatus(status, text) {
            const indicator = document.getElementById('sseStatusIndicator');
            const statusText = document.getElementById('sseStatus');
            
            indicator.className = 'status-indicator status-' + status;
            if (status === 'connecting') {
                indicator.classList.add('pulse-dot');
            } else {
                indicator.classList.remove('pulse-dot');
            }
            
            statusText.textContent = text;
        }

        function addMessage(containerId, message, type) {
            const container = document.getElementById(containerId);
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message-fade-in mb-2 p-2 rounded-lg text-sm';
            
            const timestamp = new Date().toLocaleTimeString();
            
            let className = '';
            let icon = '';
            
            switch (type) {
                case 'system':
                    className = 'bg-gray-100 text-gray-800 border-l-4 border-gray-400';
                    icon = 'ℹ️';
                    break;
                case 'chat':
                    className = 'bg-blue-50 text-blue-800 border-l-4 border-blue-400';
                    icon = '💬';
                    break;
                case 'sent':
                    className = 'bg-green-50 text-green-800 border-l-4 border-green-400 ml-8';
                    icon = '➤';
                    break;
                case 'success':
                    className = 'bg-green-50 text-green-800 border-l-4 border-green-400';
                    icon = '✅';
                    break;
                case 'error':
                    className = 'bg-red-50 text-red-800 border-l-4 border-red-400';
                    icon = '❌';
                    break;
                case 'notification':
                    className = 'bg-yellow-50 text-yellow-800 border-l-4 border-yellow-400';
                    icon = '📢';
                    break;
                case 'heartbeat':
                    className = 'bg-pink-50 text-pink-800 border-l-4 border-pink-400';
                    icon = '💓';
                    break;
                default:
                    className = 'bg-gray-50 text-gray-800 border-l-4 border-gray-300';
                    icon = '📝';
            }
            
            messageDiv.className += ' ' + className;
            messageDiv.innerHTML = '<div class="flex items-start"><span class="mr-2 text-base">' + icon + '</span><div class="flex-1"><div class="font-medium">' + message + '</div><div class="text-xs opacity-75 mt-1">' + timestamp + '</div></div></div>';
            
            container.appendChild(messageDiv);
            container.scrollTop = container.scrollHeight;
        }

        function clearEmptyState(containerId) {
            const container = document.getElementById(containerId);
            const emptyState = container.querySelector('.text-gray-500');
            if (emptyState) {
                emptyState.remove();
            }
        }

        function showAlert(message, type) {
            const alertDiv = document.createElement('div');
            alertDiv.className = 'fixed top-4 right-4 z-50 p-4 rounded-lg shadow-lg max-w-sm transform transition-transform duration-300 translate-x-full';
            
            let bgColor = '';
            let textColor = '';
            let icon = '';
            
            switch (type) {
                case 'success':
                    bgColor = 'bg-green-500';
                    textColor = 'text-white';
                    icon = '✅';
                    break;
                case 'error':
                    bgColor = 'bg-red-500';
                    textColor = 'text-white';
                    icon = '❌';
                    break;
                case 'warning':
                    bgColor = 'bg-yellow-500';
                    textColor = 'text-white';
                    icon = '⚠️';
                    break;
                default:
                    bgColor = 'bg-blue-500';
                    textColor = 'text-white';
                    icon = 'ℹ️';
            }
            
            alertDiv.className += ' ' + bgColor + ' ' + textColor;
            alertDiv.innerHTML = '<div class="flex items-center"><span class="mr-2">' + icon + '</span><span class="font-medium">' + message + '</span></div>';
            
            document.body.appendChild(alertDiv);
            
            setTimeout(function() {
                alertDiv.classList.remove('translate-x-full');
                alertDiv.classList.add('translate-x-0');
            }, 100);
            
            setTimeout(function() {
                alertDiv.classList.remove('translate-x-0');
                alertDiv.classList.add('translate-x-full');
                setTimeout(function() {
                    document.body.removeChild(alertDiv);
                }, 300);
            }, 3000);
        }

        // Event Listeners
        document.getElementById('wsMessage').addEventListener('keypress', function(e) {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                sendWebSocketMessage();
            }
        });

        document.getElementById('notifContent').addEventListener('keypress', function(e) {
            if (e.key === 'Enter' && e.ctrlKey) {
                e.preventDefault();
                sendNotification();
            }
        });
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	port := 8080

	fmt.Println(fmt.Sprintf("🚀 Server starting on :%v", port))
	fmt.Println(fmt.Sprintf("🌐 Demo: http://localhost:%v", port))
	fmt.Println(fmt.Sprintf("📚 OpenAPI spec: http://localhost:%v/openapi", port))
	fmt.Println(fmt.Sprintf("📖 Swagger UI: http://localhost:%v/openapi/swagger", port))
	fmt.Println(fmt.Sprintf("🔗 AsyncAPI spec: http://localhost:%v/asyncapi", port))
	fmt.Println(fmt.Sprintf("📄 AsyncAPI docs: http://localhost:%v/asyncapi/docs", port))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), router))
}
