#!/bin/bash

echo "Running Router Benchmarks..."
echo "============================"

# Create results directory
mkdir -p results

# Function to run benchmark and save results
run_benchmark() {
    local name=$1
    local pattern=$2
    local output_file="results/${name}.txt"

    echo "Running $name benchmark..."
    go test -bench="$pattern" -benchmem -count=3 -timeout=30m > "$output_file" 2>&1
    echo "Results saved to $output_file"
}

# Run different benchmark categories
run_benchmark "static_routes" "BenchmarkStaticRoutes"
run_benchmark "parameter_routes" "BenchmarkParameterRoutes"
run_benchmark "wildcard_routes" "BenchmarkWildcardRoutes"
run_benchmark "middleware" "BenchmarkWithMiddleware"
run_benchmark "many_routes" "BenchmarkManyRoutes"
run_benchmark "memory" "BenchmarkMemoryAllocations"
run_benchmark "concurrent" "BenchmarkConcurrentRequests"
run_benchmark "mixed_workload" "BenchmarkMixedWorkload"

# Run all benchmarks together for comparison
echo "Running comprehensive benchmark..."
go test -bench=. -benchmem -count=3 -timeout=60m > results/complete_results.txt 2>&1

echo "All benchmarks completed!"
echo "Results are available in the 'results' directory"

# Generate summary report
echo "Generating summary report..."
go run analysis.go > results/summary_report.txt 2>&1

echo "Summary report generated: results/summary_report.txt"
