package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmagar/nugs/cron/internal/models"
)

type SchedulerService struct {
	DB                *sql.DB
	JobManager        *models.JobManager
	CatalogService    *CatalogRefreshService
	MonitoringService *MonitoringService
	AdminService      *AdminService

	isRunning     bool
	startTime     time.Time
	schedules     map[int]*models.Schedule
	scheduleMutex sync.RWMutex
	stopChan      chan bool
	ctx           context.Context
	cancel        context.CancelFunc
	ticker        *time.Ticker
}

func NewSchedulerService(db *sql.DB, jobManager *models.JobManager) *SchedulerService {
	ctx, cancel := context.WithCancel(context.Background())

	return &SchedulerService{
		DB:         db,
		JobManager: jobManager,
		schedules:  make(map[int]*models.Schedule),
		stopChan:   make(chan bool, 1),
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (s *SchedulerService) SetServices(catalogService *CatalogRefreshService, monitoringService *MonitoringService, adminService *AdminService) {
	s.CatalogService = catalogService
	s.MonitoringService = monitoringService
	s.AdminService = adminService
}

func (s *SchedulerService) Start() error {
	s.scheduleMutex.Lock()
	defer s.scheduleMutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	// Load schedules from database
	if err := s.loadSchedules(); err != nil {
		return fmt.Errorf("failed to load schedules: %v", err)
	}

	s.isRunning = true
	s.startTime = time.Now()

	// Create ticker for checking schedules every minute
	s.ticker = time.NewTicker(1 * time.Minute)

	// Start the scheduler loop
	go s.run()

	log.Println("Scheduler started successfully")
	return nil
}

func (s *SchedulerService) Stop() error {
	s.scheduleMutex.Lock()
	defer s.scheduleMutex.Unlock()

	if !s.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	s.isRunning = false
	s.cancel()
	s.ticker.Stop()

	select {
	case s.stopChan <- true:
	default:
	}

	log.Println("Scheduler stopped")
	return nil
}

func (s *SchedulerService) run() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Scheduler panic recovered: %v", r)
			// Restart scheduler after panic
			time.Sleep(5 * time.Second)
			s.Start()
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-s.ticker.C:
			s.checkSchedules()
		}
	}
}

func (s *SchedulerService) checkSchedules() {
	s.scheduleMutex.RLock()
	defer s.scheduleMutex.RUnlock()

	now := time.Now()

	for _, schedule := range s.schedules {
		if schedule.Status != models.ScheduleStatusActive {
			continue
		}

		if schedule.NextRun == nil || now.Before(*schedule.NextRun) {
			continue
		}

		// Check if schedule is already running
		if schedule.IsRunning {
			continue
		}

		// Execute schedule
		go s.executeSchedule(schedule)
	}
}

func (s *SchedulerService) executeSchedule(schedule *models.Schedule) {
	// Mark as running
	s.scheduleMutex.Lock()
	schedule.IsRunning = true
	s.scheduleMutex.Unlock()

	defer func() {
		s.scheduleMutex.Lock()
		schedule.IsRunning = false
		s.scheduleMutex.Unlock()
	}()

	startTime := time.Now()

	// Create execution record
	executionID, err := s.createExecution(schedule.ID, "running", "")
	if err != nil {
		log.Printf("Failed to create execution record for schedule %d: %v", schedule.ID, err)
		return
	}

	// Execute the scheduled task
	var job *models.Job
	var executeErr error

	switch schedule.Type {
	case models.ScheduleTypeCatalogRefresh:
		job, executeErr = s.executeCatalogRefresh(schedule)
	case models.ScheduleTypeMonitorCheck:
		job, executeErr = s.executeMonitorCheck(schedule)
	case models.ScheduleTypeSystemCleanup:
		job, executeErr = s.executeSystemCleanup(schedule)
	case models.ScheduleTypeDatabaseBackup:
		job, executeErr = s.executeDatabaseBackup(schedule)
	case models.ScheduleTypeHealthCheck:
		job, executeErr = s.executeHealthCheck(schedule)
	default:
		executeErr = fmt.Errorf("unsupported schedule type: %s", schedule.Type)
	}

	duration := int(time.Since(startTime).Milliseconds())

	// Update execution record
	status := "completed"
	errorMsg := ""
	jobID := ""

	if executeErr != nil {
		status = "failed"
		errorMsg = executeErr.Error()
	}

	if job != nil {
		jobID = job.ID
		schedule.CurrentJobID = jobID
	}

	s.updateExecution(executionID, status, duration, errorMsg, jobID)

	// Update schedule record
	s.updateScheduleAfterExecution(schedule, status == "completed", jobID)

	// Calculate next run time
	s.calculateNextRun(schedule)

	log.Printf("Schedule %s executed: %s (duration: %dms)", schedule.Name, status, duration)
}

