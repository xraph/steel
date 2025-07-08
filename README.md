# ForgeRouter 🚀

A high-performance HTTP router for Go with automatic OpenAPI documentation, WebSocket/SSE support, and comprehensive testing utilities.

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/xraph/forgerouter)](https://goreportcard.com/report/github.com/xraph/forgerouter)

## ✨ Features

- **🚀 High Performance**: Fast radix tree-based routing with zero allocations for parameter extraction
- **📚 Automatic OpenAPI**: Generate complete OpenAPI 3.0 documentation from your handlers
- **🔌 Real-time**: Built-in WebSocket and Server-Sent Events (SSE) support with AsyncAPI docs
- **🎯 Type-Safe**: Opinionated handlers with automatic parameter binding and validation
- **🧪 Testing**: Comprehensive testing utilities with request builders and assertions
- **⚡ Middleware**: Flexible middleware system with built-in recovery, logging, and timeout
- **🎨 Multiple Docs**: Support for Swagger UI, ReDoc, Scalar, and Stoplight Elements
- **🔧 Developer-Friendly**: Rich debugging tools and development utilities

## 📦 Installation

```bash
go get github.com/xraph/forgerouter
```

## 🚀 Quick Start

```go
package main

import (
    "log"
    "net/http"
    router "github.com/xraph/forgerouter"
)

type User struct {
    ID   int    `json:"id" path:"id" description:"User ID"`
    Name string `json:"name" query:"name" description:"User name"`
}

type UserResponse struct {
    ID      int    `json:"id" description:"User ID"`
    Name    string `json:"name" description:"User name"`
    Created bool   `json:"created" description:"Whether user was created"`
}

func main() {
    r := router.NewFastRouter()
    
    // Add middleware
    r.Use(router.Logger, router.Recoverer)
    
    // Opinionated handler with automatic OpenAPI generation
    r.OpinionatedGET("/users/:id", func(ctx *router.FastContext, req User) (*UserResponse, error) {
        return &UserResponse{
            ID:      req.ID,
            Name:    req.Name,
            Created: false,
        }, nil
    }, router.WithSummary("Get User"), router.WithTags("users"))
    
    // Enable automatic OpenAPI documentation
    r.EnableOpenAPI()
    
    log.Println("Server starting on :8080")
    log.Println("API docs available at http://localhost:8080/openapi/docs")
    log.Fatal(http.ListenAndServe(":8080", r))
}
```

Visit `http://localhost:8080/openapi/docs` to see your automatically generated API documentation!

## 🎯 Key Features

### Opinionated Handlers

ForgeRouter's opinionated handlers provide automatic parameter binding, validation, and OpenAPI generation:

```go
type CreateUserRequest struct {
    Name  string `json:"name" body:"body" description:"User name"`
    Email string `json:"email" body:"body" description:"User email"`
    Age   int    `query:"age" description:"User age"`
}

r.OpinionatedPOST("/users", func(ctx *router.FastContext, req CreateUserRequest) (*UserResponse, error) {
    // Request automatically bound from JSON body and query parameters
    if req.Age < 18 {
        return nil, router.BadRequest("User must be 18 or older")
    }
    
    // Your business logic here
    return &UserResponse{ID: 123, Name: req.Name}, nil
}, router.WithSummary("Create User"))
```

### Real-time Communication

Built-in WebSocket and SSE support with AsyncAPI documentation:

```go
// WebSocket handler
r.WebSocket("/ws/chat", func(conn *router.WSConnection, message ChatMessage) (*ChatResponse, error) {
    return &ChatResponse{Reply: "Echo: " + message.Text}, nil
}, router.WithAsyncSummary("Chat WebSocket"))

// Server-Sent Events
r.SSE("/events/:userId", func(conn *router.SSEConnection, params EventParams) error {
    return conn.SendMessage(router.SSEMessage{
        Event: "notification",
        Data:  map[string]interface{}{"userId": params.UserID},
    })
}, router.WithAsyncSummary("User Events"))

// Enable AsyncAPI documentation
r.EnableAsyncAPI()
```

### Comprehensive Error Handling

Rich error types with automatic OpenAPI documentation:

```go
r.OpinionatedGET("/users/:id", func(ctx *router.FastContext, req GetUserRequest) (*User, error) {
    user, exists := database.GetUser(req.ID)
    if !exists {
        return nil, router.NotFound("User")
    }
    
    if !user.Active {
        return nil, router.Forbidden("User account is deactivated")
    }
    
    return user, nil
})
```

### Testing Made Easy

Built-in testing utilities for comprehensive API testing:

```go
func TestUserAPI(t *testing.T) {
    router := setupTestRouter()
    
    // Test user creation
    response := router.NewRequest("POST", "/users").
        WithJSON(map[string]interface{}{
            "name": "John Doe",
            "email": "john@example.com",
        }).
        Execute(router)
    
    router.AssertResponse(t, response).
        Status(http.StatusCreated).
        IsJSON().
        JSON("name", "John Doe")
}
```

## 📖 Documentation

### Basic Routing

```go
r := router.NewFastRouter()

// HTTP methods
r.GET("/", handler)
r.POST("/users", handler)
r.PUT("/users/:id", handler)
r.DELETE("/users/:id", handler)

// Path parameters
r.GET("/users/:id/posts/:postId", handler)

// Wildcards
r.GET("/static/*", handler)

// Route groups
r.Route("/api/v1", func(r router.Router) {
    r.GET("/users", getUsersHandler)
    r.POST("/users", createUserHandler)
})
```

### Middleware

```go
// Built-in middleware
r.Use(router.Logger)
r.Use(router.Recoverer)
r.Use(router.Timeout(30 * time.Second))

// Custom middleware
r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        // Your middleware logic
        next.ServeHTTP(w, req)
    })
})
```

### Parameter Binding Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `path` | URL path parameter | `ID int \`path:"id"\`` |
| `query` | Query parameter | `Limit int \`query:"limit"\`` |
| `header` | HTTP header | `Auth string \`header:"Authorization"\`` |
| `body` | JSON body field | `Data string \`body:"body"\`` |
| `json` | JSON field name | `Name string \`json:"name"\`` |
| `description` | OpenAPI description | `ID int \`description:"User ID"\`` |

## 🔧 Advanced Features

### Custom Response Types

```go
r.OpinionatedPOST("/users", func(ctx *router.FastContext, req CreateUserRequest) (*router.APIResponse, error) {
    user := createUser(req)
    
    return router.Created(user).
        WithHeader("Location", fmt.Sprintf("/users/%d", user.ID)), nil
})
```

### Validation Errors

```go
func validateUser(req CreateUserRequest) error {
    fields := []router.FieldError{}
    
    if req.Email == "" {
        fields = append(fields, router.NewFieldError("email", "Email is required", req.Email, "REQUIRED"))
    }
    
    if len(fields) > 0 {
        return router.UnprocessableEntity("Validation failed", fields...)
    }
    
    return nil
}
```

### Connection Management

```go
// Get connection manager for WebSocket/SSE connections
cm := router.ConnectionManager()

// Broadcast to all WebSocket connections
cm.BroadcastWS(router.WSMessage{
    Type:    "notification",
    Payload: "Hello everyone!",
})

// Broadcast to all SSE connections
cm.BroadcastSSE(router.SSEMessage{
    Event: "update",
    Data:  "System maintenance in 5 minutes",
})
```

## 📊 Documentation Viewers

ForgeRouter supports multiple documentation viewers out of the box:

- **Swagger UI**: Interactive API documentation
- **ReDoc**: Beautiful, responsive documentation
- **Scalar**: Modern documentation with excellent UX
- **Stoplight Elements**: Comprehensive documentation platform

Access them at:
- `/openapi/docs` - Documentation hub
- `/openapi/swagger` - Swagger UI
- `/openapi/redoc` - ReDoc
- `/openapi/scalar` - Scalar
- `/openapi/spotlight` - Stoplight Elements
- `/asyncapi/docs` - AsyncAPI documentation

## 🧪 Testing

### Load Testing

```go
config := router.LoadTestConfig{
    Concurrency: 10,
    Requests:    1000,
    Timeout:     30 * time.Second,
    Paths:       []string{"/api/users", "/api/posts"},
}

result := router.RunLoadTest(router, config)
fmt.Printf("RPS: %.2f, Success: %d%%\n", 
    result.RequestsPerSecond, 
    (result.SuccessRequests*100)/result.TotalRequests)
```

### Benchmark Testing

```go
func BenchmarkUserAPI(b *testing.B) {
    setup := router.NewBenchmarkSetup().
        AddStaticRoutes(100).
        AddParameterRoutes(50)
    
    router := setup.Setup()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Your benchmark code
    }
}
```

## 🔗 Integration Examples

### With Database (GORM)

```go
type UserService struct {
    db *gorm.DB
}

func (s *UserService) GetUser(ctx *router.FastContext, req GetUserRequest) (*User, error) {
    var user User
    if err := s.db.First(&user, req.ID).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, router.NotFound("User")
        }
        return nil, router.InternalServerError("Database error")
    }
    return &user, nil
}

// Register handler
r.OpinionatedGET("/users/:id", userService.GetUser)
```

### With Validation (go-playground/validator)

```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2" description:"User name"`
    Email string `json:"email" validate:"required,email" description:"User email"`
    Age   int    `json:"age" validate:"min=18" description:"User age"`
}

