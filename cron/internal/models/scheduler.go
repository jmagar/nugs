package models

import (
	"time"
)

type ScheduleStatus string

const (
	ScheduleStatusActive   ScheduleStatus = "active"
	ScheduleStatusPaused   ScheduleStatus = "paused"
	ScheduleStatusDisabled ScheduleStatus = "disabled"
	ScheduleStatusError    ScheduleStatus = "error"
)

type ScheduleType string

const (
	ScheduleTypeCatalogRefresh ScheduleType = "catalog_refresh"
	ScheduleTypeMonitorCheck   ScheduleType = "monitor_check"
	ScheduleTypeSystemCleanup  ScheduleType = "system_cleanup"
	ScheduleTypeDatabaseBackup ScheduleType = "database_backup"
	ScheduleTypeHealthCheck    ScheduleType = "health_check"
	ScheduleTypeCustom         ScheduleType = "custom"
)

type Schedule struct {
	ID          int            `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Description string         `json:"description" db:"description"`
	Type        ScheduleType   `json:"type" db:"type"`
	CronExpr    string         `json:"cron_expr" db:"cron_expr"`
	Status      ScheduleStatus `json:"status" db:"status"`
	Parameters  string         `json:"parameters,omitempty" db:"parameters"` // JSON string
	NextRun     *time.Time     `json:"next_run,omitempty" db:"next_run"`
	LastRun     *time.Time     `json:"last_run,omitempty" db:"last_run"`
	LastJobID   string         `json:"last_job_id,omitempty" db:"last_job_id"`
	LastStatus  string         `json:"last_status,omitempty" db:"last_status"`
	RunCount    int64          `json:"run_count" db:"run_count"`
	FailCount   int64          `json:"fail_count" db:"fail_count"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
	CreatedBy   string         `json:"created_by" db:"created_by"`

	// Runtime fields (not stored in DB)
	IsRunning      bool    `json:"is_running"`
	CurrentJobID   string  `json:"current_job_id,omitempty"`
	FailureRate    float64 `json:"failure_rate"`
	AverageRuntime string  `json:"average_runtime,omitempty"`
}

