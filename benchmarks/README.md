# Router Benchmark Comparison

A comprehensive benchmark suite comparing FastRouter against popular Go HTTP routers including Chi, Gin, Fiber, Echo, HttpRouter, and Gorilla Mux.

## Routers Tested

| Router | Version | Description |
|--------|---------|-------------|
| **FastRouter** | v0.1.0 | Our custom high-performance router with OpenAPI support |
| **Chi** | v5.0.10 | Lightweight, idiomatic router for Go HTTP services |
| **Gin** | v1.9.1 | Popular web framework with focus on performance |
| **Fiber** | v2.52.0 | Express-inspired framework built on Fasthttp |
| **Echo** | v4.11.4 | High performance, minimalist web framework |
| **HttpRouter** | v1.3.0 | Lightning fast HTTP router with zero allocations |
| **Gorilla Mux** | v1.8.1 | Powerful URL router and dispatcher |

## Benchmark Categories

### 1. Static Routes
Tests routing performance for static paths without parameters:
- `/`
- `/users`
- `/users/profile`
- `/api/v1/status`
- `/api/v1/health`

### 2. Parameter Routes
Tests performance with URL parameters:
- `/users/:id`
- `/users/:id/posts`
- `/users/:id/posts/:postId`
- `/api/v1/users/:userId`
- `/api/v1/users/:userId/orders/:orderId`

### 3. Wildcard Routes
Tests wildcard/catch-all route performance:
- `/static/*`
- `/files/*`
- Various file paths

### 4. Middleware Performance
Tests overhead of middleware chains with a simple header-setting middleware.

### 5. Many Routes Scenario
Tests lookup performance with 1000+ routes to simulate real-world applications.

### 6. Memory Allocations
Focuses on memory efficiency and garbage collection impact.

### 7. Concurrent Requests
Tests performance under concurrent load using `b.RunParallel()`.

### 8. Mixed Workload
Simulates realistic traffic with mixed route types and HTTP methods.

## Setup and Usage

### Prerequisites
- Go 1.21 or later
- Unix-like environment (Linux, macOS, WSL)

### Installation

```bash
# Clone or download the benchmark files
git clone <repository-url>
cd router-benchmarks

# Make setup script executable
chmod +x benchmark_setup.sh

# Run setup
./benchmark_setup.sh

# Install dependencies
go mod tidy
```

### Running Benchmarks

```bash
# Run all benchmarks (takes 15-30 minutes)
./run_benchmarks.sh

# Run specific benchmark category
go test -bench=BenchmarkStaticRoutes -benchmem

# Run with more iterations for accuracy
go test -bench=. -benchmem -count=5

# Run with CPU profiling
go test -bench=BenchmarkMixedWorkload -benchmem -cpuprofile=cpu.prof

# Run with memory profiling  
go test -bench=BenchmarkMemoryAllocations -benchmem -memprofile=mem.prof
```

### Viewing Results

Results are saved in the `results/` directory:

- `complete_results.txt` - Raw benchmark output
- `summary_report.txt` - Processed analysis and rankings
- Individual category files (e.g., `static_routes.txt`)

## Expected Performance Characteristics

### Speed Rankings (Typical)
1. **HttpRouter** - Fastest, zero allocations
2. **FastRouter** - High performance with rich features
3. **Chi** - Fast and lightweight
4. **Gin** - Good performance with framework features
5. **Echo** - Solid performance
6. **Fiber** - Fast but different architecture (Fasthttp)
7. **Gorilla Mux** - Slower due to regex matching

### Memory Efficiency Rankings (Typical)
1. **HttpRouter** - Zero allocations for most routes
2. **FastRouter** - Efficient parameter pooling
3. **Chi** - Minimal allocations
4. **Gin** - Moderate allocations
5. **Echo** - Moderate allocations
6. **Gorilla Mux** - Higher allocations

### Feature Richness vs Performance

```
High Performance, High Features │ FastRouter
                                │ Gin, Echo
                                │
Low Performance, High Features  │ Gorilla Mux
                                └─────────────────
                                 Low Features → High Features
```

## Interpreting Results

### Key Metrics

- **ns/op**: Nanoseconds per operation (lower is better)
- **B/op**: Bytes allocated per operation (lower is better)
- **allocs/op**: Number of allocations per operation (lower is better)

### What to Look For

1. **Raw Speed**: How fast can the router handle requests?
2. **Memory Efficiency**: How much memory does each request consume?
3. **Scalability**: How does performance change with more routes?
4. **Consistency**: Are results consistent across different scenarios?

### Sample Output

```
BenchmarkStaticRoutes/FastRouter_/-8         20000000    85.2 ns/op    0 B/op    0 allocs/op
BenchmarkStaticRoutes/Chi_/-8                18000000    89.1 ns/op    0 B/op    0 allocs/op  
BenchmarkStaticRoutes/HttpRouter_/-8         25000000    68.3 ns/op    0 B/op    0 allocs/op
BenchmarkStaticRoutes/Gin_/-8                15000000    95.7 ns/op   32 B/op    1 allocs/op
```

## Optimization Notes

### FastRouter Optimizations

1. **Parameter Pooling**: Reuses parameter objects to reduce allocations
2. **Efficient Tree Structure**: Radix tree with optimized node traversal
3. **Minimal Overhead**: Lightweight context and parameter extraction
4. **Zero-Copy Operations**: Where possible, avoids copying data

### Benchmark Limitations

1. **Synthetic Workload**: Real applications have different patterns
2. **Handler Simplicity**: Actual handlers do more work
3. **No I/O**: Benchmarks don't include database/network calls
4. **Single Machine**: Results vary across different hardware

## Hardware Requirements

### Recommended Specs
- **CPU**: Multi-core processor (4+ cores)
- **RAM**: 8GB+ (benchmarks use significant memory)
- **Storage**: SSD for faster Go compilation

### Benchmark Duration
- Full suite: 30-60 minutes
- Individual categories: 2-10 minutes each
- Quick run: 5-10 minutes with reduced iterations

## Troubleshooting

### Common Issues

1. **Module Path Error**: Update `go.mod` with correct FastRouter import path
2. **Dependency Issues**: Run `go mod tidy` to resolve dependencies
3. **Out of Memory**: Reduce benchmark iterations or use machine with more RAM
4. **Timeout**: Increase timeout with `-timeout=60m` flag

### Performance Variations

Results can vary based on:
- CPU architecture and speed
- Available memory and system load
- Go version and compiler optimizations
- Operating system and scheduler behavior

## Contributing

To add new routers or benchmark scenarios:

1. Add router setup function following existing patterns
2. Include router in all relevant benchmark functions
3. Update documentation and expected results
4. Test with multiple runs to ensure consistency

## License

This benchmark suite is provided under the same license as FastRouter.

---

**Note**: Benchmarks are synthetic and may not reflect real-world performance. Always benchmark your specific use case and workload patterns.