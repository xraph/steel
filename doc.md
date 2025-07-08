# FastRouter Opinionated Handlers - Complete Feature Summary

## 🚀 New Opinionated Handler System

FastRouter now includes a powerful opinionated handler system that automatically generates OpenAPI schemas and provides type-safe parameter binding!

### ✨ Key Features

#### 1. **Type-Safe Handler Signature**
```go
func MyHandler(ctx *FastContext, input MyInputStruct) (*MyOutputStruct, error)
```

#### 2. **Automatic Parameter Binding**
- **Path parameters**: `path:"id"`
- **Query parameters**: `query:"page"`
- **Headers**: `header:"Authorization"`
- **Request body**: `body:"json"` or JSON struct tags
- **Type conversion**: Automatic string → int/bool/float conversion

#### 3. **Comprehensive Struct Tags**
```go
type GetUserInput struct {
    ID     string `path:"id" required:"true" description:"User ID"`
    Format string `query:"format" default:"json" description:"Response format"`
    Token  string `header:"Authorization" required:"true"`
}
```

#### 4. **Automatic OpenAPI 3.0 Generation**
- **Schema introspection** from Go structs
- **Parameter documentation** from struct tags
- **Request/response schemas** auto-generated
- **Validation rules** from tags
- **Interactive Swagger UI** at `/openapi/swagger`

#### 5. **Rich FastContext API**
```go
ctx.Param("id")           // Path parameters
ctx.Query("page")         // Query parameters  
ctx.Header("Auth")        // Headers
ctx.JSON(200, response)   // JSON responses
ctx.BindJSON(&input)      // JSON body binding
```

## 🏷️ Supported Struct Tags

| Tag | Purpose | Example | Description |
|-----|---------|---------|-------------|
| `path:"name"` | URL path parameter | `path:"id"` | Binds to `{id}` in route |
| `query:"name"` | Query parameter | `query:"page"` | Binds to `?page=1` |
| `header:"name"` | HTTP header | `header:"Authorization"` | Binds to request header |
| `body:"json"` | Request body | `body:"json"` | Binds entire JSON body |
| `json:"name"` | JSON field name | `json:"user_id"` | JSON serialization |
| `required:"true"` | Required field | `required:"true"` | OpenAPI validation |
| `description:"text"` | Field documentation | `description:"User ID"` | OpenAPI docs |
| `default:"value"` | Default value | `default:"10"` | OpenAPI default |

## 🔥 Performance Benefits

### **Zero Overhead When Not Used**
- Standard handlers work exactly as before
- No performance impact on existing code
- Opt-in per route

### **Optimized Parameter Binding**
- **Object pooling** for parameter structs
- **Reflection caching** for type introspection
- **Direct field assignment** (no map lookups)
- **Type conversion caching**

## 📊 API Comparison

### **Before: Standard Handlers**
```go
func getUser(w http.ResponseWriter, r *http.Request) {
    // Manual parameter extraction
    userID := chi.URLParam(r, "id")
    format := r.URL.Query().Get("format")
    
    // Manual validation
    if userID == "" {
        http.Error(w, "Missing user ID", 400)
        return
    }
    
    // Manual response
    user := getUserByID(userID)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

// Manual OpenAPI documentation in separate files
```

### **After: Opinionated Handlers**
```go
type GetUserInput struct {
    ID     string `path:"id" required:"true" description:"User ID"`
    Format string `query:"format" default:"json" description:"Response format"`
}

type GetUserOutput struct {
    User *User `json:"user"`
}

func GetUserHandler(ctx *FastContext, input GetUserInput) (*GetUserOutput, error) {
    // Parameters automatically bound and validated
    user := getUserByID(input.ID) // Type-safe access
    
    return &GetUserOutput{User: user}, nil
    // JSON response and status codes handled automatically
}

// OpenAPI schema automatically generated!
```

## 🛠️ Usage Examples

### **Simple GET with path/query params**
```go
type ListUsersInput struct {
    Page     int    `query:"page" default:"1"`
    PageSize int    `query:"page_size" default:"10"`
    Search   string `query:"search" description:"Search term"`
}

router.OpinionatedGET("/users", ListUsersHandler,
    WithSummary("List Users"),
    WithTags("users"))
```

### **POST with JSON body and headers**
```go
type CreateUserInput struct {
    Name      string `json:"name" required:"true"`
    Email     string `json:"email" required:"true"`
    AuthToken string `header:"Authorization" required:"true"`
}

router.OpinionatedPOST("/users", CreateUserHandler,
    WithSummary("Create User"),
    WithDescription("Creates a new user account"))
```

### **Complex nested structures**
```go
type SearchInput struct {
    Query     string            `query:"q" required:"true"`
    Filters   map[string]string `query:"filters"`
    UserAgent string            `header:"User-Agent"`
}

type SearchOutput struct {
    Results struct {
        Users []User `json:"users"`
        Posts []Post `json:"posts"`
    } `json:"results"`
    Meta struct {
        Total    int    `json:"total"`
        Duration string `json:"duration"`
    } `json:"meta"`
}
```

## 📖 OpenAPI Documentation Features

### **Automatic Schema Generation**
- Request/response schemas from Go types
- Parameter documentation from struct tags
- Validation rules and constraints
- Default values and examples

### **Interactive Documentation**
- **Swagger UI** at `/openapi/swagger`
- **Try it out** functionality
- **Schema explorer**
- **Example requests/responses**

### **Standards Compliant**
- **OpenAPI 3.0** specification
- **JSON Schema** for validation
- **HTTP status codes**
- **Content-Type headers**

## 🎯 Migration Path

### **Gradual Adoption**
```go
router := NewFastRouter()

// Existing handlers continue to work
router.GET("/old-endpoint", traditionalHandler)

// New opinionated handlers alongside
router.OpinionatedGET("/new-endpoint", modernHandler)

// Both documented in same OpenAPI spec!
```

### **Route Groups Support**
```go
router.Route("/api/v1", func(r Router) {
    // Both styles work in groups
    r.GET("/traditional", oldHandler)
    r.OpinionatedPOST("/modern", newHandler)
})
```

## 🚀 Getting Started

### **1. Enable OpenAPI**
```go
router := NewFastRouter()
router.EnableOpenAPI() // Adds /openapi and /openapi/swagger
```

### **2. Define Input/Output Structs**
```go
type MyInput struct {
    ID   string `path:"id" required:"true"`
    Name string `query:"name"`
}

type MyOutput struct {
    Result string `json:"result"`
}
```

### **3. Register Opinionated Handler**
```go
router.OpinionatedGET("/items/{id}", MyHandler,
    WithSummary("Get Item"),
    WithTags("items"))
```

### **4. View Documentation**
- Visit `http://localhost:8080/openapi/swagger`
- Interactive API documentation automatically generated!

## 🎉 Benefits Summary

### **For Developers**
- ✅ **Type safety** - No more manual parameter parsing
- ✅ **Less boilerplate** - Automatic binding and validation
- ✅ **Better testing** - Pure functions easier to test
- ✅ **Self-documenting** - Code IS the documentation

### **For APIs**
- ✅ **Automatic documentation** - Always up-to-date
- ✅ **Consistent validation** - Struct tags define rules
- ✅ **Better error handling** - Validation happens before handler
- ✅ **Interactive docs** - Swagger UI out of the box

### **For Performance**
- ✅ **Zero overhead** when not used
- ✅ **Reflection caching** for repeated calls
- ✅ **Object pooling** for allocations
- ✅ **Type-safe access** - No map lookups

FastRouter now provides the best of both worlds: **blazing performance** with **modern developer experience**!