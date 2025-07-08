package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	// FastRouter (our implementation)
	forge_router "github.com/xraph/forgerouter"

	// Popular Go routers for comparison
	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/gorilla/mux"
	"github.com/julienschmidt/httprouter"
	"github.com/labstack/echo/v4"
)

// Common handler for all routers
func simpleHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Setup functions for each router
func setupFastRouter() http.Handler {
	router := forge_router.NewRouter()

	// Static routes
	router.GET("/", simpleHandler)
	router.GET("/users", simpleHandler)
	router.GET("/users/profile", simpleHandler)
	router.GET("/api/v1/status", simpleHandler)
	router.GET("/api/v1/health", simpleHandler)

	// Parameter routes
	router.GET("/users/:id", simpleHandler)
	router.GET("/users/:id/posts", simpleHandler)
	router.GET("/users/:id/posts/:postId", simpleHandler)
	router.GET("/api/v1/users/:userId", simpleHandler)
	router.GET("/api/v1/users/:userId/orders/:orderId", simpleHandler)

	// Wildcard routes
	router.GET("/static/*", simpleHandler)
	router.GET("/files/*", simpleHandler)

	// Method variations
	router.POST("/users", simpleHandler)
	router.PUT("/users/:id", simpleHandler)
	router.DELETE("/users/:id", simpleHandler)
	router.PATCH("/users/:id", simpleHandler)

	return router
}

func setupChi() http.Handler {
	router := chi.NewRouter()

	// Static routes
	router.Get("/", simpleHandler)
	router.Get("/users", simpleHandler)
	router.Get("/users/profile", simpleHandler)
	router.Get("/api/v1/status", simpleHandler)
	router.Get("/api/v1/health", simpleHandler)

	// Parameter routes
	router.Get("/users/{id}", simpleHandler)
	router.Get("/users/{id}/posts", simpleHandler)
	router.Get("/users/{id}/posts/{postId}", simpleHandler)
	router.Get("/api/v1/users/{userId}", simpleHandler)
	router.Get("/api/v1/users/{userId}/orders/{orderId}", simpleHandler)

	// Wildcard routes
	router.Get("/static/*", simpleHandler)
	router.Get("/files/*", simpleHandler)

	// Method variations
	router.Post("/users", simpleHandler)
	router.Put("/users/{id}", simpleHandler)
	router.Delete("/users/{id}", simpleHandler)
	router.Patch("/users/{id}", simpleHandler)

	return router
}

func setupGin() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	ginHandler := func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	}

	// Static routes
	router.GET("/", ginHandler)
	router.GET("/users", ginHandler)
	router.GET("/users/profile", ginHandler)
	router.GET("/api/v1/status", ginHandler)
	router.GET("/api/v1/health", ginHandler)

	// Parameter routes
	router.GET("/users/:id", ginHandler)
	router.GET("/users/:id/posts", ginHandler)
	router.GET("/users/:id/posts/:postId", ginHandler)
	router.GET("/api/v1/users/:userId", ginHandler)
	router.GET("/api/v1/users/:userId/orders/:orderId", ginHandler)

	// Wildcard routes
	router.GET("/static/*filepath", ginHandler)
	router.GET("/files/*filepath", ginHandler)

	// Method variations
	router.POST("/users", ginHandler)
	router.PUT("/users/:id", ginHandler)
	router.DELETE("/users/:id", ginHandler)
	router.PATCH("/users/:id", ginHandler)

	return router
}

func setupFiber() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	fiberHandler := func(c *fiber.Ctx) error {
		return c.SendString("OK")
	}

	// Static routes
	app.Get("/", fiberHandler)
	app.Get("/users", fiberHandler)
	app.Get("/users/profile", fiberHandler)
	app.Get("/api/v1/status", fiberHandler)
	app.Get("/api/v1/health", fiberHandler)

	// Parameter routes
	app.Get("/users/:id", fiberHandler)
	app.Get("/users/:id/posts", fiberHandler)
	app.Get("/users/:id/posts/:postId", fiberHandler)
	app.Get("/api/v1/users/:userId", fiberHandler)
	app.Get("/api/v1/users/:userId/orders/:orderId", fiberHandler)

	// Wildcard routes
	app.Get("/static/*", fiberHandler)
	app.Get("/files/*", fiberHandler)

	// Method variations
	app.Post("/users", fiberHandler)
	app.Put("/users/:id", fiberHandler)
	app.Delete("/users/:id", fiberHandler)
	app.Patch("/users/:id", fiberHandler)

	return app
}

