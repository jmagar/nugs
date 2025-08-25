package models

import (
	"fmt"
	"sync"
	"time"
)

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

type JobType string

const (
	JobTypeCatalogRefresh JobType = "catalog_refresh"
	JobTypeDownload       JobType = "download"
	JobTypeMonitorCheck   JobType = "monitor_check"
	JobTypeAnalytics      JobType = "analytics"
)

type Job struct {
	ID          string     `json:"id"`
	Type        JobType    `json:"type"`
	Status      JobStatus  `json:"status"`
	Progress    int        `json:"progress"` // 0-100
	Message     string     `json:"message"`
	Error       string     `json:"error,omitempty"`
	Result      any        `json:"result,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`

	// Internal fields
	Cancel chan bool `json:"-"`
}

// IsCancellationRequested checks if a cancellation has been requested for this job
func (j *Job) IsCancellationRequested() bool {
	select {
	case <-j.Cancel:
		return true
	default:
		return false
	}
}

type JobManager struct {
	jobs map[string]*Job
	mu   sync.RWMutex
}

func NewJobManager() *JobManager {
	return &JobManager{
		jobs: make(map[string]*Job),
	}
}

func (jm *JobManager) CreateJob(jobType JobType) *Job {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	job := &Job{
		ID:        generateJobID(),
		Type:      jobType,
		Status:    JobStatusPending,
		Progress:  0,
		CreatedAt: time.Now(),
		Cancel:    make(chan bool, 1),
	}

	jm.jobs[job.ID] = job
	return job
}

func (jm *JobManager) GetJob(id string) (*Job, bool) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	job, exists := jm.jobs[id]
	return job, exists
}

func (jm *JobManager) UpdateJob(id string, updates func(*Job)) error {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	job, exists := jm.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	updates(job)
	return nil
}

func (jm *JobManager) ListJobs() []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	jobs := make([]*Job, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		jobs = append(jobs, job)
	}

	return jobs
}

func (jm *JobManager) CancelJob(id string) error {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	job, exists := jm.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	select {
	case job.Cancel <- true:
		job.Status = JobStatusCancelled
	default:
		// Channel full or job already completed
	}

	return nil
}

func (jm *JobManager) CleanupOldJobs(maxAge time.Duration) int {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for id, job := range jm.jobs {
		if job.CreatedAt.Before(cutoff) && job.Status != JobStatusRunning {
			delete(jm.jobs, id)
			cleaned++
		}
	}

	return cleaned
}

func generateJobID() string {
	// Generate UUID v4
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(time.Now().UnixNano() % 256)
	}

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant bits

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Errors
var (
	ErrJobNotFound = fmt.Errorf("job not found")
)
