package handlers

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmagar/nugs/cron/internal/models"
	"github.com/jmagar/nugs/cron/internal/services"
)

type AdminHandler struct {
	AdminService *services.AdminService
	DB           *sql.DB
}

func NewAdminHandler(db *sql.DB, jobManager *models.JobManager) *AdminHandler {
	adminService := services.NewAdminService(db, jobManager)

	return &AdminHandler{
		AdminService: adminService,
		DB:           db,
	}
}

// User Management
// POST /api/v1/admin/users
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req models.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	response, err := h.AdminService.CreateUser(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	if !response.Success {
		c.JSON(http.StatusBadRequest, response)
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GET /api/v1/admin/users
func (h *AdminHandler) GetUsers(c *gin.Context) {
	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	role := c.Query("role")
	active := c.Query("active")

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if role != "" {
		whereClause += " AND role = ?"
		args = append(args, role)
	}

	if active != "" {
		whereClause += " AND active = ?"
		args = append(args, active == "true")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM users " + whereClause
	var total int64
	err := h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count users"})
		return
	}

	// Get users
	offset := (page - 1) * pageSize
	query := `
		SELECT id, username, email, role, active, last_login, created_at, updated_at
		FROM users ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query users"})
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var lastLogin sql.NullString

		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Role,
			&user.Active, &lastLogin, &user.CreatedAt, &user.UpdatedAt)

		if err != nil {
			continue
		}

		// Don't expose password hash
		user.PasswordHash = ""

		users = append(users, user)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := gin.H{
		"data":        users,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	}

	c.JSON(http.StatusOK, response)
}

// PUT /api/v1/admin/users/:id
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req models.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Get current user from context (simplified)
	updatedBy := "admin" // In real implementation, get from JWT

	err = h.AdminService.UpdateUser(userID, &req, updatedBy)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User updated successfully",
	})
}

// DELETE /api/v1/admin/users/:id
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	deletedBy := "admin" // In real implementation, get from JWT

	err = h.AdminService.DeleteUser(userID, deletedBy)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deleted successfully",
	})
}

// System Configuration
// GET /api/v1/admin/config
func (h *AdminHandler) GetSystemConfig(c *gin.Context) {
	configs, err := h.AdminService.GetSystemConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get system configuration",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  configs,
		"total": len(configs),
	})
}

// PUT /api/v1/admin/config/:key
func (h *AdminHandler) UpdateConfig(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Configuration key is required"})
		return
	}

	var req models.ConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	updatedBy := "admin" // In real implementation, get from JWT

	err := h.AdminService.UpdateConfig(key, &req, updatedBy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration updated successfully",
	})
}

// System Status and Health
// GET /api/v1/admin/status
func (h *AdminHandler) GetSystemStatus(c *gin.Context) {
	status, err := h.AdminService.GetSystemStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get system status",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GET /api/v1/admin/stats
func (h *AdminHandler) GetAdminStats(c *gin.Context) {
	stats, err := h.AdminService.GetAdminStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get admin statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Maintenance
// POST /api/v1/admin/maintenance/cleanup
func (h *AdminHandler) RunCleanup(c *gin.Context) {
	var req models.CleanupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	runBy := "admin" // In real implementation, get from JWT

	job, err := h.AdminService.RunCleanup(&req, runBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start cleanup: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"job_id":  job.ID,
		"message": "Cleanup started",
		"status":  job.Status,
	})
}

// Audit Logs
// GET /api/v1/admin/audit
func (h *AdminHandler) GetAuditLogs(c *gin.Context) {
	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize > 200 {
		pageSize = 200
	}

	// Parse filters
	filters := make(map[string]string)
	if userID := c.Query("user_id"); userID != "" {
		filters["user_id"] = userID
	}
	if action := c.Query("action"); action != "" {
		filters["action"] = action
	}
	if resource := c.Query("resource"); resource != "" {
		filters["resource"] = resource
	}

	logs, total, err := h.AdminService.GetAuditLogs(page, pageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get audit logs",
		})
		return
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := gin.H{
		"data":        logs,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	}

	c.JSON(http.StatusOK, response)
}

// Job Management
// GET /api/v1/admin/jobs
func (h *AdminHandler) GetJobs(c *gin.Context) {
	jobs := h.AdminService.JobManager.ListJobs()

	// Filter by status if provided
	status := c.Query("status")
	if status != "" {
		var filteredJobs []*models.Job
		for _, job := range jobs {
			if string(job.Status) == status {
				filteredJobs = append(filteredJobs, job)
			}
		}
		jobs = filteredJobs
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  jobs,
		"total": len(jobs),
	})
}

// GET /api/v1/admin/jobs/:id
func (h *AdminHandler) GetJob(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	job, exists := h.AdminService.JobManager.GetJob(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// DELETE /api/v1/admin/jobs/:id
func (h *AdminHandler) CancelJob(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	err := h.AdminService.JobManager.CancelJob(jobID)
	if err != nil {
		if err.Error() == "job not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel job"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Job cancelled successfully",
	})
}

// Database Management
// POST /api/v1/admin/database/backup
func (h *AdminHandler) CreateDatabaseBackup(c *gin.Context) {
	// Simplified backup endpoint
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Database backup would be created in production",
		"filename": "backup_" + strconv.FormatInt(time.Now().Unix(), 10) + ".db",
	})
}

// POST /api/v1/admin/database/optimize
func (h *AdminHandler) OptimizeDatabase(c *gin.Context) {
	// Run VACUUM on SQLite database
	_, err := h.DB.Exec("VACUUM")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to optimize database",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database optimized successfully",
	})
}

// GET /api/v1/admin/database/stats
func (h *AdminHandler) GetDatabaseStats(c *gin.Context) {
	stats := make(map[string]interface{})

	// Get table sizes
	tables := []string{"users", "artists", "shows", "downloads", "webhooks", "artist_monitors"}
	tableCounts := make(map[string]int64)

	for _, table := range tables {
		var count int64
		if h.DB.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&count) == nil {
			tableCounts[table] = count
		}
	}

	stats["table_counts"] = tableCounts
	stats["total_tables"] = len(tables)

	// Get database file size
	if stat, err := os.Stat("./data/nugs_api.db"); err == nil {
		stats["database_size_mb"] = float64(stat.Size()) / (1024 * 1024)
	}

	c.JSON(http.StatusOK, stats)
}