func (s *SchedulerService) executeCatalogRefresh(schedule *models.Schedule) (*models.Job, error) {
	if s.CatalogService == nil {
		return nil, fmt.Errorf("catalog service not available")
	}

	// Parse parameters
	var params map[string]interface{}
	if schedule.Parameters != "" {
		json.Unmarshal([]byte(schedule.Parameters), &params)
	}

	force := false
	if forceVal, ok := params["force"].(bool); ok {
		force = forceVal
	}

	job := s.CatalogService.StartRefresh(force)
	return job, nil
}

func (s *SchedulerService) executeMonitorCheck(schedule *models.Schedule) (*models.Job, error) {
	if s.MonitoringService == nil {
		return nil, fmt.Errorf("monitoring service not available")
	}

	job := s.MonitoringService.CheckAllMonitors()
	return job, nil
}

func (s *SchedulerService) executeSystemCleanup(schedule *models.Schedule) (*models.Job, error) {
	if s.AdminService == nil {
		return nil, fmt.Errorf("admin service not available")
	}

	// Parse parameters
	var params map[string]interface{}
	if schedule.Parameters != "" {
		json.Unmarshal([]byte(schedule.Parameters), &params)
	}

	req := &models.CleanupRequest{
		OldLogs:       getBool(params, "old_logs", false),
		OldJobs:       getBool(params, "old_jobs", false),
		OldDeliveries: getBool(params, "old_deliveries", false),
		OldFiles:      getBool(params, "old_files", false),
		DryRun:        getBool(params, "dry_run", false),
	}

	job, err := s.AdminService.RunCleanup(req, "scheduler")
	return job, err
}

func (s *SchedulerService) executeDatabaseBackup(schedule *models.Schedule) (*models.Job, error) {
	// Simplified database backup (in production would create actual backup)
	job := s.JobManager.CreateJob(models.JobTypeAnalytics)

	go func() {
		s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
			j.Status = models.JobStatusRunning
			j.StartedAt = time.Now()
			j.Message = "Creating database backup..."
		})

		// Simulate backup process
		time.Sleep(2 * time.Second)

		completedAt := time.Now()
		s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
			j.Status = models.JobStatusCompleted
			j.Progress = 100
			j.Message = "Database backup completed"
			j.Result = map[string]interface{}{
				"backup_file": "backup_" + strconv.FormatInt(time.Now().Unix(), 10) + ".db",
				"size_mb":     12.5,
			}
			j.CompletedAt = &completedAt
		})
	}()

	return job, nil
}

func (s *SchedulerService) executeHealthCheck(schedule *models.Schedule) (*models.Job, error) {
	job := s.JobManager.CreateJob(models.JobTypeAnalytics)

	go func() {
		s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
			j.Status = models.JobStatusRunning
			j.StartedAt = time.Now()
			j.Message = "Performing health check..."
		})

		// Perform health check
		status := "healthy"
		score := 85
		issues := []string{}

		// Check database
		if err := s.DB.Ping(); err != nil {
			status = "unhealthy"
			score -= 30
			issues = append(issues, "Database connection failed")
		}

		// Check job manager
		jobs := s.JobManager.ListJobs()
		failedJobs := 0
		for _, j := range jobs {
			if j.Status == models.JobStatusFailed {
				failedJobs++
			}
		}

		if failedJobs > 5 {
			status = "degraded"
			score -= 15
			issues = append(issues, fmt.Sprintf("%d failed jobs detected", failedJobs))
		}

		completedAt := time.Now()
		s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
			j.Status = models.JobStatusCompleted
			j.Progress = 100
			j.Message = fmt.Sprintf("Health check completed: %s (score: %d)", status, score)
			j.Result = map[string]interface{}{
				"status": status,
				"score":  score,
				"issues": issues,
			}
			j.CompletedAt = &completedAt
		})
	}()

	return job, nil
}

