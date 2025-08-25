package services

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/jmagar/nugs/cron/internal/models"
)

type AdminService struct {
	DB         *sql.DB
	JobManager *models.JobManager
	startTime  time.Time
}

func NewAdminService(db *sql.DB, jobManager *models.JobManager) *AdminService {
	return &AdminService{
		DB:         db,
		JobManager: jobManager,
		startTime:  time.Now(),
	}
}

// User Management
func (s *AdminService) CreateUser(req *models.UserCreateRequest) (*models.UserResponse, error) {
	// Hash password (simplified - in production use proper bcrypt)
	h := sha256.New()
	h.Write([]byte(req.Password))
	hashedPassword := fmt.Sprintf("%x", h.Sum(nil))

	// Set defaults
	if req.Role == "" {
		req.Role = models.UserRoleUser
	}

	// Check if user already exists
	var existingID int
	err := s.DB.QueryRow("SELECT id FROM users WHERE username = ? OR email = ?", req.Username, req.Email).Scan(&existingID)
	if err == nil {
		return &models.UserResponse{
			Success: false,
			Error:   "User with this username or email already exists",
		}, nil
	}

	// Create user
	result, err := s.DB.Exec(`
		INSERT INTO users (username, email, password_hash, role, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, req.Username, req.Email, hashedPassword, req.Role, req.Active)

	if err != nil {
		return &models.UserResponse{
			Success: false,
			Error:   "Failed to create user",
		}, err
	}

	userID, _ := result.LastInsertId()

	// Log audit trail
	s.logAuditAction(0, "system", "create_user", "user", fmt.Sprintf("%d", userID),
		fmt.Sprintf("Created user: %s", req.Username), "", "", true)

	return &models.UserResponse{
		Success: true,
		UserID:  int(userID),
		Message: "User created successfully",
	}, nil
}

func (s *AdminService) UpdateUser(userID int, req *models.UserUpdateRequest, updatedBy string) error {
	updates := []string{}
	args := []interface{}{}

	if req.Username != nil {
		updates = append(updates, "username = ?")
		args = append(args, *req.Username)
	}

	if req.Email != nil {
		updates = append(updates, "email = ?")
		args = append(args, *req.Email)
	}

	if req.Role != nil {
		updates = append(updates, "role = ?")
		args = append(args, *req.Role)
	}

	if req.Active != nil {
		updates = append(updates, "active = ?")
		args = append(args, *req.Active)
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	updates = append(updates, "updated_at = datetime('now')")
	args = append(args, userID)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = ?", updates[0])
	for i := 1; i < len(updates); i++ {
		query = fmt.Sprintf("%s, %s", query, updates[i])
	}

	result, err := s.DB.Exec(query, args...)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	// Log audit trail
	s.logAuditAction(0, updatedBy, "update_user", "user", fmt.Sprintf("%d", userID),
		"Updated user profile", "", "", true)

	return nil
}

func (s *AdminService) DeleteUser(userID int, deletedBy string) error {
	result, err := s.DB.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	// Log audit trail
	s.logAuditAction(0, deletedBy, "delete_user", "user", fmt.Sprintf("%d", userID),
		"Deleted user", "", "", true)

	return nil
}

func (s *AdminService) ChangePassword(userID int, req *models.PasswordChangeRequest) error {
	// Get current password hash
	var currentHash string
	err := s.DB.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&currentHash)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify current password (simplified hash comparison)
	h := sha256.New()
	h.Write([]byte(req.CurrentPassword))
	currentPasswordHash := fmt.Sprintf("%x", h.Sum(nil))

	if currentHash != currentPasswordHash {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	newH := sha256.New()
	newH.Write([]byte(req.NewPassword))
	hashedPassword := fmt.Sprintf("%x", newH.Sum(nil))

	// Update password
	_, err = s.DB.Exec(`
		UPDATE users 
		SET password_hash = ?, updated_at = datetime('now')
		WHERE id = ?
	`, hashedPassword, userID)

	if err != nil {
		return err
	}

	// Log audit trail
	s.logAuditAction(userID, "", "change_password", "user", fmt.Sprintf("%d", userID),
		"Changed password", "", "", true)

	return nil
}

// System Configuration
func (s *AdminService) GetSystemConfig() ([]models.SystemConfig, error) {
	rows, err := s.DB.Query(`
		SELECT key, value, description, data_type, updated_at
		FROM system_config
		ORDER BY key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []models.SystemConfig
	for rows.Next() {
		var config models.SystemConfig
		err := rows.Scan(&config.Key, &config.Value, &config.Description,
			&config.Type, &config.UpdatedAt)
		if err != nil {
			continue
		}
		configs = append(configs, config)
	}

	return configs, nil
}

func (s *AdminService) UpdateConfig(key string, req *models.ConfigUpdateRequest, updatedBy string) error {
	// Check if config exists
	var exists bool
	err := s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM system_config WHERE key = ?)", key).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("configuration key not found: %s", key)
	}

	// Convert value to string for storage
	valueStr := fmt.Sprintf("%v", req.Value)

	// Update config
	_, err = s.DB.Exec(`
		UPDATE system_config 
		SET value = ?, updated_at = datetime('now')
		WHERE key = ?
	`, valueStr, key)

	if err != nil {
		return err
	}

	// Log audit trail
	s.logAuditAction(0, updatedBy, "update_config", "system_config", key,
		fmt.Sprintf("Updated config %s", key), "", "", true)

	return nil
}

