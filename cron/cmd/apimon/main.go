package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmagar/nugs/cron/internal/api"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "stats":
		showStats()
	case "reset":
		resetStats()
	case "logs":
		showRecentLogs()
	case "errors":
		showRecentErrors()
	case "stop":
		enableEmergencyStop()
	case "start":
		disableEmergencyStop()
	case "status":
		showStatus()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("API Monitor Utility")
	fmt.Println("")
	fmt.Println("Usage: api_monitor <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  stats   - Show current API usage statistics")
	fmt.Println("  reset   - Reset API counters and circuit breaker")
	fmt.Println("  logs    - Show recent API request logs")
	fmt.Println("  errors  - Show recent API errors")
	fmt.Println("  stop    - Enable emergency stop (creates STOP_API file)")
	fmt.Println("  start   - Disable emergency stop (removes STOP_API file)")
	fmt.Println("  status  - Show overall system status")
}

func showStats() {
	client := api.NewSafeAPIClient()
	stats := client.GetStats()

	fmt.Println("=== API Usage Statistics ===")
	fmt.Printf("Date: %s\n", stats.CurrentDate)
	fmt.Printf("Time: %02d:%02d\n", stats.CurrentHour, stats.CurrentMinute)
	fmt.Println("")

	fmt.Printf("Requests Today: %d / %d\n", stats.TotalRequestsToday, 5000)
	fmt.Printf("Requests This Hour: %d / %d\n", stats.RequestsThisHour, 500)
	fmt.Printf("Requests This Minute: %d / %d\n", stats.RequestsThisMinute, 30)
	fmt.Printf("Last Request: %s\n", stats.LastRequestTime)
	fmt.Println("")

	fmt.Printf("Circuit Breaker: %s\n", getBreakerStatus(stats.CircuitBreakerOpen))
	fmt.Printf("Consecutive Errors: %d / %d\n", stats.ConsecutiveErrors, 5)
	fmt.Println("")

	if len(stats.Endpoints) > 0 {
		fmt.Println("=== Per-Endpoint Statistics ===")

		// Sort endpoints by request count
		type endpointStat struct {
			Name     string
			Count    int
			Errors   int
			ErrorPct float64
		}

		var endpoints []endpointStat
		for name, stat := range stats.Endpoints {
			errorPct := 0.0
			if stat.Count > 0 {
				errorPct = float64(stat.Errors) / float64(stat.Count) * 100
			}
			endpoints = append(endpoints, endpointStat{
				Name:     name,
				Count:    stat.Count,
				Errors:   stat.Errors,
				ErrorPct: errorPct,
			})
		}

		sort.Slice(endpoints, func(i, j int) bool {
			return endpoints[i].Count > endpoints[j].Count
		})

		fmt.Printf("%-30s %8s %8s %8s\n", "Endpoint", "Requests", "Errors", "Error %")
		fmt.Println(strings.Repeat("-", 70))
		for _, ep := range endpoints {
			fmt.Printf("%-30s %8d %8d %7.1f%%\n", ep.Name, ep.Count, ep.Errors, ep.ErrorPct)
		}
	}
}

func resetStats() {
	fmt.Print("Are you sure you want to reset all API statistics? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" {
		client := api.NewSafeAPIClient()
		client.ResetStats()
		fmt.Println("API statistics reset successfully.")
	} else {
		fmt.Println("Reset cancelled.")
	}
}

func showRecentLogs() {
	logDir := "logs/api_logs"
	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(logDir, fmt.Sprintf("api_requests_%s.log", today))

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		fmt.Println("No log file found for today.")
		return
	}

	// Read last 50 lines
	lines, err := readLastLines(logFile, 50)
	if err != nil {
		fmt.Printf("Error reading log file: %v\n", err)
		return
	}

	fmt.Printf("=== Recent API Requests (last %d) ===\n", len(lines))
	fmt.Printf("%-20s %-25s %-6s %-4s %-6s %s\n", "Time", "Endpoint", "Method", "Code", "Time", "Error")
	fmt.Println(strings.Repeat("-", 100))

	for _, line := range lines {
		var entry api.APILogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		timestamp, _ := time.Parse(time.RFC3339, entry.Timestamp)
		timeStr := timestamp.Format("15:04:05")

		errorStr := entry.Error
		if len(errorStr) > 30 {
			errorStr = errorStr[:27] + "..."
		}

		fmt.Printf("%-20s %-25s %-6s %-4d %6dms %s\n",
			timeStr, entry.Endpoint, entry.Method, entry.ResponseCode, entry.ResponseTime, errorStr)
	}
}

