package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	forgerouter2 "github.com/xraph/forgerouter"
)

// User domain models
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Post struct {
	ID      string   `json:"id"`
	UserID  string   `json:"user_id"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

// Input/Output structs with comprehensive tagging

// GET /users/{id}
type GetUserInput struct {
	ID     string `path:"id" description:"User ID" required:"true"`
	Format string `query:"format" description:"Response format (json|xml)" default:"json"`
}

type GetUserOutput struct {
	User *User `json:"user"`
}

// POST /users
type CreateUserInput struct {
	Name      string `json:"name" required:"true" description:"User's full name"`
	Email     string `json:"email" required:"true" description:"User's email address"`
	Password  string `json:"password" required:"true" description:"User's password"`
	AuthToken string `header:"Authorization" required:"true" description:"Bearer token"`
	ClientID  string `header:"X-Client-ID" description:"Client identifier"`
}

type CreateUserOutput struct {
	User    *User  `json:"user"`
	Message string `json:"message"`
}

// PUT /users/{id}
type UpdateUserInput struct {
	ID        string `path:"id" required:"true"`
	Name      string `json:"name" description:"Updated name"`
	Email     string `json:"email" description:"Updated email"`
	AuthToken string `header:"Authorization" required:"true"`
}

type UpdateUserOutput struct {
	User    *User  `json:"user"`
	Message string `json:"message"`
}

// GET /users
type ListUsersInput struct {
	Page     int    `query:"page" description:"Page number" default:"1"`
	PageSize int    `query:"page_size" description:"Items per page" default:"10"`
	Search   string `query:"search" description:"Search term"`
	SortBy   string `query:"sort_by" description:"Sort field" default:"created_at"`
	Order    string `query:"order" description:"Sort order (asc|desc)" default:"desc"`
}

type ListUsersOutput struct {
	Users      []User `json:"users"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
}

// DELETE /users/{id}
type DeleteUserInput struct {
	ID        string `path:"id" required:"true"`
	AuthToken string `header:"Authorization" required:"true"`
	Force     bool   `query:"force" description:"Force delete even with dependencies"`
}

type DeleteUserOutput struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// GET /users/{id}/posts
type GetUserPostsInput struct {
	UserID   string   `path:"id" required:"true"`
	Tags     []string `query:"tags" description:"Filter by tags"`
	Status   string   `query:"status" description:"Filter by status"`
	PageSize int      `query:"page_size" default:"20"`
}

type GetUserPostsOutput struct {
	Posts []Post `json:"posts"`
	Count int    `json:"count"`
}

// POST /users/{id}/posts
type CreatePostInput struct {
	UserID  string   `path:"id" required:"true"`
	Title   string   `json:"title" required:"true"`
	Content string   `json:"content" required:"true"`
	Tags    []string `json:"tags"`
	Draft   bool     `json:"draft" default:"false"`
}

type CreatePostOutput struct {
	Post    *Post  `json:"post"`
	Message string `json:"message"`
}

// Complex nested structure example
type SearchInput struct {
	Query      string            `query:"q" required:"true" description:"Search query"`
	Type       string            `query:"type" description:"Search type (users|posts|all)" default:"all"`
	Filters    map[string]string `query:"filters" description:"Additional filters"`
	UserAgent  string            `header:"User-Agent" description:"Client user agent"`
	APIVersion string            `header:"X-API-Version" default:"v1"`
}

type SearchOutput struct {
	Results struct {
		Users []User `json:"users"`
		Posts []Post `json:"posts"`
	} `json:"results"`
	Meta struct {
		Query     string `json:"query"`
		Total     int    `json:"total"`
		Duration  string `json:"duration"`
		Timestamp string `json:"timestamp"`
	} `json:"meta"`
}

// Mock data store
var users = make(map[string]*User)
var posts = make(map[string]*Post)

// Handler implementations
func GetUserHandler(ctx *forgerouter2.FastContext, input GetUserInput) (*GetUserOutput, error) {
	user, exists := users[input.ID]
	if !exists {
		return nil, errors.New("user not found")
	}

	return &GetUserOutput{User: user}, nil
}