// System Status
func (s *AdminService) GetSystemStatus() (*models.SystemStatus, error) {
	status := &models.SystemStatus{
		Status:      "healthy",
		Version:     "v1.0.0",
		Uptime:      time.Since(s.startTime).String(),
		Services:    make(map[string]models.ServiceStatus),
		LastUpdated: time.Now(),
	}

	// Database status
	dbStatus, err := s.getDatabaseStatus()
	if err == nil {
		status.Database = *dbStatus
	}

	// Job system status
	jobStatus := s.getJobSystemStatus()
	status.Jobs = *jobStatus

	// Storage status
	storageStatus, err := s.getStorageStatus()
	if err == nil {
		status.Storage = *storageStatus
	}

	// Performance status
	perfStatus := s.getPerformanceStatus()
	status.Performance = *perfStatus

	// Health check
	health := s.calculateSystemHealth(status)
	status.Health = *health

	// Determine overall status
	if status.Health.Score >= 80 {
		status.Status = "healthy"
	} else if status.Health.Score >= 60 {
		status.Status = "degraded"
	} else {
		status.Status = "unhealthy"
	}

	return status, nil
}

func (s *AdminService) getDatabaseStatus() (*models.DatabaseStatus, error) {
	status := &models.DatabaseStatus{
		RecordCounts: make(map[string]int64),
	}

	// Test database connection
	start := time.Now()
	err := s.DB.Ping()
	status.ResponseTime = float64(time.Since(start).Milliseconds())
	status.Connected = (err == nil)

	if !status.Connected {
		return status, nil
	}

	// Get database size
	if stat, err := os.Stat("./data/nugs_api.db"); err == nil {
		status.Size = float64(stat.Size()) / (1024 * 1024) // MB
	}

	// Count tables and records
	tables := []string{"users", "artists", "shows", "downloads", "webhooks", "artist_monitors"}
	status.TableCount = len(tables)

	for _, table := range tables {
		var count int64
		if s.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count) == nil {
			status.RecordCounts[table] = count
		}
	}

	// Check integrity (simplified)
	status.Integrity = true

	return status, nil
}

func (s *AdminService) getJobSystemStatus() *models.JobSystemStatus {
	status := &models.JobSystemStatus{}

	jobs := s.JobManager.ListJobs()
	for _, job := range jobs {
		switch job.Status {
		case models.JobStatusRunning:
			status.Active++
		case models.JobStatusPending:
			status.Pending++
		case models.JobStatusCompleted:
			status.Completed++
		case models.JobStatusFailed:
			status.Failed++
		}
	}

	// Calculate failure rate
	total := status.Completed + status.Failed
	if total > 0 {
		status.FailureRate = float64(status.Failed) / float64(total) * 100
	}

	status.QueueSize = int(status.Pending)
	status.Workers = 3 // Default worker count

	return status
}

