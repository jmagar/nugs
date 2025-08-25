package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// APIConfig holds configuration for API safety features
type APIConfig struct {
	MaxRequestsPerMinute int    `json:"max_requests_per_minute"`
	MaxRequestsPerHour   int    `json:"max_requests_per_hour"`
	MaxRequestsPerDay    int    `json:"max_requests_per_day"`
	MaxConsecutiveErrors int    `json:"max_consecutive_errors"`
	RetryDelaySeconds    int    `json:"retry_delay_seconds"`
	RetryMaxAttempts     int    `json:"retry_max_attempts"`
	EnableEmergencyStop  bool   `json:"enable_emergency_stop"`
	LogDirectory         string `json:"log_directory"`
}

// APIStats tracks API usage statistics
type APIStats struct {
	TotalRequestsToday int                      `json:"total_requests_today"`
	RequestsThisHour   int                      `json:"requests_this_hour"`
	RequestsThisMinute int                      `json:"requests_this_minute"`
	LastRequestTime    string                   `json:"last_request_time"`
	Endpoints          map[string]EndpointStats `json:"endpoints"`
	ConsecutiveErrors  int                      `json:"consecutive_errors"`
	CircuitBreakerOpen bool                     `json:"circuit_breaker_open"`
	CurrentDate        string                   `json:"current_date"`
	CurrentHour        int                      `json:"current_hour"`
	CurrentMinute      int                      `json:"current_minute"`
}

// EndpointStats tracks per-endpoint statistics
type EndpointStats struct {
	Count  int `json:"count"`
	Errors int `json:"errors"`
}

// APILogEntry represents a single API request log entry
type APILogEntry struct {
	Timestamp    string `json:"timestamp"`
	Endpoint     string `json:"endpoint"`
	Method       string `json:"method"`
	ResponseCode int    `json:"response_code"`
	ResponseTime int64  `json:"response_time_ms"`
	Error        string `json:"error,omitempty"`
}

// SafeAPIClient provides rate-limited, logged API access
type SafeAPIClient struct {
	config     *APIConfig
	stats      *APIStats
	mutex      sync.Mutex
	httpClient *http.Client
	token      string
}