func createUserWithValidation(ctx *router.FastContext, req CreateUserRequest) (*User, error) {
    if err := validator.New().Struct(req); err != nil {
        var fields []router.FieldError
        for _, err := range err.(validator.ValidationErrors) {
            fields = append(fields, router.NewFieldError(
                err.Field(), 
                err.Error(), 
                err.Value(), 
                err.Tag(),
            ))
        }
        return nil, router.UnprocessableEntity("Validation failed", fields...)
    }
    
    // Create user...
    return &User{}, nil
}
```

## 🏗️ Architecture

ForgeRouter is built with performance and developer experience in mind:

- **Radix Tree Routing**: Efficient O(log n) route matching
- **Zero Allocations**: Parameter extraction without memory allocations
- **Reflection-Based Binding**: Automatic request binding using Go reflection
- **OpenAPI Generation**: Real-time schema generation from Go types
- **Connection Pooling**: Efficient parameter object pooling
- **Middleware Chain**: Flexible middleware composition
- **Error Handling**: Structured error responses with automatic documentation

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [Chi](https://github.com/go-chi/chi) and [Gin](https://github.com/gin-gonic/gin)
- OpenAPI 3.0 specification
- AsyncAPI 2.6 specification
- Go community for amazing ecosystem

## 📞 Support

- 📖 [Documentation](https://forgerouter.dev)
- 🐛 [Report Bug](https://github.com/xraph/forgerouter/issues)
- 💡 [Request Feature](https://github.com/xraph/forgerouter/issues)
- 💬 [Discussions](https://github.com/xraph/forgerouter/discussions)

---

**Made with ❤️ by the ForgeRouter team**