func CreateUserHandler(ctx *forgerouter2.FastContext, input CreateUserInput) (*CreateUserOutput, error) {
	// Check authorization
	if input.AuthToken != "Bearer valid-token" {
		return nil, forgerouter2.Unauthorized("Authorization required for user deletion")
	}

	// Create new user
	user := &User{
		ID:        fmt.Sprintf("user_%d", len(users)+1),
		Name:      input.Name,
		Email:     input.Email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	users[user.ID] = user

	return &CreateUserOutput{
		User:    user,
		Message: "User created successfully",
	}, nil
}

func UpdateUserHandler(ctx *forgerouter2.FastContext, input UpdateUserInput) (*UpdateUserOutput, error) {
	user, exists := users[input.ID]
	if !exists {
		return nil, errors.New("user not found")
	}

	// Update fields if provided
	if input.Name != "" {
		user.Name = input.Name
	}
	if input.Email != "" {
		user.Email = input.Email
	}
	user.UpdatedAt = time.Now()

	return &UpdateUserOutput{
		User:    user,
		Message: "User updated successfully",
	}, nil
}

func ListUsersHandler(ctx *forgerouter2.FastContext, input ListUsersInput) (*ListUsersOutput, error) {
	// Convert map to slice
	userList := make([]User, 0, len(users))
	for _, user := range users {
		userList = append(userList, *user)
	}

	// Apply pagination (simplified)
	start := (input.Page - 1) * input.PageSize
	end := start + input.PageSize
	if end > len(userList) {
		end = len(userList)
	}
	if start > len(userList) {
		start = len(userList)
	}

	pagedUsers := userList[start:end]
	totalPages := (len(userList) + input.PageSize - 1) / input.PageSize

	return &ListUsersOutput{
		Users:      pagedUsers,
		Total:      len(userList),
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalPages: totalPages,
	}, nil
}

func DeleteUserHandler(ctx *forgerouter2.FastContext, input DeleteUserInput) (*DeleteUserOutput, error) {
	_, exists := users[input.ID]
	if !exists {
		return nil, errors.New("user not found")
	}

	delete(users, input.ID)

	return &DeleteUserOutput{
		Message: "User deleted successfully",
		Success: true,
	}, nil
}

func GetUserPostsHandler(ctx *forgerouter2.FastContext, input GetUserPostsInput) (*GetUserPostsOutput, error) {
	// Filter posts by user ID
	userPosts := make([]Post, 0)
	for _, post := range posts {
		if post.UserID == input.UserID {
			userPosts = append(userPosts, *post)
		}
	}

	return &GetUserPostsOutput{
		Posts: userPosts,
		Count: len(userPosts),
	}, nil
}

func CreatePostHandler(ctx *forgerouter2.FastContext, input CreatePostInput) (*CreatePostOutput, error) {
	// Verify user exists
	_, exists := users[input.UserID]
	if !exists {
		return nil, errors.New("user not found")
	}

	post := &Post{
		ID:      fmt.Sprintf("post_%d", len(posts)+1),
		UserID:  input.UserID,
		Title:   input.Title,
		Content: input.Content,
		Tags:    input.Tags,
	}

	posts[post.ID] = post

	return &CreatePostOutput{
		Post:    post,
		Message: "Post created successfully",
	}, nil
}

func SearchHandler(ctx *forgerouter2.FastContext, input SearchInput) (*SearchOutput, error) {
	start := time.Now()

	result := &SearchOutput{}

	// Simple search implementation
	for _, user := range users {
		if strings.Contains(strings.ToLower(user.Name), strings.ToLower(input.Query)) ||
			strings.Contains(strings.ToLower(user.Email), strings.ToLower(input.Query)) {
			result.Results.Users = append(result.Results.Users, *user)
		}
	}

	for _, post := range posts {
		if strings.Contains(strings.ToLower(post.Title), strings.ToLower(input.Query)) ||
			strings.Contains(strings.ToLower(post.Content), strings.ToLower(input.Query)) {
			result.Results.Posts = append(result.Results.Posts, *post)
		}
	}

	// Set metadata
	result.Meta.Query = input.Query
	result.Meta.Total = len(result.Results.Users) + len(result.Results.Posts)
	result.Meta.Duration = time.Since(start).String()
	result.Meta.Timestamp = time.Now().Format(time.RFC3339)

	return result, nil
}

// Health check with minimal input/output
func HealthCheckHandler(ctx *forgerouter2.FastContext, input struct{}) (*struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}, error) {
	return &struct {
		Status    string    `json:"status"`
		Timestamp time.Time `json:"timestamp"`
		Version   string    `json:"version"`
	}{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}, nil
}