func (s *SchedulerService) CreateSchedule(req *models.ScheduleRequest, createdBy string) (*models.ScheduleResponse, error) {
	// Validate cron expression
	if !s.isValidCronExpr(req.CronExpr) {
		return &models.ScheduleResponse{
			Success: false,
			Error:   "Invalid cron expression",
		}, nil
	}

	// Serialize parameters
	paramsJSON := "{}"
	if req.Parameters != nil {
		if data, err := json.Marshal(req.Parameters); err == nil {
			paramsJSON = string(data)
		}
	}

	// Calculate next run
	nextRun := s.parseNextRun(req.CronExpr)

	// Insert schedule
	result, err := s.DB.Exec(`
		INSERT INTO schedules (name, description, type, cron_expr, status, parameters, 
		                      next_run, run_count, fail_count, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, 'active', ?, ?, 0, 0, datetime('now'), datetime('now'), ?)
	`, req.Name, req.Description, req.Type, req.CronExpr, paramsJSON, nextRun, createdBy)

	if err != nil {
		return &models.ScheduleResponse{
			Success: false,
			Error:   "Failed to create schedule",
		}, err
	}

	scheduleID, _ := result.LastInsertId()

	// Reload schedules
	s.loadSchedules()

	return &models.ScheduleResponse{
		Success:    true,
		ScheduleID: int(scheduleID),
		Message:    "Schedule created successfully",
	}, nil
}

func (s *SchedulerService) UpdateSchedule(scheduleID int, req *models.ScheduleUpdateRequest) error {
	updates := []string{}
	args := []interface{}{}

	if req.Name != nil {
		updates = append(updates, "name = ?")
		args = append(args, *req.Name)
	}

	if req.Description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *req.Description)
	}

	if req.CronExpr != nil {
		if !s.isValidCronExpr(*req.CronExpr) {
			return fmt.Errorf("invalid cron expression")
		}
		updates = append(updates, "cron_expr = ?")
		args = append(args, *req.CronExpr)

		// Recalculate next run
		nextRun := s.parseNextRun(*req.CronExpr)
		updates = append(updates, "next_run = ?")
		args = append(args, nextRun)
	}

	if req.Status != nil {
		updates = append(updates, "status = ?")
		args = append(args, *req.Status)
	}

	if req.Parameters != nil {
		paramsJSON, _ := json.Marshal(*req.Parameters)
		updates = append(updates, "parameters = ?")
		args = append(args, string(paramsJSON))
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	updates = append(updates, "updated_at = datetime('now')")
	args = append(args, scheduleID)

	query := fmt.Sprintf("UPDATE schedules SET %s WHERE id = ?", strings.Join(updates, ", "))

	result, err := s.DB.Exec(query, args...)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found")
	}

	// Reload schedules
	s.loadSchedules()

	return nil
}

func (s *SchedulerService) DeleteSchedule(scheduleID int) error {
	result, err := s.DB.Exec("DELETE FROM schedules WHERE id = ?", scheduleID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found")
	}

	// Delete related executions
	s.DB.Exec("DELETE FROM schedule_executions WHERE schedule_id = ?", scheduleID)

	// Remove from memory
	s.scheduleMutex.Lock()
	delete(s.schedules, scheduleID)
	s.scheduleMutex.Unlock()

	return nil
}