func showRecentErrors() {
	logDir := "logs/api_logs"
	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(logDir, fmt.Sprintf("api_requests_%s.log", today))

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		fmt.Println("No log file found for today.")
		return
	}

	file, err := os.Open(logFile)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return
	}
	defer file.Close()

	var errors []api.APILogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry api.APILogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		if entry.Error != "" || entry.ResponseCode >= 400 {
			errors = append(errors, entry)
		}
	}

	if len(errors) == 0 {
		fmt.Println("No errors found in today's logs.")
		return
	}

	// Show last 20 errors
	start := 0
	if len(errors) > 20 {
		start = len(errors) - 20
	}

	fmt.Printf("=== Recent API Errors (last %d) ===\n", len(errors)-start)
	fmt.Printf("%-20s %-25s %-4s %s\n", "Time", "Endpoint", "Code", "Error")
	fmt.Println(strings.Repeat("-", 90))

	for i := start; i < len(errors); i++ {
		entry := errors[i]
		timestamp, _ := time.Parse(time.RFC3339, entry.Timestamp)
		timeStr := timestamp.Format("15:04:05")

		errorStr := entry.Error
		if len(errorStr) > 50 {
			errorStr = errorStr[:47] + "..."
		}

		fmt.Printf("%-20s %-25s %-4d %s\n", timeStr, entry.Endpoint, entry.ResponseCode, errorStr)
	}
}

func enableEmergencyStop() {
	file, err := os.Create("configs/STOP_API")
	if err != nil {
		fmt.Printf("Error creating emergency stop file: %v\n", err)
		return
	}
	defer file.Close()

	timestamp := time.Now().Format(time.RFC3339)
	file.WriteString(fmt.Sprintf("Emergency stop enabled at: %s\n", timestamp))

	fmt.Println("Emergency stop ENABLED - All API requests will be blocked!")
	fmt.Println("Use 'api_monitor start' to re-enable API requests.")
}

func disableEmergencyStop() {
	if _, err := os.Stat("configs/STOP_API"); os.IsNotExist(err) {
		fmt.Println("Emergency stop is not currently enabled.")
		return
	}

	err := os.Remove("configs/STOP_API")
	if err != nil {
		fmt.Printf("Error removing emergency stop file: %v\n", err)
		return
	}

	fmt.Println("Emergency stop DISABLED - API requests are now allowed.")
}

func showStatus() {
	client := api.NewSafeAPIClient()
	stats := client.GetStats()

	fmt.Println("=== System Status ===")

	// Emergency stop status
	if _, err := os.Stat("configs/STOP_API"); err == nil {
		fmt.Println("Emergency Stop: ENABLED ⛔")
	} else {
		fmt.Println("Emergency Stop: DISABLED ✓")
	}

	// Circuit breaker status
	fmt.Printf("Circuit Breaker: %s\n", getBreakerStatus(stats.CircuitBreakerOpen))

	// Rate limit status
	rateLimitStatus := "✓ OK"
	if stats.RequestsThisMinute >= 25 { // 25/30 = 83% threshold
		rateLimitStatus = "⚠️ HIGH"
	}
	if stats.RequestsThisMinute >= 30 {
		rateLimitStatus = "⛔ LIMIT"
	}
	fmt.Printf("Rate Limits: %s (%d/30 per min, %d/500 per hour, %d/5000 per day)\n",
		rateLimitStatus, stats.RequestsThisMinute, stats.RequestsThisHour, stats.TotalRequestsToday)

	// Error rate
	errorRate := 0.0
	totalRequests := 0
	totalErrors := 0
	for _, ep := range stats.Endpoints {
		totalRequests += ep.Count
		totalErrors += ep.Errors
	}
	if totalRequests > 0 {
		errorRate = float64(totalErrors) / float64(totalRequests) * 100
	}

	errorStatus := "✓ OK"
	if errorRate > 5 {
		errorStatus = "⚠️ HIGH"
	}
	if errorRate > 20 {
		errorStatus = "⛔ CRITICAL"
	}
	fmt.Printf("Error Rate: %s (%.1f%%)\n", errorStatus, errorRate)

	// Log file status
	logDir := "logs/api_logs"
	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(logDir, fmt.Sprintf("api_requests_%s.log", today))

	if stat, err := os.Stat(logFile); err == nil {
		fmt.Printf("Log File: %s (%.1f KB)\n", logFile, float64(stat.Size())/1024)
	} else {
		fmt.Printf("Log File: No logs for today\n")
	}

	// Configuration
	config := api.LoadAPIConfig()
	fmt.Printf("Config: %d/min, %d/hr, %d/day limits\n",
		config.MaxRequestsPerMinute, config.MaxRequestsPerHour, config.MaxRequestsPerDay)
}

func getBreakerStatus(open bool) string {
	if open {
		return "⛔ OPEN (blocking requests)"
	}
	return "✓ CLOSED (allowing requests)"
}

func readLastLines(filename string, n int) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return last n lines
	if len(lines) > n {
		return lines[len(lines)-n:], nil
	}

	return lines, nil
}
