package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobManager_CreateJob(t *testing.T) {
	jm := NewJobManager()

	job := jm.CreateJob(JobTypeCatalogRefresh)

	assert.NotEmpty(t, job.ID)
	assert.Equal(t, JobTypeCatalogRefresh, job.Type)
	assert.Equal(t, JobStatusPending, job.Status)
	assert.Equal(t, 0, job.Progress)
	assert.NotNil(t, job.Cancel)
	assert.WithinDuration(t, time.Now(), job.CreatedAt, time.Second)

	// Verify job is stored in manager
	retrievedJob, exists := jm.GetJob(job.ID)
	require.True(t, exists)
	assert.Equal(t, job.ID, retrievedJob.ID)
}

func TestJobManager_GetJob(t *testing.T) {
	jm := NewJobManager()

	// Test getting non-existent job
	job, exists := jm.GetJob("non-existent")
	assert.False(t, exists)
	assert.Nil(t, job)

	// Test getting existing job
	createdJob := jm.CreateJob(JobTypeDownload)
	retrievedJob, exists := jm.GetJob(createdJob.ID)
	require.True(t, exists)
	assert.Equal(t, createdJob.ID, retrievedJob.ID)
	assert.Equal(t, JobTypeDownload, retrievedJob.Type)
}

func TestJobManager_UpdateJob(t *testing.T) {
	jm := NewJobManager()
	job := jm.CreateJob(JobTypeMonitorCheck)

	// Test updating job status
	err := jm.UpdateJob(job.ID, func(j *Job) {
		j.Status = JobStatusRunning
		j.Progress = 50
		j.Message = "Processing artists"
	})
	assert.NoError(t, err)

	updatedJob, exists := jm.GetJob(job.ID)
	require.True(t, exists)
	assert.Equal(t, JobStatusRunning, updatedJob.Status)
	assert.Equal(t, 50, updatedJob.Progress)
	assert.Equal(t, "Processing artists", updatedJob.Message)

	// Test updating non-existent job
	err = jm.UpdateJob("non-existent", func(j *Job) {
		j.Status = JobStatusRunning
	})
	assert.Error(t, err)
}

func TestJobManager_CancelJob(t *testing.T) {
	jm := NewJobManager()
	job := jm.CreateJob(JobTypeCatalogRefresh)

	// Test canceling job
	err := jm.CancelJob(job.ID)
	assert.NoError(t, err)

	canceledJob, exists := jm.GetJob(job.ID)
	require.True(t, exists)
	assert.Equal(t, JobStatusCancelled, canceledJob.Status)

	// Test canceling non-existent job
	err = jm.CancelJob("non-existent")
	assert.Error(t, err)
}

func TestJobManager_ListJobs(t *testing.T) {
	jm := NewJobManager()

	// Create various jobs
	job1 := jm.CreateJob(JobTypeCatalogRefresh)
	job2 := jm.CreateJob(JobTypeDownload)
	job3 := jm.CreateJob(JobTypeMonitorCheck)

	// Update one job to running
	jm.UpdateJob(job2.ID, func(j *Job) {
		j.Status = JobStatusRunning
		j.Progress = 25
		j.Message = "Downloading..."
	})

	jobs := jm.ListJobs()
	assert.Equal(t, 3, len(jobs))

	// Check that all jobs are present
	jobIDs := make(map[string]bool)
	for _, job := range jobs {
		jobIDs[job.ID] = true
	}

	assert.True(t, jobIDs[job1.ID])
	assert.True(t, jobIDs[job2.ID])
	assert.True(t, jobIDs[job3.ID])
}

func TestJobManager_CleanupOldJobs(t *testing.T) {
	jm := NewJobManager()

	// Create old completed job by manually setting CreatedAt
	oldJob := jm.CreateJob(JobTypeDownload)
	jm.UpdateJob(oldJob.ID, func(j *Job) {
		j.Status = JobStatusCompleted
		j.CreatedAt = time.Now().Add(-25 * time.Hour) // Make it old
	})

	// Create recent job (will have current CreatedAt)
	recentJob := jm.CreateJob(JobTypeCatalogRefresh)

	// Cleanup jobs older than 24 hours
	cleanedCount := jm.CleanupOldJobs(24 * time.Hour)

	assert.Equal(t, 1, cleanedCount)

	// Old job should be removed
	_, exists := jm.GetJob(oldJob.ID)
	assert.False(t, exists)

	// Recent job should still exist
	_, exists = jm.GetJob(recentJob.ID)
	assert.True(t, exists)
}

func TestJob_IsCancellationRequested(t *testing.T) {
	jm := NewJobManager()
	job := jm.CreateJob(JobTypeDownload)

	// Initially no cancellation requested
	assert.False(t, job.IsCancellationRequested())

	// Cancel the job synchronously
	err := jm.CancelJob(job.ID)
	assert.NoError(t, err)

	// Now cancellation should be requested
	assert.True(t, job.IsCancellationRequested())
}
