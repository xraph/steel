package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ChartData represents data for chart visualization
type ChartData struct {
	Labels   []string  `json:"labels"`
	Datasets []Dataset `json:"datasets"`
}

type Dataset struct {
	Label           string    `json:"label"`
	Data            []float64 `json:"data"`
	BackgroundColor []string  `json:"backgroundColor"`
	BorderColor     []string  `json:"borderColor"`
	BorderWidth     int       `json:"borderWidth"`
}

type BenchmarkSummary struct {
	Category string
	Results  []RouterResult
}

type RouterResult struct {
	Router      string
	NsPerOp     float64
	BytesPerOp  int64
	AllocsPerOp int64
	Relative    float64
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run visualization.go <benchmark_results_file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	summaries, err := parseBenchmarkFile(filename)
	if err != nil {
		fmt.Printf("Error parsing benchmark file: %v\n", err)
		os.Exit(1)
	}

	// Generate HTML report with charts
	err = generateHTMLReport(summaries)
	if err != nil {
		fmt.Printf("Error generating HTML report: %v\n", err)
		os.Exit(1)
	}

	// Generate JSON data for external tools
	err = generateJSONData(summaries)
	if err != nil {
		fmt.Printf("Error generating JSON data: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Visualization files generated:")
	fmt.Println("- benchmark_report.html (Interactive charts)")
	fmt.Println("- benchmark_data.json (Raw data)")
	fmt.Println("- performance_summary.txt (Text summary)")
}

func parseBenchmarkFile(filename string) ([]BenchmarkSummary, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	benchmarkMap := make(map[string][]RouterResult)
	scanner := bufio.NewScanner(file)

	// Enhanced regex to capture more benchmark formats
	re := regexp.MustCompile(`^Benchmark([^/]+)(?:/([^-]+))?-\d+\s+\d+\s+([\d.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op`)

	for scanner.Scan() {
		line := scanner.Text()
		if matches := re.FindStringSubmatch(line); len(matches) >= 6 {
			category := matches[1]
			router := extractRouterFromBenchmark(matches[0])

			nsPerOp, _ := strconv.ParseFloat(matches[3], 64)
			bytesPerOp, _ := strconv.ParseInt(matches[4], 10, 64)
			allocsPerOp, _ := strconv.ParseInt(matches[5], 10, 64)

			result := RouterResult{
				Router:      router,
				NsPerOp:     nsPerOp,
				BytesPerOp:  bytesPerOp,
				AllocsPerOp: allocsPerOp,
			}

			benchmarkMap[category] = append(benchmarkMap[category], result)
		}
	}

	// Convert map to slice and calculate relative performance
	var summaries []BenchmarkSummary
	for category, results := range benchmarkMap {
		// Sort by performance and calculate relative speeds
		sort.Slice(results, func(i, j int) bool {
			return results[i].NsPerOp < results[j].NsPerOp
		})

		if len(results) > 0 {
			fastest := results[0].NsPerOp
			for i := range results {
				results[i].Relative = results[i].NsPerOp / fastest
			}
		}

		summaries = append(summaries, BenchmarkSummary{
			Category: category,
			Results:  results,
		})
	}

	return summaries, scanner.Err()
}

func extractRouterFromBenchmark(benchmarkName string) string {
	routers := []string{"ForgeRouter", "Chi", "Gin", "Fiber", "Echo", "HttpRouter", "GorillaMux"}

	for _, router := range routers {
		if strings.Contains(benchmarkName, router) {
			return router
		}
	}
	return "Unknown"
}

func generateHTMLReport(summaries []BenchmarkSummary) error {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Router Benchmark Results</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f6fa;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .header {
            text-align: center;
            margin-bottom: 40px;
            padding: 20px;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .chart-container {
            background: white;
            margin: 20px 0;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .chart-wrapper {
            position: relative;
            height: 400px;
            margin: 20px 0;
        }
        .summary-table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        .summary-table th,
        .summary-table td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        .summary-table th {
            background-color: #f8f9fa;
            font-weight: 600;
        }
        .summary-table tr:hover {
            background-color: #f5f5f5;
        }
        .winner {
            background-color: #d4edda !important;
            font-weight: bold;
        }
        .metric-cards {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }
        .metric-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            text-align: center;
        }
        .metric-value {
            font-size: 2em;
            font-weight: bold;
            color: #007bff;
        }
        .metric-label {
            color: #666;
            margin-top: 5px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Router Benchmark Results</h1>
            <p>Performance comparison of Go HTTP routers</p>
        </div>

        <div class="metric-cards">
            <div class="metric-card">
                <div class="metric-value">{{.TotalBenchmarks}}</div>
                <div class="metric-label">Total Benchmarks</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">{{.TotalRouters}}</div>
                <div class="metric-label">Routers Tested</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">{{.FastestRouter}}</div>
                <div class="metric-label">Overall Fastest</div>
            </div>
        </div>

        {{range .Summaries}}
        <div class="chart-container">
            <h2>{{.Category}} Performance</h2>
            
            <div class="chart-wrapper">
                <canvas id="chart-{{.Category}}"></canvas>
            </div>

            <table class="summary-table">
                <thead>
                    <tr>
                        <th>Router</th>
                        <th>ns/op</th>
                        <th>B/op</th>
                        <th>allocs/op</th>
                        <th>Relative Speed</th>
                    </tr>
                </thead>
                <tbody>
                    {{range $i, $result := .Results}}
                    <tr {{if eq $i 0}}class="winner"{{end}}>
                        <td>{{$result.Router}}</td>
                        <td>{{printf "%.1f" $result.NsPerOp}}</td>
                        <td>{{$result.BytesPerOp}}</td>
                        <td>{{$result.AllocsPerOp}}</td>
                        <td>{{printf "%.2fx" $result.Relative}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{end}}
    </div>

    <script>
        // Chart.js configuration
        Chart.defaults.responsive = true;
        Chart.defaults.maintainAspectRatio = false;

        const chartColors = [
            '#FF6384', '#36A2EB', '#FFCE56', '#4BC0C0',
            '#9966FF', '#FF9F40', '#FF6384', '#C9CBCF'
        ];

        {{range .Summaries}}
        // Chart for {{.Category}}
        const ctx{{.Category}} = document.getElementById('chart-{{.Category}}').getContext('2d');
        new Chart(ctx{{.Category}}, {
            type: 'bar',
            data: {
                labels: [{{range .Results}}'{{.Router}}',{{end}}],
                datasets: [{
                    label: 'ns/op',
                    data: [{{range .Results}}{{.NsPerOp}},{{end}}],
                    backgroundColor: chartColors.slice(0, {{len .Results}}),
                    borderColor: chartColors.slice(0, {{len .Results}}),
                    borderWidth: 1
                }]
            },
            options: {
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Nanoseconds per Operation'
                        }
                    }
                },
                plugins: {
                    title: {
                        display: true,
                        text: '{{.Category}} - Lower is Better'
                    },
                    legend: {
                        display: false
                    }
                }
            }
        });
        {{end}}
    </script>
</body>
</html>`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return err
	}

	// Calculate summary statistics
	totalBenchmarks := len(summaries)
	routerSet := make(map[string]bool)
	var fastestRouter string
	bestAverage := float64(^uint(0) >> 1) // Max float64

	routerTotals := make(map[string]float64)
	routerCounts := make(map[string]int)

	for _, summary := range summaries {
		for _, result := range summary.Results {
			routerSet[result.Router] = true
			routerTotals[result.Router] += result.NsPerOp
			routerCounts[result.Router]++
		}
	}

	// Find overall fastest router by average
	for router, total := range routerTotals {
		average := total / float64(routerCounts[router])
		if average < bestAverage {
			bestAverage = average
			fastestRouter = router
		}
	}

	data := struct {
		Summaries       []BenchmarkSummary
		TotalBenchmarks int
		TotalRouters    int
		FastestRouter   string
	}{
		Summaries:       summaries,
		TotalBenchmarks: totalBenchmarks,
		TotalRouters:    len(routerSet),
		FastestRouter:   fastestRouter,
	}

	file, err := os.Create("benchmark_report.html")
	if err != nil {
		return err
	}
	defer file.Close()

	return t.Execute(file, data)
}

func generateJSONData(summaries []BenchmarkSummary) error {
	// Convert to JSON-friendly format
	data := make(map[string]interface{})
	data["summaries"] = summaries
	data["timestamp"] = fmt.Sprintf("%d", os.Getpid()) // Simple timestamp

	file, err := os.Create("benchmark_data.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// Additional analysis functions...

func generatePerformanceSummary(summaries []BenchmarkSummary) error {
	file, err := os.Create("performance_summary.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "Router Performance Summary")
	fmt.Fprintln(file, "==========================")
	fmt.Fprintln(file)

	// Overall rankings
	routerStats := make(map[string]*RouterStats)
	for _, summary := range summaries {
		for _, result := range summary.Results {
			if routerStats[result.Router] == nil {
				routerStats[result.Router] = &RouterStats{}
			}
			stats := routerStats[result.Router]
			stats.TotalTests++
			stats.TotalNsPerOp += result.NsPerOp
			stats.TotalBytesPerOp += result.BytesPerOp
			stats.TotalAllocsPerOp += result.AllocsPerOp
			stats.WinCount += getWinCount(result, summary.Results)
		}
	}

	// Calculate averages and sort
	type RouterSummary struct {
		Name           string
		AvgNsPerOp     float64
		AvgBytesPerOp  float64
		AvgAllocsPerOp float64
		WinPercentage  float64
		TestCount      int
	}

	var summariesList []RouterSummary
	for router, stats := range routerStats {
		if stats.TotalTests > 0 {
			summariesList = append(summariesList, RouterSummary{
				Name:           router,
				AvgNsPerOp:     stats.TotalNsPerOp / float64(stats.TotalTests),
				AvgBytesPerOp:  float64(stats.TotalBytesPerOp) / float64(stats.TotalTests),
				AvgAllocsPerOp: float64(stats.TotalAllocsPerOp) / float64(stats.TotalTests),
				WinPercentage:  float64(stats.WinCount) / float64(stats.TotalTests) * 100,
				TestCount:      stats.TotalTests,
			})
		}
	}

	// Sort by average performance
	sort.Slice(summariesList, func(i, j int) bool {
		return summariesList[i].AvgNsPerOp < summariesList[j].AvgNsPerOp
	})

	fmt.Fprintf(file, "%-12s %-12s %-12s %-12s %-8s %-8s\n",
		"Router", "Avg ns/op", "Avg B/op", "Avg allocs", "Win %", "Tests")
	fmt.Fprintln(file, strings.Repeat("-", 75))

	for i, summary := range summariesList {
		fmt.Fprintf(file, "%-12s %-12.1f %-12.1f %-12.1f %-8.1f %-8d\n",
			summary.Name, summary.AvgNsPerOp, summary.AvgBytesPerOp,
			summary.AvgAllocsPerOp, summary.WinPercentage, summary.TestCount)

		if i == 0 {
			fmt.Fprintf(file, "             ⭐ OVERALL WINNER\n")
		}
	}

	return nil
}

func getWinCount(result RouterResult, allResults []RouterResult) int {
	// Check if this result is the fastest in its category
	for _, other := range allResults {
		if other.NsPerOp < result.NsPerOp {
			return 0
		}
	}
	return 1
}

type RouterStats struct {
	TotalTests       int
	TotalNsPerOp     float64
	TotalBytesPerOp  int64
	TotalAllocsPerOp int64
	WinCount         int
}