func setupEcho() http.Handler {
	router := echo.New()
	router.HideBanner = true

	echoHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	}

	// Static routes
	router.GET("/", echoHandler)
	router.GET("/users", echoHandler)
	router.GET("/users/profile", echoHandler)
	router.GET("/api/v1/status", echoHandler)
	router.GET("/api/v1/health", echoHandler)

	// Parameter routes
	router.GET("/users/:id", echoHandler)
	router.GET("/users/:id/posts", echoHandler)
	router.GET("/users/:id/posts/:postId", echoHandler)
	router.GET("/api/v1/users/:userId", echoHandler)
	router.GET("/api/v1/users/:userId/orders/:orderId", echoHandler)

	// Wildcard routes
	router.GET("/static/*", echoHandler)
	router.GET("/files/*", echoHandler)

	// Method variations
	router.POST("/users", echoHandler)
	router.PUT("/users/:id", echoHandler)
	router.DELETE("/users/:id", echoHandler)
	router.PATCH("/users/:id", echoHandler)

	return router
}

func setupHttpRouter() http.Handler {
	router := httprouter.New()

	httpRouterHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}

	// Static routes
	router.GET("/", httpRouterHandler)
	router.GET("/users", httpRouterHandler)
	router.GET("/users/profile", httpRouterHandler)
	router.GET("/api/v1/status", httpRouterHandler)
	router.GET("/api/v1/health", httpRouterHandler)

	// // Parameter routes
	// router.GET("/users/:id", httpRouterHandler)
	// router.GET("/users/:id/posts", httpRouterHandler)
	// router.GET("/users/:id/posts/:postId", httpRouterHandler)
	// router.GET("/api/v1/users/:userId", httpRouterHandler)
	// router.GET("/api/v1/users/:userId/orders/:orderId", httpRouterHandler)

	// Wildcard routes
	router.GET("/static/*filepath", httpRouterHandler)
	router.GET("/files/*filepath", httpRouterHandler)

	// Method variations
	router.POST("/users", httpRouterHandler)
	// router.PUT("/users/:id", httpRouterHandler)
	// router.DELETE("/users/:id", httpRouterHandler)
	// router.PATCH("/users/:id", httpRouterHandler)

	return router
}

func setupGorillaMux() http.Handler {
	router := mux.NewRouter()

	// Static routes
	router.HandleFunc("/", simpleHandler).Methods("GET")
	router.HandleFunc("/users", simpleHandler).Methods("GET")
	router.HandleFunc("/users/profile", simpleHandler).Methods("GET")
	router.HandleFunc("/api/v1/status", simpleHandler).Methods("GET")
	router.HandleFunc("/api/v1/health", simpleHandler).Methods("GET")

	// Parameter routes
	router.HandleFunc("/users/{id}", simpleHandler).Methods("GET")
	router.HandleFunc("/users/{id}/posts", simpleHandler).Methods("GET")
	router.HandleFunc("/users/{id}/posts/{postId}", simpleHandler).Methods("GET")
	router.HandleFunc("/api/v1/users/{userId}", simpleHandler).Methods("GET")
	router.HandleFunc("/api/v1/users/{userId}/orders/{orderId}", simpleHandler).Methods("GET")

	// Wildcard routes
	router.PathPrefix("/static/").HandlerFunc(simpleHandler).Methods("GET")
	router.PathPrefix("/files/").HandlerFunc(simpleHandler).Methods("GET")

	// Method variations
	router.HandleFunc("/users", simpleHandler).Methods("POST")
	router.HandleFunc("/users/{id}", simpleHandler).Methods("PUT")
	router.HandleFunc("/users/{id}", simpleHandler).Methods("DELETE")
	router.HandleFunc("/users/{id}", simpleHandler).Methods("PATCH")

	return router
}