// NewSafeAPIClient creates a new safe API client
func NewSafeAPIClient() *SafeAPIClient {
	config := LoadAPIConfig()
	stats := loadAPIStats()

	// Ensure log directory exists
	os.MkdirAll(config.LogDirectory, 0755)

	return &SafeAPIClient{
		config: config,
		stats:  stats,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Authenticate with Nugs.net API
func (c *SafeAPIClient) Authenticate(email, password string) error {
	// First login call
	loginURL := fmt.Sprintf("https://streamapi.nugs.net/api.aspx?method=user.site.login&pw=%s&username=%s",
		strings.Replace(url.QueryEscape(password), "+", "%20", -1),
		url.QueryEscape(email))

	_, err := c.safeGet(loginURL, "user.site.login")
	if err != nil {
		return err
	}

	// Second call to get the actual token
	tokenURL := fmt.Sprintf("https://streamapi.nugs.net/secureapi.aspx?method=user.site.login&pw=%s&username=%s",
		strings.Replace(url.QueryEscape(password), "+", "%20", -1),
		url.QueryEscape(email))

	body, err := c.safeGet(tokenURL, "user.site.login.secure")
	if err != nil {
		return err
	}

	// Parse the response to get the token
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}

	// Navigate through the nested structure
	if response, ok := result["Response"].(map[string]interface{}); ok {
		if token, ok := response["secureAuthenticationString"].(string); ok && token != "" {
			c.token = token
			return nil
		}
	}

	return fmt.Errorf("authentication failed - could not extract token")
}

// GetArtistCatalog fetches the full artist catalog
func (c *SafeAPIClient) GetArtistCatalog() ([]byte, error) {
	return c.safeGet("https://streamapi.nugs.net/api.aspx?method=catalog.artists", "catalog.artists")
}

// GetArtistShows fetches all shows for a specific artist
func (c *SafeAPIClient) GetArtistShows(artistID int) ([]byte, error) {
	if c.token == "" {
		return nil, fmt.Errorf("not authenticated - call Authenticate() first")
	}

	showsURL := fmt.Sprintf("https://streamapi.nugs.net/api.aspx?method=catalog.containersAll&artistList=%d&availableOnly=1&token=%s",
		artistID, c.token)

	return c.safeGet(showsURL, "catalog.containersAll")
}

// GetFullCatalog fetches the complete catalog (no authentication needed)
func (c *SafeAPIClient) GetFullCatalog() ([]byte, error) {
	return c.safeGet("https://streamapi.nugs.net/api.aspx?method=catalog.containersAll&availableOnly=1", "catalog.containersAll.full")
}

// safeGet performs a safe HTTP GET with all safety features
func (c *SafeAPIClient) safeGet(url, endpoint string) ([]byte, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	startTime := time.Now()

	// Check emergency stop
	if c.config.EnableEmergencyStop {
		if _, err := os.Stat("configs/STOP_API"); err == nil {
			return nil, fmt.Errorf("API calls stopped by emergency stop file")
		}
	}

	// Check circuit breaker
	if c.stats.CircuitBreakerOpen {
		// Try to recover after 5 minutes
		if lastReq, err := time.Parse(time.RFC3339, c.stats.LastRequestTime); err == nil {
			if time.Since(lastReq) > 5*time.Minute {
				c.stats.CircuitBreakerOpen = false
				c.stats.ConsecutiveErrors = 0
				log.Println("Circuit breaker reset - attempting recovery")
			} else {
				return nil, fmt.Errorf("circuit breaker open - too many consecutive errors")
			}
		}
	}

	// Check rate limits
	if err := c.checkRateLimits(); err != nil {
		return nil, err
	}

	// Update counters before request
	c.updateRequestCounters()

	// Make the actual HTTP request with retries
	var body []byte
	var lastError error

	for attempt := 1; attempt <= c.config.RetryMaxAttempts; attempt++ {
		resp, err := c.httpClient.Get(url)
		responseTime := time.Since(startTime).Milliseconds()

		logEntry := APILogEntry{
			Timestamp:    time.Now().Format(time.RFC3339),
			Endpoint:     endpoint,
			Method:       "GET",
			ResponseTime: responseTime,
		}

		if err != nil {
			logEntry.Error = err.Error()
			logEntry.ResponseCode = 0
			c.logRequest(logEntry)

			lastError = err
			c.handleError(endpoint)

			if attempt < c.config.RetryMaxAttempts {
				backoff := time.Duration(c.config.RetryDelaySeconds*attempt) * time.Second
				log.Printf("Request failed (attempt %d/%d), retrying in %v: %v",
					attempt, c.config.RetryMaxAttempts, backoff, err)
				time.Sleep(backoff)
				startTime = time.Now() // Reset timing for retry
				continue
			}
			break
		}

		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)

		logEntry.ResponseCode = resp.StatusCode

		if err != nil {
			logEntry.Error = err.Error()
			c.logRequest(logEntry)
			lastError = err
			continue
		}

		if resp.StatusCode != 200 {
			logEntry.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
			c.logRequest(logEntry)
			c.handleError(endpoint)
			lastError = fmt.Errorf("HTTP %d", resp.StatusCode)

			if attempt < c.config.RetryMaxAttempts {
				backoff := time.Duration(c.config.RetryDelaySeconds*attempt) * time.Second
				log.Printf("HTTP error %d (attempt %d/%d), retrying in %v",
					resp.StatusCode, attempt, c.config.RetryMaxAttempts, backoff)
				time.Sleep(backoff)
				startTime = time.Now()
				continue
			}
			break
		}

		// Success
		c.logRequest(logEntry)
		c.handleSuccess(endpoint)
		c.saveAPIStats()
		return body, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %v", c.config.RetryMaxAttempts, lastError)
}

// checkRateLimits verifies we haven't exceeded any rate limits
func (c *SafeAPIClient) checkRateLimits() error {
	now := time.Now()

	// Reset counters if needed
	currentDate := now.Format("2006-01-02")
	currentHour := now.Hour()
	currentMinute := now.Minute()

	if c.stats.CurrentDate != currentDate {
		c.stats.TotalRequestsToday = 0
		c.stats.RequestsThisHour = 0
		c.stats.RequestsThisMinute = 0
		c.stats.CurrentDate = currentDate
		c.stats.CurrentHour = currentHour
		c.stats.CurrentMinute = currentMinute
	} else if c.stats.CurrentHour != currentHour {
		c.stats.RequestsThisHour = 0
		c.stats.RequestsThisMinute = 0
		c.stats.CurrentHour = currentHour
		c.stats.CurrentMinute = currentMinute
	} else if c.stats.CurrentMinute != currentMinute {
		c.stats.RequestsThisMinute = 0
		c.stats.CurrentMinute = currentMinute
	}

	// Check limits
	if c.stats.RequestsThisMinute >= c.config.MaxRequestsPerMinute {
		return fmt.Errorf("rate limit exceeded: %d requests this minute (max: %d)",
			c.stats.RequestsThisMinute, c.config.MaxRequestsPerMinute)
	}

	if c.stats.RequestsThisHour >= c.config.MaxRequestsPerHour {
		return fmt.Errorf("rate limit exceeded: %d requests this hour (max: %d)",
			c.stats.RequestsThisHour, c.config.MaxRequestsPerHour)
	}

	if c.stats.TotalRequestsToday >= c.config.MaxRequestsPerDay {
		return fmt.Errorf("rate limit exceeded: %d requests today (max: %d)",
			c.stats.TotalRequestsToday, c.config.MaxRequestsPerDay)
	}

	return nil
}

// updateRequestCounters increments all request counters
func (c *SafeAPIClient) updateRequestCounters() {
	c.stats.TotalRequestsToday++
	c.stats.RequestsThisHour++
	c.stats.RequestsThisMinute++
	c.stats.LastRequestTime = time.Now().Format(time.RFC3339)
}

// handleError processes API errors and updates circuit breaker
func (c *SafeAPIClient) handleError(endpoint string) {
	c.stats.ConsecutiveErrors++

	if c.stats.Endpoints == nil {
		c.stats.Endpoints = make(map[string]EndpointStats)
	}

	stats := c.stats.Endpoints[endpoint]
	stats.Errors++
	c.stats.Endpoints[endpoint] = stats

	// Open circuit breaker if too many consecutive errors
	if c.stats.ConsecutiveErrors >= c.config.MaxConsecutiveErrors {
		c.stats.CircuitBreakerOpen = true
		log.Printf("Circuit breaker opened after %d consecutive errors", c.stats.ConsecutiveErrors)
	}
}

// handleSuccess processes successful API responses
func (c *SafeAPIClient) handleSuccess(endpoint string) {
	c.stats.ConsecutiveErrors = 0
	c.stats.CircuitBreakerOpen = false

	if c.stats.Endpoints == nil {
		c.stats.Endpoints = make(map[string]EndpointStats)
	}

	stats := c.stats.Endpoints[endpoint]
	stats.Count++
	c.stats.Endpoints[endpoint] = stats
}

// logRequest writes a request log entry to the daily log file
func (c *SafeAPIClient) logRequest(entry APILogEntry) {
	logFile := filepath.Join(c.config.LogDirectory,
		fmt.Sprintf("api_requests_%s.log", time.Now().Format("2006-01-02")))

	logData, _ := json.Marshal(entry)
	logLine := string(logData) + "\n"

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to write API log: %v", err)
		return
	}
	defer file.Close()

	file.WriteString(logLine)
}

