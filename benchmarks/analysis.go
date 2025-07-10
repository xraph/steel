package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type BenchmarkResult struct {
	Name        string
	Router      string
	NsPerOp     float64
	AllocsPerOp int64
	BytesPerOp  int64
}

func main() {
	results, err := parseBenchmarkResults("results/complete_results.txt")
	if err != nil {
		fmt.Printf("Error parsing results: %v\n", err)
		return
	}

	generateReport(results)
}

func parseBenchmarkResults(filename string) ([]BenchmarkResult, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []BenchmarkResult
	scanner := bufio.NewScanner(file)

	// Regex to parse benchmark lines
	re := regexp.MustCompile(`^Benchmark(\w+)(?:_(.+?))?-\d+\s+\d+\s+([\d.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)

		if len(matches) >= 6 {
			nsPerOp, _ := strconv.ParseFloat(matches[3], 64)
			bytesPerOp, _ := strconv.ParseInt(matches[4], 10, 64)
			allocsPerOp, _ := strconv.ParseInt(matches[5], 10, 64)

			result := BenchmarkResult{
				Name:        matches[1],
				Router:      extractRouter(matches[0]),
				NsPerOp:     nsPerOp,
				BytesPerOp:  bytesPerOp,
				AllocsPerOp: allocsPerOp,
			}
			results = append(results, result)
		}
	}

	return results, scanner.Err()
}

func extractRouter(benchmarkName string) string {
	routers := []string{"Steel", "Chi", "Gin", "Fiber", "Echo", "HttpRouter", "GorillaMux"}

	for _, router := range routers {
		if strings.Contains(benchmarkName, router) {
			return router
		}
	}
	return "Unknown"
}

func generateReport(results []BenchmarkResult) {
	fmt.Println("Router Benchmark Comparison Report")
	fmt.Println("==================================")
	fmt.Println()

	// Group results by benchmark type
	benchmarkGroups := make(map[string][]BenchmarkResult)
	for _, result := range results {
		benchmarkGroups[result.Name] = append(benchmarkGroups[result.Name], result)
	}

	// Generate reports for each benchmark type
	for benchName, benchResults := range benchmarkGroups {
		if len(benchResults) < 2 {
			continue
		}

		fmt.Printf("### %s\n", benchName)
		fmt.Println()

		// Sort by performance (ns/op)
		sort.Slice(benchResults, func(i, j int) bool {
			return benchResults[i].NsPerOp < benchResults[j].NsPerOp
		})

		fmt.Printf("%-12s %-12s %-12s %-12s %-12s\n", "Router", "ns/op", "B/op", "allocs/op", "Relative")
		fmt.Println(strings.Repeat("-", 70))

		fastest := benchResults[0].NsPerOp

		for _, result := range benchResults {
			relative := result.NsPerOp / fastest
			fmt.Printf("%-12s %-12.0f %-12d %-12d %.2fx\n",
				result.Router, result.NsPerOp, result.BytesPerOp, result.AllocsPerOp, relative)
		}
		fmt.Println()
	}

	// Overall performance summary
	fmt.Println("### Overall Performance Summary")
	fmt.Println()

	routerStats := make(map[string]*RouterStats)
	for _, result := range results {
		if routerStats[result.Router] == nil {
			routerStats[result.Router] = &RouterStats{}
		}
		stats := routerStats[result.Router]
		stats.TotalTests++
		stats.TotalNsPerOp += result.NsPerOp
		stats.TotalBytesPerOp += result.BytesPerOp
		stats.TotalAllocsPerOp += result.AllocsPerOp
	}

	type RouterSummary struct {
		Name           string
		AvgNsPerOp     float64
		AvgBytesPerOp  float64
		AvgAllocsPerOp float64
		TestCount      int
	}

	var summaries []RouterSummary
	for router, stats := range routerStats {
		if stats.TotalTests > 0 {
			summaries = append(summaries, RouterSummary{
				Name:           router,
				AvgNsPerOp:     stats.TotalNsPerOp / float64(stats.TotalTests),
				AvgBytesPerOp:  float64(stats.TotalBytesPerOp) / float64(stats.TotalTests),
				AvgAllocsPerOp: float64(stats.TotalAllocsPerOp) / float64(stats.TotalTests),
				TestCount:      stats.TotalTests,
			})
		}
	}

	// Sort by average performance
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].AvgNsPerOp < summaries[j].AvgNsPerOp
	})

	fmt.Printf("%-12s %-12s %-12s %-12s %-8s\n", "Router", "Avg ns/op", "Avg B/op", "Avg allocs/op", "Tests")
	fmt.Println(strings.Repeat("-", 65))

	for _, summary := range summaries {
		fmt.Printf("%-12s %-12.0f %-12.0f %-12.1f %-8d\n",
			summary.Name, summary.AvgNsPerOp, summary.AvgBytesPerOp,
			summary.AvgAllocsPerOp, summary.TestCount)
	}
	fmt.Println()

	// Performance insights
	if len(summaries) > 0 {
		fastest := summaries[0]
		fmt.Printf("**Winner: %s** - Fastest average performance at %.0f ns/op\n", fastest.Name, fastest.AvgNsPerOp)

		// Find most memory efficient
		sort.Slice(summaries, func(i, j int) bool {
			return summaries[i].AvgBytesPerOp < summaries[j].AvgBytesPerOp
		})
		memEfficient := summaries[0]
		fmt.Printf("**Most Memory Efficient: %s** - %.0f bytes/op average\n", memEfficient.Name, memEfficient.AvgBytesPerOp)

		// Find least allocations
		sort.Slice(summaries, func(i, j int) bool {
			return summaries[i].AvgAllocsPerOp < summaries[j].AvgAllocsPerOp
		})
		leastAllocs := summaries[0]
		fmt.Printf("**Fewest Allocations: %s** - %.1f allocs/op average\n", leastAllocs.Name, leastAllocs.AvgAllocsPerOp)
	}
}

type RouterStats struct {
	TotalTests       int
	TotalNsPerOp     float64
	TotalBytesPerOp  int64
	TotalAllocsPerOp int64
	WinCount         int
}