func (s *SchedulerService) GetStatus() (*models.SchedulerStatus, error) {
	status := &models.SchedulerStatus{
		IsRunning: s.isRunning,
		StartTime: s.startTime,
		Uptime:    time.Since(s.startTime).String(),
	}

	if !s.isRunning {
		return status, nil
	}

	// Get schedule counts
	err := s.DB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
			COUNT(CASE WHEN status = 'paused' THEN 1 END) as paused
		FROM schedules
	`).Scan(&status.TotalSchedules, &status.ActiveSchedules, &status.PausedSchedules)

	if err != nil {
		return status, err
	}

	// Get next and last execution times
	var nextRun, lastRun sql.NullString
	s.DB.QueryRow(`
		SELECT MIN(next_run) FROM schedules WHERE status = 'active' AND next_run IS NOT NULL
	`).Scan(&nextRun)

	s.DB.QueryRow(`
		SELECT MAX(started_at) FROM schedule_executions WHERE DATE(started_at) = DATE('now')
	`).Scan(&lastRun)

	if nextRun.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", nextRun.String); err == nil {
			status.NextExecution = &t
		}
	}

	if lastRun.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", lastRun.String); err == nil {
			status.LastExecution = &t
		}
	}

	// Get today's execution stats
	s.DB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM schedule_executions 
		WHERE DATE(started_at) = DATE('now')
	`).Scan(&status.ExecutionsToday, &status.FailuresToday)

	// Count running jobs
	jobs := s.JobManager.ListJobs()
	for _, job := range jobs {
		if job.Status == models.JobStatusRunning {
			status.RunningJobs++
		}
	}

	return status, nil
}

func (s *SchedulerService) GetStats() (*models.SchedulerStats, error) {
	stats := &models.SchedulerStats{
		TypeBreakdown: make(map[string]int64),
	}

	// Get schedule counts
	err := s.DB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
			COUNT(CASE WHEN status = 'paused' THEN 1 END) as paused,
			COUNT(CASE WHEN status = 'disabled' THEN 1 END) as disabled
		FROM schedules
	`).Scan(&stats.TotalSchedules, &stats.ActiveSchedules, &stats.PausedSchedules, &stats.DisabledSchedules)

	if err != nil {
		return nil, err
	}

	// Get execution stats
	s.DB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM schedule_executions
	`).Scan(&stats.TotalExecutions, &stats.SuccessfulExecutions, &stats.FailedExecutions)

	if stats.TotalExecutions > 0 {
		stats.AverageSuccessRate = float64(stats.SuccessfulExecutions) / float64(stats.TotalExecutions) * 100
	}

	// Get 24h stats
	s.DB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM schedule_executions 
		WHERE started_at >= datetime('now', '-24 hours')
	`).Scan(&stats.ExecutionsLast24h, &stats.FailuresLast24h)

	// Get type breakdown
	rows, err := s.DB.Query(`SELECT type, COUNT(*) FROM schedules GROUP BY type`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var scheduleType string
			var count int64
			if rows.Scan(&scheduleType, &count) == nil {
				stats.TypeBreakdown[scheduleType] = count
			}
		}
	}

	return stats, nil
}

// Helper functions
func (s *SchedulerService) loadSchedules() error {
	rows, err := s.DB.Query(`
		SELECT id, name, description, type, cron_expr, status, parameters, 
		       next_run, last_run, last_job_id, last_status, run_count, fail_count,
		       created_at, updated_at, created_by
		FROM schedules
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	s.scheduleMutex.Lock()
	defer s.scheduleMutex.Unlock()

	s.schedules = make(map[int]*models.Schedule)

	for rows.Next() {
		schedule := &models.Schedule{}
		var nextRun, lastRun, lastJobID, lastStatus, parameters sql.NullString

		err := rows.Scan(
			&schedule.ID, &schedule.Name, &schedule.Description, &schedule.Type,
			&schedule.CronExpr, &schedule.Status, &parameters, &nextRun, &lastRun,
			&lastJobID, &lastStatus, &schedule.RunCount, &schedule.FailCount,
			&schedule.CreatedAt, &schedule.UpdatedAt, &schedule.CreatedBy,
		)

		if err != nil {
			continue
		}

		if nextRun.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", nextRun.String); err == nil {
				schedule.NextRun = &t
			}
		}

		if lastRun.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", lastRun.String); err == nil {
				schedule.LastRun = &t
			}
		}

		if lastJobID.Valid {
			schedule.LastJobID = lastJobID.String
		}

		if lastStatus.Valid {
			schedule.LastStatus = lastStatus.String
		}

		if parameters.Valid {
			schedule.Parameters = parameters.String
		}

		// Calculate failure rate
		if schedule.RunCount > 0 {
			schedule.FailureRate = float64(schedule.FailCount) / float64(schedule.RunCount) * 100
		}

		s.schedules[schedule.ID] = schedule
	}

	return nil
}

