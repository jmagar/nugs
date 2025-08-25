package models

import (
	"time"
)

type UserRole string

const (
	UserRoleAdmin     UserRole = "admin"
	UserRoleModerator UserRole = "moderator"
	UserRoleUser      UserRole = "user"
	UserRoleReadonly  UserRole = "readonly"
)

type User struct {
	ID           int        `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"` // Never expose password
	Role         UserRole   `json:"role" db:"role"`
	Active       bool       `json:"active" db:"active"`
	LastLogin    *time.Time `json:"last_login,omitempty" db:"last_login"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`

	// Additional fields for admin view
	LoginCount int64  `json:"login_count,omitempty"`
	LastIP     string `json:"last_ip,omitempty"`
}

type SystemConfig struct {
	Key         string    `json:"key" db:"key"`
	Value       string    `json:"value" db:"value"`
	Description string    `json:"description" db:"description"`
	Type        string    `json:"type" db:"type"` // string, number, boolean, json
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	UpdatedBy   string    `json:"updated_by,omitempty" db:"updated_by"`
}

type SystemMaintenance struct {
	ID          int        `json:"id" db:"id"`
	Type        string     `json:"type" db:"type"`         // cleanup, backup, reindex, optimize
	Status      string     `json:"status" db:"status"`     // scheduled, running, completed, failed
	Progress    int        `json:"progress" db:"progress"` // 0-100
	Message     string     `json:"message" db:"message"`
	Result      string     `json:"result,omitempty" db:"result"`
	Error       string     `json:"error,omitempty" db:"error"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty" db:"scheduled_at"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedBy   string     `json:"created_by" db:"created_by"`
}

type AuditLog struct {
	ID         int       `json:"id" db:"id"`
	UserID     int       `json:"user_id" db:"user_id"`
	Username   string    `json:"username" db:"username"`
	Action     string    `json:"action" db:"action"`
	Resource   string    `json:"resource" db:"resource"`
	ResourceID string    `json:"resource_id,omitempty" db:"resource_id"`
	Details    string    `json:"details,omitempty" db:"details"`
	IPAddress  string    `json:"ip_address" db:"ip_address"`
	UserAgent  string    `json:"user_agent,omitempty" db:"user_agent"`
	Success    bool      `json:"success" db:"success"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type SystemStatus struct {
	Status      string                   `json:"status"` // healthy, degraded, down
	Version     string                   `json:"version"`
	Uptime      string                   `json:"uptime"`
	Database    DatabaseStatus           `json:"database"`
	Jobs        JobSystemStatus          `json:"jobs"`
	Storage     StorageStatus            `json:"storage"`
	Services    map[string]ServiceStatus `json:"services"`
	Performance PerformanceStatus        `json:"performance"`
	Health      SystemHealth             `json:"health"`
	LastUpdated time.Time                `json:"last_updated"`
}

type DatabaseStatus struct {
	Connected    bool             `json:"connected"`
	Size         float64          `json:"size_mb"`
	TableCount   int              `json:"table_count"`
	RecordCounts map[string]int64 `json:"record_counts"`
	Integrity    bool             `json:"integrity_ok"`
	LastBackup   *time.Time       `json:"last_backup,omitempty"`
	ResponseTime float64          `json:"response_time_ms"`
}

type JobSystemStatus struct {
	Active      int64   `json:"active_jobs"`
	Pending     int64   `json:"pending_jobs"`
	Completed   int64   `json:"completed_jobs"`
	Failed      int64   `json:"failed_jobs"`
	QueueSize   int     `json:"queue_size"`
	Workers     int     `json:"worker_count"`
	FailureRate float64 `json:"failure_rate"`
}

type StorageStatus struct {
	TotalGB      float64    `json:"total_gb"`
	UsedGB       float64    `json:"used_gb"`
	FreeGB       float64    `json:"free_gb"`
	UsagePercent float64    `json:"usage_percent"`
	FileCount    int64      `json:"file_count"`
	OldestFile   *time.Time `json:"oldest_file,omitempty"`
}

type ServiceStatus struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"` // running, stopped, error
	Healthy      bool      `json:"healthy"`
	LastCheck    time.Time `json:"last_check"`
	ResponseTime float64   `json:"response_time_ms"`
	ErrorCount   int       `json:"error_count"`
	Details      string    `json:"details,omitempty"`
}

type PerformanceStatus struct {
	CPUUsage    float64 `json:"cpu_usage_percent"`
	MemoryUsage float64 `json:"memory_usage_mb"`
	MemoryTotal float64 `json:"memory_total_mb"`
	Connections int     `json:"active_connections"`
	RequestRate float64 `json:"requests_per_second"`
	ErrorRate   float64 `json:"error_rate_percent"`
}

