package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmagar/nugs/cron/internal/models"
	"github.com/jmagar/nugs/cron/internal/services"
)

type SchedulerHandler struct {
	SchedulerService *services.SchedulerService
	DB               *sql.DB
}

func NewSchedulerHandler(db *sql.DB, jobManager *models.JobManager) *SchedulerHandler {
	schedulerService := services.NewSchedulerService(db, jobManager)

	return &SchedulerHandler{
		SchedulerService: schedulerService,
		DB:               db,
	}
}

func (h *SchedulerHandler) SetServices(catalogService *services.CatalogRefreshService, monitoringService *services.MonitoringService, adminService *services.AdminService) {
	h.SchedulerService.SetServices(catalogService, monitoringService, adminService)
}

// Scheduler Control
// POST /api/v1/scheduler/start
func (h *SchedulerHandler) StartScheduler(c *gin.Context) {
	err := h.SchedulerService.Start()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Scheduler started successfully",
	})
}

// POST /api/v1/scheduler/stop
func (h *SchedulerHandler) StopScheduler(c *gin.Context) {
	err := h.SchedulerService.Stop()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Scheduler stopped successfully",
	})
}

// GET /api/v1/scheduler/status
func (h *SchedulerHandler) GetSchedulerStatus(c *gin.Context) {
	status, err := h.SchedulerService.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get scheduler status",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GET /api/v1/scheduler/stats
func (h *SchedulerHandler) GetSchedulerStats(c *gin.Context) {
	stats, err := h.SchedulerService.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get scheduler statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Schedule Management
// POST /api/v1/scheduler/schedules
func (h *SchedulerHandler) CreateSchedule(c *gin.Context) {
	var req models.ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	createdBy := "admin" // In real implementation, get from JWT

	response, err := h.SchedulerService.CreateSchedule(&req, createdBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule"})
		return
	}

	if !response.Success {
		c.JSON(http.StatusBadRequest, response)
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GET /api/v1/scheduler/schedules
func (h *SchedulerHandler) GetSchedules(c *gin.Context) {
	// Parse pagination and filters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	status := c.Query("status")
	scheduleType := c.Query("type")

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if status != "" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}

	if scheduleType != "" {
		whereClause += " AND type = ?"
		args = append(args, scheduleType)
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM schedules " + whereClause
	var total int64
	err := h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count schedules"})
		return
	}

	// Get schedules
	offset := (page - 1) * pageSize
	query := `
		SELECT s.id, s.name, s.description, s.type, s.cron_expr, s.status, s.parameters,
		       s.next_run, s.last_run, s.last_job_id, s.last_status, s.run_count, s.fail_count,
		       s.created_at, s.updated_at, s.created_by,
		       COUNT(se.id) as execution_count,
		       AVG(CASE WHEN se.duration_ms > 0 THEN se.duration_ms END) as avg_runtime
		FROM schedules s
		LEFT JOIN schedule_executions se ON s.id = se.schedule_id ` + whereClause + `
		GROUP BY s.id
		ORDER BY s.created_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query schedules"})
		return
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var schedule models.Schedule
		var nextRun, lastRun, lastJobID, lastStatus, parameters sql.NullString
		var avgRuntime sql.NullFloat64
		var executionCount int64

		err := rows.Scan(
			&schedule.ID, &schedule.Name, &schedule.Description, &schedule.Type,
			&schedule.CronExpr, &schedule.Status, &parameters, &nextRun, &lastRun,
			&lastJobID, &lastStatus, &schedule.RunCount, &schedule.FailCount,
			&schedule.CreatedAt, &schedule.UpdatedAt, &schedule.CreatedBy,
			&executionCount, &avgRuntime,
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

		// Calculate failure rate and average runtime
		if schedule.RunCount > 0 {
			schedule.FailureRate = float64(schedule.FailCount) / float64(schedule.RunCount) * 100
		}

		if avgRuntime.Valid {
			schedule.AverageRuntime = strconv.FormatFloat(avgRuntime.Float64, 'f', 1, 64) + "ms"
		}

		schedules = append(schedules, schedule)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := gin.H{
		"data":        schedules,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	}

	c.JSON(http.StatusOK, response)
}

// GET /api/v1/scheduler/schedules/:id
func (h *SchedulerHandler) GetSchedule(c *gin.Context) {
	scheduleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	query := `
		SELECT s.id, s.name, s.description, s.type, s.cron_expr, s.status, s.parameters,
		       s.next_run, s.last_run, s.last_job_id, s.last_status, s.run_count, s.fail_count,
		       s.created_at, s.updated_at, s.created_by,
		       COUNT(se.id) as execution_count,
		       COUNT(CASE WHEN se.status = 'completed' THEN 1 END) as success_count,
		       AVG(CASE WHEN se.duration_ms > 0 THEN se.duration_ms END) as avg_runtime
		FROM schedules s
		LEFT JOIN schedule_executions se ON s.id = se.schedule_id
		WHERE s.id = ?
		GROUP BY s.id
	`

	var schedule models.Schedule
	var nextRun, lastRun, lastJobID, lastStatus, parameters sql.NullString
	var avgRuntime sql.NullFloat64
	var executionCount, successCount int64

	err = h.DB.QueryRow(query, scheduleID).Scan(
		&schedule.ID, &schedule.Name, &schedule.Description, &schedule.Type,
		&schedule.CronExpr, &schedule.Status, &parameters, &nextRun, &lastRun,
		&lastJobID, &lastStatus, &schedule.RunCount, &schedule.FailCount,
		&schedule.CreatedAt, &schedule.UpdatedAt, &schedule.CreatedBy,
		&executionCount, &successCount, &avgRuntime,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get schedule"})
		return
	}

	// Parse nullable fields
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

	// Calculate metrics
	if schedule.RunCount > 0 {
		schedule.FailureRate = float64(schedule.FailCount) / float64(schedule.RunCount) * 100
	}

	if avgRuntime.Valid {
		schedule.AverageRuntime = strconv.FormatFloat(avgRuntime.Float64, 'f', 1, 64) + "ms"
	}

	c.JSON(http.StatusOK, schedule)
}

// PUT /api/v1/scheduler/schedules/:id
func (h *SchedulerHandler) UpdateSchedule(c *gin.Context) {
	scheduleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	var req models.ScheduleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	err = h.SchedulerService.UpdateSchedule(scheduleID, &req)
	if err != nil {
		if err.Error() == "schedule not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Schedule updated successfully",
	})
}

// DELETE /api/v1/scheduler/schedules/:id
func (h *SchedulerHandler) DeleteSchedule(c *gin.Context) {
	scheduleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	err = h.SchedulerService.DeleteSchedule(scheduleID)
	if err != nil {
		if err.Error() == "schedule not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete schedule"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Schedule deleted successfully",
	})
}

// GET /api/v1/scheduler/schedules/:id/executions
func (h *SchedulerHandler) GetScheduleExecutions(c *gin.Context) {
	scheduleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	status := c.Query("status")

	// Build WHERE clause
	whereClause := "WHERE se.schedule_id = ?"
	args := []interface{}{scheduleID}

	if status != "" {
		whereClause += " AND se.status = ?"
		args = append(args, status)
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM schedule_executions se " + whereClause
	var total int64
	err = h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count executions"})
		return
	}

	// Get executions
	offset := (page - 1) * pageSize
	query := `
		SELECT se.id, se.schedule_id, se.job_id, se.status, se.started_at, 
		       se.completed_at, se.duration_ms, se.error, se.result,
		       s.name as schedule_name, s.type as schedule_type
		FROM schedule_executions se
		JOIN schedules s ON se.schedule_id = s.id ` + whereClause + `
		ORDER BY se.started_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query executions"})
		return
	}
	defer rows.Close()

	var executions []models.ScheduleExecution
	for rows.Next() {
		var execution models.ScheduleExecution
		var completedAt sql.NullString
		var errorMsg, result sql.NullString

		err := rows.Scan(
			&execution.ID, &execution.ScheduleID, &execution.JobID, &execution.Status,
			&execution.StartedAt, &completedAt, &execution.Duration, &errorMsg, &result,
			&execution.ScheduleName, &execution.ScheduleType,
		)

		if err != nil {
			continue
		}

		if completedAt.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", completedAt.String); err == nil {
				execution.CompletedAt = &t
			}
		}

		if errorMsg.Valid {
			execution.Error = errorMsg.String
		}

		if result.Valid {
			execution.Result = result.String
		}

		executions = append(executions, execution)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := gin.H{
		"data":        executions,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	}

	c.JSON(http.StatusOK, response)
}

// Templates and Helpers
// GET /api/v1/scheduler/templates
func (h *SchedulerHandler) GetScheduleTemplates(c *gin.Context) {
	category := c.Query("category")

	var templates []models.ScheduleTemplate
	for _, template := range models.DefaultScheduleTemplates {
		if category == "" || template.Category == category {
			templates = append(templates, template)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  templates,
		"total": len(templates),
	})
}

// GET /api/v1/scheduler/cron-patterns
func (h *SchedulerHandler) GetCronPatterns(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"patterns": models.CommonCronPatterns,
		"total":    len(models.CommonCronPatterns),
	})
}