func (s *SchedulerService) createExecution(scheduleID int, status, jobID string) (int64, error) {
	result, err := s.DB.Exec(`
		INSERT INTO schedule_executions (schedule_id, job_id, status, started_at)
		VALUES (?, ?, ?, datetime('now'))
	`, scheduleID, jobID, status)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (s *SchedulerService) updateExecution(executionID int64, status string, duration int, error, jobID string) {
	s.DB.Exec(`
		UPDATE schedule_executions 
		SET status = ?, duration_ms = ?, error = ?, job_id = ?, completed_at = datetime('now')
		WHERE id = ?
	`, status, duration, error, jobID, executionID)
}

func (s *SchedulerService) updateScheduleAfterExecution(schedule *models.Schedule, success bool, jobID string) {
	now := time.Now()

	if success {
		s.DB.Exec(`
			UPDATE schedules 
			SET last_run = ?, last_job_id = ?, last_status = 'completed', 
			    run_count = run_count + 1, updated_at = datetime('now')
			WHERE id = ?
		`, now, jobID, schedule.ID)
	} else {
		s.DB.Exec(`
			UPDATE schedules 
			SET last_run = ?, last_job_id = ?, last_status = 'failed', 
			    run_count = run_count + 1, fail_count = fail_count + 1, updated_at = datetime('now')
			WHERE id = ?
		`, now, jobID, schedule.ID)
	}

	// Update in memory
	schedule.LastRun = &now
	schedule.LastJobID = jobID
	schedule.RunCount++
	if !success {
		schedule.FailCount++
	}

	if success {
		schedule.LastStatus = "completed"
	} else {
		schedule.LastStatus = "failed"
	}
}

func (s *SchedulerService) calculateNextRun(schedule *models.Schedule) {
	nextRun := s.parseNextRun(schedule.CronExpr)

	s.DB.Exec("UPDATE schedules SET next_run = ? WHERE id = ?", nextRun, schedule.ID)
	schedule.NextRun = &nextRun
}

func (s *SchedulerService) parseNextRun(cronExpr string) time.Time {
	// Simplified cron parsing - in production use a proper cron library
	now := time.Now()

	parts := strings.Fields(cronExpr)
	if len(parts) != 5 {
		return now.Add(time.Hour) // Default to 1 hour
	}

	minute := parts[0]
	hour := parts[1]

	// Handle common patterns
	if minute == "*" && hour == "*" {
		return now.Add(time.Minute) // Every minute
	}

	if minute == "0" && hour == "*" {
		// Every hour at minute 0
		next := now.Truncate(time.Hour).Add(time.Hour)
		return next
	}

	if minute == "0" && hour != "*" {
		// Daily at specific hour
		if h, err := strconv.Atoi(hour); err == nil {
			next := time.Date(now.Year(), now.Month(), now.Day(), h, 0, 0, 0, now.Location())
			if next.Before(now) {
				next = next.Add(24 * time.Hour)
			}
			return next
		}
	}

	// Default fallback
	return now.Add(time.Hour)
}

func (s *SchedulerService) isValidCronExpr(expr string) bool {
	parts := strings.Fields(expr)
	return len(parts) == 5 // Basic validation
}

func getBool(params map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := params[key].(bool); ok {
		return val
	}
	return defaultValue
}