// LoadAPIConfig loads configuration from configs/api_config.json
func LoadAPIConfig() *APIConfig {
	config := &APIConfig{
		MaxRequestsPerMinute: 30,
		MaxRequestsPerHour:   500,
		MaxRequestsPerDay:    5000,
		MaxConsecutiveErrors: 5,
		RetryDelaySeconds:    2,
		RetryMaxAttempts:     3,
		EnableEmergencyStop:  true,
		LogDirectory:         "logs/api_logs",
	}

	if data, err := ioutil.ReadFile("configs/api_config.json"); err == nil {
		json.Unmarshal(data, config)
	}

	return config
}

// loadAPIStats loads statistics from data/api_stats.json
func loadAPIStats() *APIStats {
	stats := &APIStats{
		Endpoints:     make(map[string]EndpointStats),
		CurrentDate:   time.Now().Format("2006-01-02"),
		CurrentHour:   time.Now().Hour(),
		CurrentMinute: time.Now().Minute(),
	}

	if data, err := ioutil.ReadFile("data/api_stats.json"); err == nil {
		json.Unmarshal(data, stats)
	}

	return stats
}

// saveAPIStats saves current statistics to data/api_stats.json
func (c *SafeAPIClient) saveAPIStats() {
	data, _ := json.MarshalIndent(c.stats, "", "  ")
	ioutil.WriteFile("data/api_stats.json", data, 0644)
}

// GetStats returns current API statistics
func (c *SafeAPIClient) GetStats() *APIStats {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.stats
}

// ResetStats resets API statistics (for manual override)
func (c *SafeAPIClient) ResetStats() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.stats.TotalRequestsToday = 0
	c.stats.RequestsThisHour = 0
	c.stats.RequestsThisMinute = 0
	c.stats.ConsecutiveErrors = 0
	c.stats.CircuitBreakerOpen = false
	c.stats.Endpoints = make(map[string]EndpointStats)

	c.saveAPIStats()
	log.Println("API statistics reset")
}