// POST /api/v1/scheduler/schedules/bulk
func (h *SchedulerHandler) BulkScheduleOperation(c *gin.Context) {
	var req models.BulkScheduleOperation
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	response := &models.BulkScheduleResponse{
		ProcessedCount: len(req.ScheduleIDs),
		SuccessCount:   0,
		FailedCount:    0,
		Errors:         []string{},
	}

	for _, scheduleID := range req.ScheduleIDs {
		var err error

		switch req.Operation {
		case "enable":
			status := models.ScheduleStatusActive
			updateReq := &models.ScheduleUpdateRequest{Status: &status}
			err = h.SchedulerService.UpdateSchedule(scheduleID, updateReq)
		case "disable":
			status := models.ScheduleStatusDisabled
			updateReq := &models.ScheduleUpdateRequest{Status: &status}
			err = h.SchedulerService.UpdateSchedule(scheduleID, updateReq)
		case "pause":
			status := models.ScheduleStatusPaused
			updateReq := &models.ScheduleUpdateRequest{Status: &status}
			err = h.SchedulerService.UpdateSchedule(scheduleID, updateReq)
		case "resume":
			status := models.ScheduleStatusActive
			updateReq := &models.ScheduleUpdateRequest{Status: &status}
			err = h.SchedulerService.UpdateSchedule(scheduleID, updateReq)
		case "delete":
			err = h.SchedulerService.DeleteSchedule(scheduleID)
		default:
			err = fmt.Errorf("unsupported operation: %s", req.Operation)
		}

		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, fmt.Sprintf("Schedule ID %d: %v", scheduleID, err))
		} else {
			response.SuccessCount++
		}
	}

	response.Success = response.SuccessCount > 0
	response.Message = fmt.Sprintf("Operation '%s' processed: %d successful, %d failed",
		req.Operation, response.SuccessCount, response.FailedCount)

	c.JSON(http.StatusOK, response)
}

