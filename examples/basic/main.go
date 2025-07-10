package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/steel"
	"github.com/xraph/steel/middleware"
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
func GetUserHandler(ctx *steel.Context, input GetUserInput) (*GetUserOutput, error) {
	user, exists := users[input.ID]
	if !exists {
		return nil, errors.New("user not found")
	}

	return &GetUserOutput{User: user}, nil
}

func CreateUserHandler(ctx *steel.Context, input CreateUserInput) (*CreateUserOutput, error) {
	// Check authorization
	if input.AuthToken != "Bearer valid-token" {
		return nil, steel.Unauthorized("Authorization required for user deletion")
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

func UpdateUserHandler(ctx *steel.Context, input UpdateUserInput) (*UpdateUserOutput, error) {
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

func ListUsersHandler(ctx *steel.Context, input ListUsersInput) (*ListUsersOutput, error) {
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

func DeleteUserHandler(ctx *steel.Context, input DeleteUserInput) (*DeleteUserOutput, error) {
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

func GetUserPostsHandler(ctx *steel.Context, input GetUserPostsInput) (*GetUserPostsOutput, error) {
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

func CreatePostHandler(ctx *steel.Context, input CreatePostInput) (*CreatePostOutput, error) {
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

func SearchHandler(ctx *steel.Context, input SearchInput) (*SearchOutput, error) {
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

// HealthCheckHandler Health check with minimal input/output
func HealthCheckHandler(ctx *steel.Context, input struct{}) (*struct {
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
	router := steel.NewRouter()

	// Enable OpenAPI documentation
	router.EnableOpenAPI()

	// Global middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

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
		steel.WithSummary("Health Check"),
		steel.WithDescription("Returns the health status of the API"),
		steel.WithTags("system"))

	// Search endpoint
	router.OpinionatedGET("/search", SearchHandler,
		steel.WithSummary("Search"),
		steel.WithDescription("Search across users and posts"),
		steel.WithTags("search"))

	// User management API
	router.Route("/api/v1", func(r steel.Router) {
		r.Route("/users", func(r steel.Router) {
			// List users
			r.OpinionatedGET("/", ListUsersHandler,
				steel.WithSummary("List Users"),
				steel.WithDescription("Get paginated list of users with optional search"),
				steel.WithTags("users"))

			// Create user
			r.OpinionatedPOST("/", CreateUserHandler,
				steel.WithSummary("Create User"),
				steel.WithDescription("Create a new user account"),
				steel.WithTags("users"))

			// User-specific routes
			r.Route("/{id}", func(r steel.Router) {
				// Get user
				r.OpinionatedGET("/", GetUserHandler,
					steel.WithSummary("Get User"),
					steel.WithDescription("Get user details by ID"),
					steel.WithTags("users"))

				// Update user
				r.OpinionatedPUT("/", UpdateUserHandler,
					steel.WithSummary("Update User"),
					steel.WithDescription("Update user information"),
					steel.WithTags("users"))

				// Delete user
				r.OpinionatedDELETE("/", DeleteUserHandler,
					steel.WithSummary("Delete User"),
					steel.WithDescription("Delete user account"),
					steel.WithTags("users"))

				// User posts
				r.Route("/posts", func(r steel.Router) {
					// Get user's posts
					r.OpinionatedGET("/", GetUserPostsHandler,
						steel.WithSummary("Get User Posts"),
						steel.WithDescription("Get all posts by a specific user"),
						steel.WithTags("users", "posts"))

					// Create post for user
					r.OpinionatedPOST("/", CreatePostHandler,
						steel.WithSummary("Create Post"),
						steel.WithDescription("Create a new post for the user"),
						steel.WithTags("users", "posts"))
				})
			})
		})
	})

	// Mixed traditional and opinionated handlers
	router.GET("/traditional", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is a traditional handler"))
	})

	fmt.Println("🚀 SteelRouter with Opinionated Handlers")
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