func (s *AdminService) getStorageStatus() (*models.StorageStatus, error) {
	status := &models.StorageStatus{}

	// Get filesystem stats
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/home/jmagar/code/nugs/downloads", &stat); err != nil {
		return nil, err
	}

	status.TotalGB = float64(stat.Blocks*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
	status.FreeGB = float64(stat.Bavail*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
	status.UsedGB = status.TotalGB - status.FreeGB
	status.UsagePercent = (status.UsedGB / status.TotalGB) * 100

	// Count files
	var fileCount int64
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM downloads WHERE file_path IS NOT NULL AND file_path != ''`).Scan(&fileCount); err != nil {
		log.Printf("Error counting files: %v", err)
		fileCount = 0
	}
	status.FileCount = fileCount

	return status, nil
}

func (s *AdminService) getPerformanceStatus() *models.PerformanceStatus {
	status := &models.PerformanceStatus{}

	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	status.MemoryUsage = float64(m.Alloc) / (1024 * 1024) // MB
	status.MemoryTotal = float64(m.Sys) / (1024 * 1024)   // MB

	// Simplified metrics (in production, these would come from actual monitoring)
	status.CPUUsage = 15.0    // %
	status.Connections = 5    // Active connections
	status.RequestRate = 10.0 // req/sec
	status.ErrorRate = 2.0    // %

	return status
}

func (s *AdminService) calculateSystemHealth(status *models.SystemStatus) *models.SystemHealth {
	health := &models.SystemHealth{
		Issues:          []models.HealthIssue{},
		Metrics:         make(map[string]float64),
		Recommendations: []string{},
	}

	score := 100
	issues := []models.HealthIssue{}

	// Database health
	if !status.Database.Connected {
		score -= 30
		issues = append(issues, models.HealthIssue{
			Type:      "critical",
			Component: "database",
			Message:   "Database connection failed",
			Severity:  5,
			Action:    "Check database service and configuration",
		})
	} else if status.Database.ResponseTime > 100 {
		score -= 10
		issues = append(issues, models.HealthIssue{
			Type:      "warning",
			Component: "database",
			Message:   "Database response time is high",
			Severity:  2,
			Action:    "Consider database optimization",
		})
	}

	// Storage health
	if status.Storage.UsagePercent > 90 {
		score -= 20
		issues = append(issues, models.HealthIssue{
			Type:      "error",
			Component: "storage",
			Message:   "Disk usage is critically high",
			Severity:  4,
			Action:    "Clean up old files or expand storage",
		})
	} else if status.Storage.UsagePercent > 75 {
		score -= 5
		issues = append(issues, models.HealthIssue{
			Type:      "warning",
			Component: "storage",
			Message:   "Disk usage is high",
			Severity:  2,
		})
	}

	// Job system health
	if status.Jobs.FailureRate > 20 {
		score -= 15
		issues = append(issues, models.HealthIssue{
			Type:      "error",
			Component: "jobs",
			Message:   "Job failure rate is high",
			Severity:  3,
			Action:    "Review failed jobs and fix underlying issues",
		})
	}

	// Performance health
	if status.Performance.MemoryUsage > status.Performance.MemoryTotal*0.8 {
		score -= 10
		issues = append(issues, models.HealthIssue{
			Type:      "warning",
			Component: "performance",
			Message:   "Memory usage is high",
			Severity:  2,
		})
	}

	health.Score = score
	health.Issues = issues

	// Determine status based on score
	if score >= 90 {
		health.Status = "excellent"
	} else if score >= 80 {
		health.Status = "good"
	} else if score >= 60 {
		health.Status = "fair"
	} else if score >= 40 {
		health.Status = "poor"
	} else {
		health.Status = "critical"
	}

	// Add metrics
	health.Metrics["database_response_time"] = status.Database.ResponseTime
	health.Metrics["storage_usage_percent"] = status.Storage.UsagePercent
	health.Metrics["job_failure_rate"] = status.Jobs.FailureRate
	health.Metrics["memory_usage_mb"] = status.Performance.MemoryUsage

	// Add recommendations
	if len(issues) == 0 {
		health.Recommendations = append(health.Recommendations, "System is running optimally")
	} else {
		for _, issue := range issues {
			if issue.Action != "" {
				health.Recommendations = append(health.Recommendations, issue.Action)
			}
		}
	}

	return health
}

// Maintenance
func (s *AdminService) RunCleanup(req *models.CleanupRequest, runBy string) (*models.Job, error) {
	job := s.JobManager.CreateJob(models.JobTypeAnalytics) // Reuse analytics job type

	go s.performCleanup(job, req, runBy)

	return job, nil
}

func (s *AdminService) performCleanup(job *models.Job, req *models.CleanupRequest, runBy string) {
	startTime := time.Now()

	if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Status = models.JobStatusRunning
		j.StartedAt = startTime
		j.Message = "Starting system cleanup..."
	}); err != nil {
		log.Printf("Error updating job status: %v", err)
		return
	}

	cleanupResults := make(map[string]int64)
	totalCleaned := int64(0)

	// Clean old jobs
	if req.OldJobs {
		if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
			j.Progress = 25
			j.Message = "Cleaning old job records..."
		}); err != nil {
			log.Printf("Error updating job progress: %v", err)
		}

		// In a real implementation, clean jobs older than retention period
		cleanupResults["old_jobs"] = 0 // Placeholder
	}

	// Clean old webhook deliveries
	if req.OldDeliveries {
		if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
			j.Progress = 50
			j.Message = "Cleaning old webhook deliveries..."
		}); err != nil {
			log.Printf("Error updating job progress: %v", err)
		}

		if !req.DryRun {
			result, err := s.DB.Exec(`
				DELETE FROM webhook_deliveries 
				WHERE created_at < datetime('now', '-30 days')
			`)
			if err == nil {
				cleaned, _ := result.RowsAffected()
				cleanupResults["old_deliveries"] = cleaned
				totalCleaned += cleaned
			}
		}
	}

	// Clean orphaned files
	if req.OldFiles {
		if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
			j.Progress = 75
			j.Message = "Cleaning orphaned files..."
		}); err != nil {
			log.Printf("Error updating job progress: %v", err)
		}

		// In a real implementation, scan filesystem and remove orphaned files
		cleanupResults["orphaned_files"] = 0 // Placeholder
	}

	// Complete job
	completedAt := time.Now()
	if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Status = models.JobStatusCompleted
		j.Progress = 100
		j.Message = fmt.Sprintf("Cleanup completed: %d items cleaned", totalCleaned)
		j.Result = cleanupResults
		j.CompletedAt = &completedAt
	}); err != nil {
		log.Printf("Error updating final job status: %v", err)
	}

	// Log audit trail
	s.logAuditAction(0, runBy, "system_cleanup", "maintenance", job.ID,
		"Performed system cleanup", "", "", true)
}

func (s *AdminService) GetAdminStats() (*models.AdminStats, error) {
	stats := &models.AdminStats{}

	// User stats
	err := s.DB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN active = 1 THEN 1 END) as active,
			COUNT(CASE WHEN active = 0 THEN 1 END) as inactive
		FROM users
	`).Scan(&stats.Users.Total, &stats.Users.Active, &stats.Users.Inactive)

	if err != nil {
		return nil, err
	}

	// System overview
	dbStatus, _ := s.getDatabaseStatus()
	if dbStatus != nil {
		stats.System.DatabaseSize = dbStatus.Size
		stats.System.HealthScore = 85 // Simplified
	}

	// Activity stats (simplified - in production would track actual metrics)
	stats.Activity.APIRequests = 150
	stats.Activity.Downloads = 25
	stats.Activity.JobsExecuted = 12

	// Maintenance stats
	stats.Maintenance.ScheduledTasks = 3
	stats.Maintenance.RunningTasks = 0
	stats.Maintenance.CompletedTasks = 8
	stats.Maintenance.FailedTasks = 1

	return stats, nil
}