// Benchmark static routes
func BenchmarkStaticRoutes(b *testing.B) {
	routers := map[string]http.Handler{
		"FastRouter": setupFastRouter(),
		"Chi":        setupChi(),
		"Gin":        setupGin(),
		"Echo":       setupEcho(),
		"HttpRouter": setupHttpRouter(),
		"GorillaMux": setupGorillaMux(),
	}

	// Special handling for Fiber (doesn't implement http.Handler)
	fiberApp := setupFiber()

	testPaths := []string{
		"/",
		"/users",
		"/users/profile",
		"/api/v1/status",
		"/api/v1/health",
	}

	for name, router := range routers {
		for _, path := range testPaths {
			b.Run(fmt.Sprintf("%s_%s", name, path), func(b *testing.B) {
				req := httptest.NewRequest("GET", path, nil)
				w := httptest.NewRecorder()

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					w.Body.Reset()
					router.ServeHTTP(w, req)
				}
			})
		}
	}

	// Benchmark Fiber separately
	for _, path := range testPaths {
		b.Run(fmt.Sprintf("Fiber_%s", path), func(b *testing.B) {
			req := httptest.NewRequest("GET", path, nil)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := fiberApp.Test(req)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark parameter routes
func BenchmarkParameterRoutes(b *testing.B) {
	routers := map[string]http.Handler{
		"FastRouter": setupFastRouter(),
		"Chi":        setupChi(),
		"Gin":        setupGin(),
		"Echo":       setupEcho(),
		"HttpRouter": setupHttpRouter(),
		"GorillaMux": setupGorillaMux(),
	}

	fiberApp := setupFiber()

	testPaths := []string{
		"/users/123",
		"/users/456/posts",
		"/users/789/posts/abc",
		"/api/v1/users/123",
		"/api/v1/users/456/orders/789",
	}

	for name, router := range routers {
		for _, path := range testPaths {
			b.Run(fmt.Sprintf("%s_%s", name, path), func(b *testing.B) {
				req := httptest.NewRequest("GET", path, nil)
				w := httptest.NewRecorder()

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					w.Body.Reset()
					router.ServeHTTP(w, req)
				}
			})
		}
	}

	// Benchmark Fiber separately
	for _, path := range testPaths {
		b.Run(fmt.Sprintf("Fiber_%s", path), func(b *testing.B) {
			req := httptest.NewRequest("GET", path, nil)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := fiberApp.Test(req)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark wildcard routes
func BenchmarkWildcardRoutes(b *testing.B) {
	routers := map[string]http.Handler{
		"FastRouter": setupFastRouter(),
		"Chi":        setupChi(),
		"Gin":        setupGin(),
		"Echo":       setupEcho(),
		"HttpRouter": setupHttpRouter(),
		"GorillaMux": setupGorillaMux(),
	}

	fiberApp := setupFiber()

	testPaths := []string{
		"/static/css/style.css",
		"/static/js/app.js",
		"/files/documents/report.pdf",
		"/static/images/logo.png",
	}

	for name, router := range routers {
		for _, path := range testPaths {
			b.Run(fmt.Sprintf("%s_%s", name, path), func(b *testing.B) {
				req := httptest.NewRequest("GET", path, nil)
				w := httptest.NewRecorder()

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					w.Body.Reset()
					router.ServeHTTP(w, req)
				}
			})
		}
	}

	// Benchmark Fiber separately
	for _, path := range testPaths {
		b.Run(fmt.Sprintf("Fiber_%s", path), func(b *testing.B) {
			req := httptest.NewRequest("GET", path, nil)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := fiberApp.Test(req)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark with middleware
func BenchmarkWithMiddleware(b *testing.B) {
	// Setup routers with middleware
	middlewareFunc := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simple middleware that adds a header
			w.Header().Set("X-Test", "benchmark")
			next.ServeHTTP(w, r)
		})
	}

	// FastRouter with middleware
	fastRouter := forge_router.NewRouter()
	fastRouter.Use(middlewareFunc)
	fastRouter.GET("/test", simpleHandler)

	// Chi with middleware
	chiRouter := chi.NewRouter()
	chiRouter.Use(middlewareFunc)
	chiRouter.Get("/test", simpleHandler)

	// Gin with middleware
	gin.SetMode(gin.ReleaseMode)
	ginRouter := gin.New()
	ginRouter.Use(func(c *gin.Context) {
		c.Header("X-Test", "benchmark")
		c.Next()
	})
	ginRouter.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	routers := map[string]http.Handler{
		"FastRouter": fastRouter,
		"Chi":        chiRouter,
		"Gin":        ginRouter,
	}

	for name, router := range routers {
		b.Run(name, func(b *testing.B) {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				w.Body.Reset()
				router.ServeHTTP(w, req)
			}
		})
	}
}

// Benchmark route lookup performance with many routes
func BenchmarkManyRoutes(b *testing.B) {
	// Create routers with many routes
	fastRouter := forge_router.NewRouter()
	chiRouter := chi.NewRouter()
	gin.SetMode(gin.ReleaseMode)
	ginRouter := gin.New()

	// Add 1000 routes to each router
	for i := 0; i < 1000; i++ {
		path := fmt.Sprintf("/route%d", i)
		paramPath := fmt.Sprintf("/route%d/:id", i)

		// FastRouter
		fastRouter.GET(path, simpleHandler)
		fastRouter.GET(paramPath, simpleHandler)

		// Chi
		chiRouter.Get(path, simpleHandler)
		chiRouter.Get(strings.Replace(paramPath, ":", "", 1), simpleHandler)

		// Gin
		ginRouter.GET(path, func(c *gin.Context) { c.String(http.StatusOK, "OK") })
		ginRouter.GET(paramPath, func(c *gin.Context) { c.String(http.StatusOK, "OK") })
	}

	routers := map[string]http.Handler{
		"FastRouter": fastRouter,
		"Chi":        chiRouter,
		"Gin":        ginRouter,
	}

	// Test lookup performance for routes at different positions
	testRoutes := []string{
		"/route0",       // First route
		"/route500",     // Middle route
		"/route999",     // Last route
		"/route500/123", // Parameter route
	}

	for name, router := range routers {
		for _, route := range testRoutes {
			b.Run(fmt.Sprintf("%s_%s", name, route), func(b *testing.B) {
				req := httptest.NewRequest("GET", route, nil)
				w := httptest.NewRecorder()

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					w.Body.Reset()
					router.ServeHTTP(w, req)
				}
			})
		}
	}
}

// Memory allocation benchmark
func BenchmarkMemoryAllocations(b *testing.B) {
	routers := map[string]http.Handler{
		"FastRouter": setupFastRouter(),
		"Chi":        setupChi(),
		"Gin":        setupGin(),
		"Echo":       setupEcho(),
		"HttpRouter": setupHttpRouter(),
	}

	for name, router := range routers {
		b.Run(name, func(b *testing.B) {
			req := httptest.NewRequest("GET", "/users/123/posts/456", nil)
			w := httptest.NewRecorder()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				w.Body.Reset()
				router.ServeHTTP(w, req)
			}
		})
	}
}

// Concurrent request benchmark
func BenchmarkConcurrentRequests(b *testing.B) {
	routers := map[string]http.Handler{
		"FastRouter": setupFastRouter(),
		"Chi":        setupChi(),
		"Gin":        setupGin(),
		"HttpRouter": setupHttpRouter(),
	}

	for name, router := range routers {
		b.Run(name, func(b *testing.B) {
			req := httptest.NewRequest("GET", "/users/123", nil)

			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				w := httptest.NewRecorder()
				for pb.Next() {
					w.Body.Reset()
					router.ServeHTTP(w, req)
				}
			})
		})
	}
}

// Mixed workload benchmark (realistic scenario)
func BenchmarkMixedWorkload(b *testing.B) {
	routers := map[string]http.Handler{
		"FastRouter": setupFastRouter(),
		"Chi":        setupChi(),
		"Gin":        setupGin(),
		"HttpRouter": setupHttpRouter(),
	}

	requests := []*http.Request{
		httptest.NewRequest("GET", "/", nil),
		httptest.NewRequest("GET", "/users", nil),
		httptest.NewRequest("GET", "/users/123", nil),
		httptest.NewRequest("POST", "/users", nil),
		httptest.NewRequest("GET", "/users/456/posts", nil),
		httptest.NewRequest("GET", "/api/v1/status", nil),
		httptest.NewRequest("GET", "/static/css/style.css", nil),
		httptest.NewRequest("PUT", "/users/789", nil),
	}

	for name, router := range routers {
		b.Run(name, func(b *testing.B) {
			w := httptest.NewRecorder()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req := requests[i%len(requests)]
				w.Body.Reset()
				router.ServeHTTP(w, req)
			}
		})
	}
}