type ScheduleExecution struct {
	ID          int        `json:"id" db:"id"`
	ScheduleID  int        `json:"schedule_id" db:"schedule_id"`
	JobID       string     `json:"job_id" db:"job_id"`
	Status      string     `json:"status" db:"status"` // pending, running, completed, failed
	StartedAt   time.Time  `json:"started_at" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	Duration    int        `json:"duration_ms" db:"duration_ms"`
	Error       string     `json:"error,omitempty" db:"error"`
	Result      string     `json:"result,omitempty" db:"result"` // JSON string

	// Related data (populated via JOIN)
	ScheduleName string `json:"schedule_name,omitempty"`
	ScheduleType string `json:"schedule_type,omitempty"`
}

type SchedulerStatus struct {
	IsRunning       bool       `json:"is_running"`
	StartTime       time.Time  `json:"start_time"`
	ActiveSchedules int        `json:"active_schedules"`
	PausedSchedules int        `json:"paused_schedules"`
	TotalSchedules  int        `json:"total_schedules"`
	RunningJobs     int        `json:"running_jobs"`
	NextExecution   *time.Time `json:"next_execution,omitempty"`
	LastExecution   *time.Time `json:"last_execution,omitempty"`
	ExecutionsToday int64      `json:"executions_today"`
	FailuresToday   int64      `json:"failures_today"`
	Uptime          string     `json:"uptime"`
}

type SchedulerStats struct {
	TotalSchedules       int64               `json:"total_schedules"`
	ActiveSchedules      int64               `json:"active_schedules"`
	PausedSchedules      int64               `json:"paused_schedules"`
	DisabledSchedules    int64               `json:"disabled_schedules"`
	TotalExecutions      int64               `json:"total_executions"`
	SuccessfulExecutions int64               `json:"successful_executions"`
	FailedExecutions     int64               `json:"failed_executions"`
	AverageSuccessRate   float64             `json:"average_success_rate"`
	ExecutionsLast24h    int64               `json:"executions_last_24h"`
	FailuresLast24h      int64               `json:"failures_last_24h"`
	TypeBreakdown        map[string]int64    `json:"type_breakdown"`
	PopularSchedules     []ScheduleStats     `json:"popular_schedules"`
	RecentActivity       []ScheduleExecution `json:"recent_activity"`
}

type ScheduleStats struct {
	ScheduleID     int        `json:"schedule_id"`
	ScheduleName   string     `json:"schedule_name"`
	Type           string     `json:"type"`
	ExecutionCount int64      `json:"execution_count"`
	SuccessCount   int64      `json:"success_count"`
	FailureCount   int64      `json:"failure_count"`
	SuccessRate    float64    `json:"success_rate"`
	LastRun        *time.Time `json:"last_run,omitempty"`
	NextRun        *time.Time `json:"next_run,omitempty"`
	AverageRuntime float64    `json:"average_runtime_ms"`
}

// Request/Response models
type ScheduleRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Type        ScheduleType           `json:"type" binding:"required"`
	CronExpr    string                 `json:"cron_expr" binding:"required"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type ScheduleUpdateRequest struct {
	Name        *string                 `json:"name,omitempty"`
	Description *string                 `json:"description,omitempty"`
	CronExpr    *string                 `json:"cron_expr,omitempty"`
	Status      *ScheduleStatus         `json:"status,omitempty"`
	Parameters  *map[string]interface{} `json:"parameters,omitempty"`
}

type ScheduleResponse struct {
	Success    bool   `json:"success"`
	ScheduleID int    `json:"schedule_id,omitempty"`
	Message    string `json:"message"`
	Error      string `json:"error,omitempty"`
}

type ScheduleTestRequest struct {
	ScheduleID int                    `json:"schedule_id" binding:"required"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type ScheduleTestResponse struct {
	Success  bool   `json:"success"`
	JobID    string `json:"job_id,omitempty"`
	Message  string `json:"message"`
	Error    string `json:"error,omitempty"`
	Duration int    `json:"duration_ms"`
}

type BulkScheduleOperation struct {
	ScheduleIDs []int                  `json:"schedule_ids" binding:"required"`
	Operation   string                 `json:"operation" binding:"required"` // enable, disable, pause, resume, delete
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type BulkScheduleResponse struct {
	Success        bool     `json:"success"`
	ProcessedCount int      `json:"processed_count"`
	SuccessCount   int      `json:"success_count"`
	FailedCount    int      `json:"failed_count"`
	Errors         []string `json:"errors,omitempty"`
	Message        string   `json:"message"`
}

// Predefined schedule templates
type ScheduleTemplate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        ScheduleType           `json:"type"`
	CronExpr    string                 `json:"cron_expr"`
	Parameters  map[string]interface{} `json:"parameters"`
	Category    string                 `json:"category"`
}

var DefaultScheduleTemplates = []ScheduleTemplate{
	{
		Name:        "Daily Catalog Refresh",
		Description: "Refresh the catalog daily at 3 AM",
		Type:        ScheduleTypeCatalogRefresh,
		CronExpr:    "0 3 * * *",
		Parameters:  map[string]interface{}{"force": false},
		Category:    "Data Management",
	},
	{
		Name:        "Hourly Monitor Check",
		Description: "Check all active monitors every hour",
		Type:        ScheduleTypeMonitorCheck,
		CronExpr:    "0 * * * *",
		Parameters:  map[string]interface{}{},
		Category:    "Monitoring",
	},
	{
		Name:        "Weekly System Cleanup",
		Description: "Clean up old data every Sunday at 2 AM",
		Type:        ScheduleTypeSystemCleanup,
		CronExpr:    "0 2 * * 0",
		Parameters: map[string]interface{}{
			"old_logs":       true,
			"old_jobs":       true,
			"old_deliveries": true,
			"dry_run":        false,
		},
		Category: "Maintenance",
	},
	{
		Name:        "Daily Database Backup",
		Description: "Create database backup daily at 1 AM",
		Type:        ScheduleTypeDatabaseBackup,
		CronExpr:    "0 1 * * *",
		Parameters: map[string]interface{}{
			"include_database": true,
			"include_files":    false,
			"compress":         true,
		},
		Category: "Backup",
	},
	{
		Name:        "15-minute Health Check",
		Description: "Check system health every 15 minutes",
		Type:        ScheduleTypeHealthCheck,
		CronExpr:    "*/15 * * * *",
		Parameters:  map[string]interface{}{},
		Category:    "Monitoring",
	},
}

// Cron expression helpers
type CronPattern struct {
	Expression  string `json:"expression"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

var CommonCronPatterns = []CronPattern{
	{"* * * * *", "Every minute", "Runs every minute"},
	{"*/5 * * * *", "Every 5 minutes", "Runs at :00, :05, :10, etc."},
	{"*/15 * * * *", "Every 15 minutes", "Runs at :00, :15, :30, :45"},
	{"*/30 * * * *", "Every 30 minutes", "Runs at :00 and :30"},
	{"0 * * * *", "Every hour", "Runs at the start of every hour"},
	{"0 */2 * * *", "Every 2 hours", "Runs at 00:00, 02:00, 04:00, etc."},
	{"0 */6 * * *", "Every 6 hours", "Runs at 00:00, 06:00, 12:00, 18:00"},
	{"0 0 * * *", "Daily at midnight", "Runs once per day at 00:00"},
	{"0 2 * * *", "Daily at 2 AM", "Runs once per day at 02:00"},
	{"0 0 */2 * *", "Every 2 days", "Runs every other day at midnight"},
	{"0 0 * * 0", "Weekly on Sunday", "Runs once per week on Sunday at midnight"},
	{"0 0 1 * *", "Monthly", "Runs once per month on the 1st at midnight"},
	{"0 0 1 */3 *", "Quarterly", "Runs every 3 months on the 1st at midnight"},
	{"0 0 1 1 *", "Yearly", "Runs once per year on Jan 1st at midnight"},
}