// GET /api/v1/scheduler/executions
func (h *SchedulerHandler) GetAllExecutions(c *gin.Context) {
	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	status := c.Query("status")
	scheduleType := c.Query("type")

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if status != "" {
		whereClause += " AND se.status = ?"
		args = append(args, status)
	}

	if scheduleType != "" {
		whereClause += " AND s.type = ?"
		args = append(args, scheduleType)
	}

	// Count total
	countQuery := `
		SELECT COUNT(*) 
		FROM schedule_executions se 
		JOIN schedules s ON se.schedule_id = s.id ` + whereClause

	var total int64
	err := h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count executions"})
		return
	}

	// Get executions
	offset := (page - 1) * pageSize
	query := `
		SELECT se.id, se.schedule_id, se.job_id, se.status, se.started_at, 
		       se.completed_at, se.duration_ms, 
		       s.name as schedule_name, s.type as schedule_type
		FROM schedule_executions se
		JOIN schedules s ON se.schedule_id = s.id ` + whereClause + `
		ORDER BY se.started_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query executions"})
		return
	}
	defer rows.Close()

	var executions []gin.H
	for rows.Next() {
		var id, scheduleID, duration int
		var jobID, status, scheduleName, scheduleType string
		var startedAt time.Time
		var completedAt sql.NullString

		err := rows.Scan(&id, &scheduleID, &jobID, &status, &startedAt,
			&completedAt, &duration, &scheduleName, &scheduleType)

		if err != nil {
			continue
		}

		execution := gin.H{
			"id":            id,
			"schedule_id":   scheduleID,
			"schedule_name": scheduleName,
			"schedule_type": scheduleType,
			"job_id":        jobID,
			"status":        status,
			"started_at":    startedAt,
			"duration_ms":   duration,
		}

		if completedAt.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", completedAt.String); err == nil {
				execution["completed_at"] = t
			}
		}

		executions = append(executions, execution)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := gin.H{
		"data":        executions,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	}

	c.JSON(http.StatusOK, response)
}