type SystemHealth struct {
	Score           int                `json:"score"`  // 0-100
	Status          string             `json:"status"` // excellent, good, fair, poor, critical
	Issues          []HealthIssue      `json:"issues"`
	Metrics         map[string]float64 `json:"metrics"`
	Recommendations []string           `json:"recommendations"`
}

type HealthIssue struct {
	Type      string `json:"type"` // warning, error, critical
	Component string `json:"component"`
	Message   string `json:"message"`
	Severity  int    `json:"severity"` // 1-5
	Action    string `json:"action,omitempty"`
}

// Request/Response models
type UserCreateRequest struct {
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=8"`
	Role     UserRole `json:"role"`
	Active   bool     `json:"active"`
}

type UserUpdateRequest struct {
	Username *string   `json:"username,omitempty"`
	Email    *string   `json:"email,omitempty"`
	Role     *UserRole `json:"role,omitempty"`
	Active   *bool     `json:"active,omitempty"`
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

type ConfigUpdateRequest struct {
	Value interface{} `json:"value" binding:"required"`
}

type MaintenanceRequest struct {
	Type        string                 `json:"type" binding:"required"` // cleanup, backup, reindex
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type CleanupRequest struct {
	OldLogs       bool `json:"old_logs"`       // Clean logs older than retention period
	OldJobs       bool `json:"old_jobs"`       // Clean completed jobs older than retention
	OldDeliveries bool `json:"old_deliveries"` // Clean webhook deliveries
	OldFiles      bool `json:"old_files"`      // Clean orphaned download files
	DryRun        bool `json:"dry_run"`        // Preview what would be cleaned
}

type BackupRequest struct {
	IncludeDatabase bool   `json:"include_database"`
	IncludeFiles    bool   `json:"include_files"`
	IncludeConfig   bool   `json:"include_config"`
	Destination     string `json:"destination,omitempty"`
	Compress        bool   `json:"compress"`
}

type UserResponse struct {
	Success bool   `json:"success"`
	UserID  int    `json:"user_id,omitempty"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type AdminStats struct {
	Users       UserStats        `json:"users"`
	System      SystemOverview   `json:"system"`
	Activity    ActivityStats    `json:"activity"`
	Maintenance MaintenanceStats `json:"maintenance"`
}

type UserStats struct {
	Total         int64            `json:"total"`
	Active        int64            `json:"active"`
	Inactive      int64            `json:"inactive"`
	ByRole        map[string]int64 `json:"by_role"`
	Recent        int64            `json:"recent_logins_24h"`
	NeverLoggedIn int64            `json:"never_logged_in"`
}

type SystemOverview struct {
	DatabaseSize float64         `json:"database_size_mb"`
	FileCount    int64           `json:"total_files"`
	StorageUsed  float64         `json:"storage_used_gb"`
	ConfigCount  int             `json:"config_entries"`
	HealthScore  int             `json:"health_score"`
	Services     map[string]bool `json:"service_status"`
}

type ActivityStats struct {
	APIRequests    int64 `json:"api_requests_24h"`
	Downloads      int64 `json:"downloads_24h"`
	Alerts         int64 `json:"alerts_24h"`
	JobsExecuted   int64 `json:"jobs_executed_24h"`
	WebhooksSent   int64 `json:"webhooks_sent_24h"`
	ErrorsReported int64 `json:"errors_24h"`
}

type MaintenanceStats struct {
	ScheduledTasks int64      `json:"scheduled_tasks"`
	RunningTasks   int64      `json:"running_tasks"`
	CompletedTasks int64      `json:"completed_tasks_24h"`
	FailedTasks    int64      `json:"failed_tasks_24h"`
	NextScheduled  *time.Time `json:"next_scheduled,omitempty"`
	LastBackup     *time.Time `json:"last_backup,omitempty"`
	LastCleanup    *time.Time `json:"last_cleanup,omitempty"`
}

type ScheduleConfig struct {
	CatalogRefresh string `json:"catalog_refresh"` // Cron expression
	MonitorCheck   string `json:"monitor_check"`   // Cron expression
	SystemCleanup  string `json:"system_cleanup"`  // Cron expression
	DatabaseBackup string `json:"database_backup"` // Cron expression
	HealthCheck    string `json:"health_check"`    // Cron expression
	LogRotation    string `json:"log_rotation"`    // Cron expression
}

type NotificationSettings struct {
	EmailEnabled    bool     `json:"email_enabled"`
	WebhookEnabled  bool     `json:"webhook_enabled"`
	SlackEnabled    bool     `json:"slack_enabled"`
	AlertThreshold  string   `json:"alert_threshold"` // info, warning, error, critical
	Recipients      []string `json:"recipients"`
	SlackWebhookURL string   `json:"slack_webhook_url,omitempty"`
	EmailSMTPServer string   `json:"email_smtp_server,omitempty"`
	EmailFrom       string   `json:"email_from,omitempty"`
}
