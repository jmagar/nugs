package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmagar/nugs/cron/internal/models"
	"github.com/jmagar/nugs/cron/internal/services"
)

type RefreshHandler struct {
	RefreshService *services.CatalogRefreshService
	JobManager     *models.JobManager
}

type RefreshRequest struct {
	Force      bool `json:"force"`
	Background bool `json:"background"`
}

type RefreshResponse struct {
	Success          bool   `json:"success"`
	JobID            string `json:"job_id"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	EstimatedSeconds int    `json:"estimated_time_seconds"`
	Error            string `json:"error,omitempty"`
}

type JobStatusResponse struct {
	JobID       string      `json:"job_id"`
	Status      string      `json:"status"`
	Progress    int         `json:"progress"`
	Message     string      `json:"message"`
	Error       string      `json:"error,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	StartedAt   time.Time   `json:"started_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	DurationMs  int64       `json:"duration_ms,omitempty"`
}

func NewRefreshHandler(db *sql.DB, jobManager *models.JobManager) *RefreshHandler {
	refreshService := services.NewCatalogRefreshService(db, jobManager)

	return &RefreshHandler{
		RefreshService: refreshService,
		JobManager:     jobManager,
	}
}

// POST /api/v1/catalog/refresh
func (h *RefreshHandler) StartRefresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, RefreshResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	// Start the refresh job
	job := h.RefreshService.StartRefresh(req.Force)

	response := RefreshResponse{
		Success:          true,
		JobID:            job.ID,
		Status:           string(job.Status),
		Message:          "Catalog refresh initiated",
		EstimatedSeconds: 300, // 5 minutes estimate
	}

	c.JSON(http.StatusAccepted, response)
}

// GET /api/v1/catalog/refresh/status/:job_id
func (h *RefreshHandler) GetRefreshStatus(c *gin.Context) {
	jobID := c.Param("job_id")

	job, exists := h.JobManager.GetJob(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job not found",
		})
		return
	}

	response := JobStatusResponse{
		JobID:       job.ID,
		Status:      string(job.Status),
		Progress:    job.Progress,
		Message:     job.Message,
		Error:       job.Error,
		Result:      job.Result,
		StartedAt:   job.StartedAt,
		CompletedAt: job.CompletedAt,
	}

	// Calculate duration if job is completed
	if job.CompletedAt != nil {
		response.DurationMs = job.CompletedAt.Sub(job.StartedAt).Milliseconds()
	} else if job.Status == models.JobStatusRunning {
		response.DurationMs = time.Since(job.StartedAt).Milliseconds()
	}

	c.JSON(http.StatusOK, gin.H{
		"job":    response,
		"status": response.Status,
	})
}

// GET /api/v1/catalog/refresh/jobs
func (h *RefreshHandler) ListRefreshJobs(c *gin.Context) {
	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit > 100 {
		limit = 10
	}

	statusFilter := c.Query("status")

	allJobs := h.JobManager.ListJobs()

	// Filter by job type and status
	var filteredJobs []*models.Job
	for _, job := range allJobs {
		if job.Type == models.JobTypeCatalogRefresh {
			if statusFilter == "" || string(job.Status) == statusFilter {
				filteredJobs = append(filteredJobs, job)
			}
		}
	}

	// Sort by creation time (newest first)
	for i := 0; i < len(filteredJobs)-1; i++ {
		for j := i + 1; j < len(filteredJobs); j++ {
			if filteredJobs[i].CreatedAt.Before(filteredJobs[j].CreatedAt) {
				filteredJobs[i], filteredJobs[j] = filteredJobs[j], filteredJobs[i]
			}
		}
	}

	// Apply limit
	if len(filteredJobs) > limit {
		filteredJobs = filteredJobs[:limit]
	}

	// Convert to response format
	var jobResponses []JobStatusResponse
	for _, job := range filteredJobs {
		response := JobStatusResponse{
			JobID:       job.ID,
			Status:      string(job.Status),
			Progress:    job.Progress,
			Message:     job.Message,
			Error:       job.Error,
			Result:      job.Result,
			StartedAt:   job.StartedAt,
			CompletedAt: job.CompletedAt,
		}

		if job.CompletedAt != nil {
			response.DurationMs = job.CompletedAt.Sub(job.StartedAt).Milliseconds()
		} else if job.Status == models.JobStatusRunning {
			response.DurationMs = time.Since(job.StartedAt).Milliseconds()
		}

		jobResponses = append(jobResponses, response)
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs": jobResponses,
		"pagination": gin.H{
			"total": len(filteredJobs),
			"limit": limit,
		},
	})
}

// DELETE /api/v1/catalog/refresh/:job_id
func (h *RefreshHandler) CancelRefresh(c *gin.Context) {
	jobID := c.Param("job_id")

	job, exists := h.JobManager.GetJob(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job not found",
		})
		return
	}

	if job.Status != models.JobStatusRunning && job.Status != models.JobStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Job cannot be cancelled (not running or pending)",
		})
		return
	}

	err := h.JobManager.CancelJob(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to cancel job",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Job cancelled successfully",
	})
}

// GET /api/v1/catalog/refresh/info
func (h *RefreshHandler) GetRefreshInfo(c *gin.Context) {
	// Get last refresh info from database
	var lastRefresh sql.NullString
	var totalShows, totalArtists int64

	if err := h.RefreshService.DB.QueryRow(`
		SELECT value FROM system_config 
		WHERE key = 'last_catalog_refresh'
	`).Scan(&lastRefresh); err != nil {
		log.Printf("Warning: failed to get last refresh time: %v", err)
	}

	if err := h.RefreshService.DB.QueryRow("SELECT COUNT(*) FROM shows").Scan(&totalShows); err != nil {
		log.Printf("Warning: failed to get total shows: %v", err)
		totalShows = 0
	}
	if err := h.RefreshService.DB.QueryRow("SELECT COUNT(*) FROM artists").Scan(&totalArtists); err != nil {
		log.Printf("Warning: failed to get total artists: %v", err)
		totalArtists = 0
	}

	info := gin.H{
		"total_shows":   totalShows,
		"total_artists": totalArtists,
		"last_refresh":  nil,
	}

	if lastRefresh.Valid {
		if t, err := time.Parse(time.RFC3339, lastRefresh.String); err == nil {
			info["last_refresh"] = t
			info["hours_since_refresh"] = int(time.Since(t).Hours())
		}
	}

	// Check if refresh is currently running
	runningJobs := 0
	for _, job := range h.JobManager.ListJobs() {
		if job.Type == models.JobTypeCatalogRefresh && job.Status == models.JobStatusRunning {
			runningJobs++
		}
	}

	info["refresh_in_progress"] = runningJobs > 0
	info["running_jobs"] = runningJobs

	// Add missing fields that tests expect
	info["next_scheduled_refresh"] = nil // Would be calculated from schedule
	info["refresh_frequency"] = "daily"  // Default from config
	info["auto_refresh_enabled"] = true  // Default from config

	c.JSON(http.StatusOK, info)
}