func main() {
	router := forgerouter2.NewRouter()

	// Enable OpenAPI documentation
	router.EnableOpenAPI()

	// Global middleware
	router.Use(forgerouter2.Logger)
	router.Use(forgerouter2.Recoverer)

	// Create some sample data
	users["user_1"] = &User{
		ID:        "user_1",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
	}

	posts["post_1"] = &Post{
		ID:      "post_1",
		UserID:  "user_1",
		Title:   "Hello World",
		Content: "This is my first post!",
		Tags:    []string{"introduction", "hello"},
	}

	// Health endpoint
	router.OpinionatedGET("/health", HealthCheckHandler,
		forgerouter2.WithSummary("Health Check"),
		forgerouter2.WithDescription("Returns the health status of the API"),
		forgerouter2.WithTags("system"))

	// Search endpoint
	router.OpinionatedGET("/search", SearchHandler,
		forgerouter2.WithSummary("Search"),
		forgerouter2.WithDescription("Search across users and posts"),
		forgerouter2.WithTags("search"))

	// User management API
	router.Route("/api/v1", func(r forgerouter2.Router) {
		r.Route("/users", func(r forgerouter2.Router) {
			// List users
			r.OpinionatedGET("/", ListUsersHandler,
				forgerouter2.WithSummary("List Users"),
				forgerouter2.WithDescription("Get paginated list of users with optional search"),
				forgerouter2.WithTags("users"))

			// Create user
			r.OpinionatedPOST("/", CreateUserHandler,
				forgerouter2.WithSummary("Create User"),
				forgerouter2.WithDescription("Create a new user account"),
				forgerouter2.WithTags("users"))

			// User-specific routes
			r.Route("/{id}", func(r forgerouter2.Router) {
				// Get user
				r.OpinionatedGET("/", GetUserHandler,
					forgerouter2.WithSummary("Get User"),
					forgerouter2.WithDescription("Get user details by ID"),
					forgerouter2.WithTags("users"))

				// Update user
				r.OpinionatedPUT("/", UpdateUserHandler,
					forgerouter2.WithSummary("Update User"),
					forgerouter2.WithDescription("Update user information"),
					forgerouter2.WithTags("users"))

				// Delete user
				r.OpinionatedDELETE("/", DeleteUserHandler,
					forgerouter2.WithSummary("Delete User"),
					forgerouter2.WithDescription("Delete user account"),
					forgerouter2.WithTags("users"))

				// User posts
				r.Route("/posts", func(r forgerouter2.Router) {
					// Get user's posts
					r.OpinionatedGET("/", GetUserPostsHandler,
						forgerouter2.WithSummary("Get User Posts"),
						forgerouter2.WithDescription("Get all posts by a specific user"),
						forgerouter2.WithTags("users", "posts"))

					// Create post for user
					r.OpinionatedPOST("/", CreatePostHandler,
						forgerouter2.WithSummary("Create Post"),
						forgerouter2.WithDescription("Create a new post for the user"),
						forgerouter2.WithTags("users", "posts"))
				})
			})
		})
	})

	// Mixed traditional and opinionated handlers
	router.GET("/traditional", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is a traditional handler"))
	})

	fmt.Println("🚀 FastRouter with Opinionated Handlers")
	fmt.Println("=======================================")
	fmt.Println("Server starting on :8080")
	fmt.Println("")
	fmt.Println("📚 Documentation:")
	fmt.Println("  OpenAPI spec: http://localhost:8080/openapi")
	fmt.Println("  Swagger UI:   http://localhost:8080/openapi/swagger")
	fmt.Println("")
	fmt.Println("🔧 Example API calls:")
	fmt.Println("  GET  /health")
	fmt.Println("  GET  /api/v1/users")
	fmt.Println("  POST /api/v1/users")
	fmt.Println("  GET  /api/v1/users/user_1")
	fmt.Println("  GET  /api/v1/users/user_1/posts")
	fmt.Println("  GET  /search?q=john")
	fmt.Println("")
	fmt.Println("📋 Test with curl:")
	fmt.Println(`  curl "http://localhost:8080/health"`)
	fmt.Println(`  curl "http://localhost:8080/api/v1/users?page=1&page_size=5"`)
	fmt.Println(`  curl -X POST "http://localhost:8080/api/v1/users" \`)
	fmt.Println(`       -H "Authorization: Bearer valid-token" \`)
	fmt.Println(`       -H "Content-Type: application/json" \`)
	fmt.Println(`       -d '{"name":"Jane Doe","email":"jane@example.com","password":"secret"}'`)

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		return
	}
}