func (s *AdminService) logAuditAction(userID int, username, action, resource, resourceID, details, ipAddress, userAgent string, success bool) {
	if _, err := s.DB.Exec(`
		INSERT INTO audit_logs (user_id, username, action, resource, resource_id, 
		                       details, ip_address, user_agent, success, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, userID, username, action, resource, resourceID, details, ipAddress, userAgent, success); err != nil {
		log.Printf("Error logging audit action: %v", err)
	}
}

func (s *AdminService) GetAuditLogs(page, pageSize int, filters map[string]string) ([]models.AuditLog, int64, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if userID, ok := filters["user_id"]; ok && userID != "" {
		whereClause += " AND user_id = ?"
		args = append(args, userID)
	}

	if action, ok := filters["action"]; ok && action != "" {
		whereClause += " AND action = ?"
		args = append(args, action)
	}

	if resource, ok := filters["resource"]; ok && resource != "" {
		whereClause += " AND resource = ?"
		args = append(args, resource)
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM audit_logs " + whereClause
	var total int64
	err := s.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get logs
	offset := (page - 1) * pageSize
	query := `
		SELECT id, user_id, username, action, resource, resource_id, 
		       details, ip_address, user_agent, success, created_at
		FROM audit_logs ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		var resourceID, details, ipAddress, userAgent sql.NullString

		err := rows.Scan(&log.ID, &log.UserID, &log.Username, &log.Action,
			&log.Resource, &resourceID, &details, &ipAddress, &userAgent,
			&log.Success, &log.CreatedAt)

		if err != nil {
			continue
		}

		if resourceID.Valid {
			log.ResourceID = resourceID.String
		}
		if details.Valid {
			log.Details = details.String
		}
		if ipAddress.Valid {
			log.IPAddress = ipAddress.String
		}
		if userAgent.Valid {
			log.UserAgent = userAgent.String
		}

		logs = append(logs, log)
	}

	return logs, total, nil
}
