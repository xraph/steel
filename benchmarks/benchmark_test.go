package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestBenchmarkSetup verifies that all routers are correctly configured
func TestBenchmarkSetup(t *testing.T) {
	t.Run("FastRouter Setup", func(t *testing.T) {
		router := setupFastRouter()
		testRouter(t, router, "FastRouter")
	})

	t.Run("Chi Setup", func(t *testing.T) {
		router := setupChi()
		testRouter(t, router, "Chi")
	})

	t.Run("Gin Setup", func(t *testing.T) {
		router := setupGin()
		testRouter(t, router, "Gin")
	})

	t.Run("Echo Setup", func(t *testing.T) {
		router := setupEcho()
		testRouter(t, router, "Echo")
	})

	// t.Run("HttpRouter Setup", func(t *testing.T) {
	// 	router := setupHttpRouter()
	// 	testRouter(t, router, "HttpRouter")
	// })

	t.Run("GorillaMux Setup", func(t *testing.T) {
		router := setupGorillaMux()
		testRouter(t, router, "GorillaMux")
	})

	t.Run("Fiber Setup", func(t *testing.T) {
		app := setupFiber()
		testFiber(t, app)
	})
}

func testRouter(t *testing.T, router http.Handler, name string) {
	testCases := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/", http.StatusOK},
		{"GET", "/users", http.StatusOK},
		{"GET", "/users/123", http.StatusOK},
		{"GET", "/users/123/posts", http.StatusOK},
		{"GET", "/static/test.css", http.StatusOK},
		{"POST", "/users", http.StatusOK},
		{"PUT", "/users/123", http.StatusOK},
		{"DELETE", "/users/123", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.method+"_"+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tc.status {
				t.Errorf("%s: expected status %d for %s %s, got %d",
					name, tc.status, tc.method, tc.path, w.Code)
			}
		})
	}
}

func testFiber(t *testing.T, app *fiber.App) {
	testCases := []string{
		"/",
		"/users",
		"/users/123",
		"/static/test.css",
	}

	for _, path := range testCases {
		t.Run("Fiber_"+path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Fiber test failed: %v", err)
			}

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Fiber: expected status %d for %s, got %d",
					http.StatusOK, path, resp.StatusCode)
			}
		})
	}
}

// BenchmarkQuickComparison - A quick benchmark for development
func BenchmarkQuickComparison(b *testing.B) {
	routers := map[string]http.Handler{
		"FastRouter": setupFastRouter(),
		"Chi":        setupChi(),
		"Gin":        setupGin(),
		// "HttpRouter": setupHttpRouter(),
	}

	path := "/users/123/posts/456"

	for name, router := range routers {
		b.Run(name, func(b *testing.B) {
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

/*
=============================================================================
QUICK REFERENCE GUIDE
=============================================================================

## Running Specific Benchmarks

# Quick comparison (4 main routers)
go test -bench=BenchmarkQuickComparison -benchmem

# Test only static routes
go test -bench=BenchmarkStaticRoutes -benchmem

# Test only parameter routes
go test -bench=BenchmarkParameterRoutes -benchmem

# Test specific router
go test -bench=BenchmarkStaticRoutes.*FastRouter -benchmem

# Test with more iterations for accuracy
go test -bench=BenchmarkQuickComparison -benchmem -count=5

# Test with CPU profiling
go test -bench=BenchmarkQuickComparison -benchmem -cpuprofile=cpu.prof

# Test with memory profiling
go test -bench=BenchmarkMemoryAllocations -benchmem -memprofile=mem.prof

## Analyzing Results

# View CPU profile
go tool pprof cpu.prof

# View memory profile
go tool pprof mem.prof

# Generate visualization
go run visualization.go results/complete_results.txt

## Expected Performance Order (Typical)

### Raw Speed (ns/op - lower is better)
1. HttpRouter      (~50-80 ns/op)
2. FastRouter      (~70-100 ns/op)
3. Chi             (~80-120 ns/op)
4. Gin             (~90-140 ns/op)
5. Echo            (~100-150 ns/op)
6. Fiber           (~80-130 ns/op)*
7. GorillaMux      (~300-800 ns/op)

*Fiber uses fasthttp, results not directly comparable

### Memory Efficiency (B/op - lower is better)
1. HttpRouter      (0 B/op for most routes)
2. FastRouter      (0-32 B/op)
3. Chi             (0-48 B/op)
4. Gin             (32-96 B/op)
5. Echo            (32-128 B/op)
6. GorillaMux      (64-256 B/op)

### Allocations (allocs/op - lower is better)
1. HttpRouter      (0 allocs/op)
2. FastRouter      (0-1 allocs/op)
3. Chi             (0-2 allocs/op)
4. Gin             (1-3 allocs/op)
5. Echo            (1-4 allocs/op)
6. GorillaMux      (2-8 allocs/op)

## Interpreting Relative Performance

- 1.0x = baseline (fastest)
- 1.5x = 50% slower
- 2.0x = 100% slower (twice as slow)
- 3.0x = 200% slower (three times as slow)

## Choosing a Router

### High Performance Applications
- **HttpRouter**: Maximum speed, minimal features
- **FastRouter**: Great balance of speed and features
- **Chi**: Lightweight with good performance

### Full-Featured Applications
- **Gin**: Popular framework with good performance
- **Echo**: Modern framework with decent performance
- **FastRouter**: High performance with OpenAPI support

### Legacy Applications
- **GorillaMux**: Mature but slower, lots of features

## Common Benchmark Patterns

### Static Route Performance
```
BenchmarkStaticRoutes/FastRouter_/-8    20000000    85.2 ns/op
```
- 20M operations per second
- 85.2 nanoseconds per operation
- 8 = GOMAXPROCS

### Parameter Route Performance
```
BenchmarkParameterRoutes/FastRouter_/users/123-8    10000000    125.5 ns/op    32 B/op    1 allocs/op
```
- Parameter extraction adds overhead
- Memory allocation for parameter storage
- Still very fast for most applications

### Memory Pressure
```
BenchmarkMemoryAllocations/FastRouter-8    5000000    95.2 ns/op    0 B/op    0 allocs/op
```
- Zero allocations = no garbage collection pressure
- Important for high-throughput applications

## Performance Tips

1. **Use Static Routes When Possible**: Always faster than parameters
2. **Minimize Middleware**: Each middleware adds overhead
3. **Pool Objects**: Reuse objects to reduce allocations
4. **Benchmark Your Use Case**: Synthetic benchmarks != real world
5. **Profile Memory**: Use -memprofile to find allocation hotspots

## Hardware Impact

Results vary significantly based on:
- **CPU**: Newer processors are faster
- **Memory**: More RAM = less GC pressure
- **Architecture**: ARM vs x86 performance differs
- **Go Version**: Newer Go versions often faster

## Troubleshooting Slow Benchmarks

1. **Close other applications**: Reduce system load
2. **Use consistent hardware**: Don't compare across machines
3. **Run multiple times**: Use -count=5 for reliability
4. **Check thermal throttling**: CPU may slow down when hot
5. **Disable power management**: Use maximum CPU frequency

=============================================================================
*/
