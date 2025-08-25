package services

import (
	"database/sql"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jmagar/nugs/cron/internal/models"
)

type DownloadManager struct {
	DB              *sql.DB
	JobManager      *models.JobManager
	maxConcurrent   int
	downloadPath    string
	activeDownloads sync.Map
	queueMutex      sync.Mutex
}

type ActiveDownload struct {
	Download   *models.Download
	Job        *models.Job
	CancelChan chan bool
}

func NewDownloadManager(db *sql.DB, jobManager *models.JobManager) *DownloadManager {
	return &DownloadManager{
		DB:              db,
		JobManager:      jobManager,
		maxConcurrent:   3,                                  // Default to 3 concurrent downloads
		downloadPath:    "/home/jmagar/code/nugs/downloads", // Default path
		activeDownloads: sync.Map{},
	}
}

func (dm *DownloadManager) QueueDownload(req *models.DownloadRequest) (*models.DownloadResponse, error) {
	// Validate show exists and get details
	var showExists int
	var artistName sql.NullString
	err := dm.DB.QueryRow(`
		SELECT COUNT(*) as show_count,
		       COALESCE(a.name, 'Unknown Artist') as artist_name
		FROM shows s 
		LEFT JOIN artists a ON s.artist_id = a.id 
		WHERE s.container_id = ?
	`, req.ShowID).Scan(&showExists, &artistName)

	if err != nil || showExists == 0 {
		return &models.DownloadResponse{
			Success: false,
			Error:   "Show not found",
		}, fmt.Errorf("show not found: %d", req.ShowID)
	}

	artistNameStr := "Unknown Artist"
	if artistName.Valid {
		artistNameStr = artistName.String
	}

	// Set defaults
	if req.Quality == "" {
		req.Quality = models.DownloadQualityStandard
	}
	if req.Priority == 0 {
		req.Priority = 5
	}

	// Check if download already exists
	var existingID int
	err = dm.DB.QueryRow(`
		SELECT id FROM downloads 
		WHERE show_id = ? AND format = ? AND quality = ?
		AND status NOT IN ('failed', 'cancelled')
	`, req.ShowID, req.Format, req.Quality).Scan(&existingID)

	if err == nil {
		return &models.DownloadResponse{
			Success: false,
			Error:   "Download already exists for this show/format/quality combination",
		}, nil
	}

	// Create download record
	result, err := dm.DB.Exec(`
		INSERT INTO downloads (user_id, show_id, container_id, artist_name, show_date, venue, format, quality, status, size_mb, created_at)
		SELECT 1, s.id, s.container_id, ?, s.date, s.venue, ?, ?, 'pending', 0, datetime('now')
		FROM shows s WHERE s.container_id = ?
	`, artistNameStr, string(req.Format), string(req.Quality), req.ShowID)

	if err != nil {
		return &models.DownloadResponse{
			Success: false,
			Error:   "Failed to create download record",
		}, err
	}

	downloadID, err := result.LastInsertId()
	if err != nil {
		return &models.DownloadResponse{
			Success: false,
			Error:   "Failed to get download ID",
		}, err
	}

	// Set queue position
	_, err = dm.DB.Exec(`
		UPDATE downloads SET queue_position = (
			SELECT COALESCE(MAX(queue_position), 0) + 1 
			FROM downloads 
			WHERE status IN ('pending', 'queued')
		), status = 'queued'
		WHERE id = ?
	`, downloadID)

	if err != nil {
		return &models.DownloadResponse{
			Success: false,
			Error:   "Failed to set queue position",
		}, err
	}

	// Start download processing if not at capacity
	go dm.processQueue()

	return &models.DownloadResponse{
		Success:    true,
		DownloadID: int(downloadID),
		Status:     "queued",
		Message:    fmt.Sprintf("Download queued for %s", artistNameStr),
	}, nil
}

func (dm *DownloadManager) processQueue() {
	dm.queueMutex.Lock()
	defer dm.queueMutex.Unlock()

	// Check how many downloads are currently active
	activeCount := 0
	dm.activeDownloads.Range(func(key, value interface{}) bool {
		activeCount++
		return true
	})

	if activeCount >= dm.maxConcurrent {
		return // Already at capacity
	}

	// Get next download from queue
	rows, err := dm.DB.Query(`
		SELECT d.id, d.show_id, d.container_id, d.artist_name, 
		       d.format, d.quality, d.status, s.venue, s.city
		FROM downloads d
		JOIN shows s ON d.show_id = s.id
		WHERE d.status = 'queued' AND d.queue_position IS NOT NULL
		ORDER BY d.queue_position ASC
		LIMIT ?
	`, dm.maxConcurrent-activeCount)

	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var downloadID, showID, containerID int
		var artistName, format, quality, status, venueName, venueCity string

		err := rows.Scan(&downloadID, &showID, &containerID, &artistName,
			&format, &quality, &status, &venueName, &venueCity)
		if err != nil {
			continue
		}

		download := &models.Download{
			ID:          downloadID,
			ShowID:      showID,
			ContainerID: containerID,
			ArtistName:  artistName,
			Format:      models.DownloadFormat(format),
			Quality:     models.DownloadQuality(quality),
			Status:      models.DownloadStatus(status),
			ShowTitle:   venueName + ", " + venueCity,
			VenueName:   venueName,
		}

		// Start download in background
		go dm.startDownload(download)
	}
}

func (dm *DownloadManager) startDownload(download *models.Download) {
	// Create job for tracking
	job := dm.JobManager.CreateJob(models.JobTypeDownload)

	activeDownload := &ActiveDownload{
		Download:   download,
		Job:        job,
		CancelChan: make(chan bool, 1),
	}

	dm.activeDownloads.Store(download.ID, activeDownload)
	defer dm.activeDownloads.Delete(download.ID)

	// Update job status
	if err := dm.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Status = models.JobStatusRunning
		j.StartedAt = time.Now()
		j.Message = fmt.Sprintf("Starting download: %s - %s", download.ArtistName, download.ShowTitle)
	}); err != nil {
		log.Printf("Error updating job status: %v", err)
		return
	}

	// Update download status
	dm.updateDownloadStatus(download.ID, models.DownloadStatusInProgress, "")

	// Execute download using nugs-dl
	err := dm.executeDownload(download, job)

	completedAt := time.Now()
	if err != nil {
		// Download failed
		dm.updateDownloadStatus(download.ID, models.DownloadStatusFailed, err.Error())

		if updateErr := dm.JobManager.UpdateJob(job.ID, func(j *models.Job) {
			j.Status = models.JobStatusFailed
			j.Error = err.Error()
			j.Message = "Download failed"
			j.CompletedAt = &completedAt
		}); updateErr != nil {
			log.Printf("Error updating job status: %v", updateErr)
		}
	} else {
		// Download succeeded
		dm.updateDownloadStatus(download.ID, models.DownloadStatusCompleted, "")

		if updateErr := dm.JobManager.UpdateJob(job.ID, func(j *models.Job) {
			j.Status = models.JobStatusCompleted
			j.Progress = 100
			j.Message = "Download completed successfully"
			j.CompletedAt = &completedAt
		}); updateErr != nil {
			log.Printf("Error updating job status: %v", updateErr)
		}
	}

	// Remove from queue by clearing queue position
	if _, err := dm.DB.Exec("UPDATE downloads SET queue_position = NULL WHERE id = ?", download.ID); err != nil {
		log.Printf("Warning: failed to clear queue position: %v", err)
	}

	// Process next in queue
	go dm.processQueue()
}

func (dm *DownloadManager) executeDownload(download *models.Download, job *models.Job) error {
	// Map format to nugs-dl format number
	// 1 = ALAC, 2 = FLAC, 4 = 360 Reality Audio
	var formatNum string
	switch download.Format {
	case models.DownloadFormatFLAC:
		formatNum = "2" // 16-bit / 44.1 kHz FLAC
	case models.DownloadFormatALAC:
		formatNum = "1" // 16-bit / 44.1 kHz ALAC
	case models.DownloadFormatMP3:
		formatNum = "4" // 360 Reality Audio / best available
	default:
		formatNum = "2" // Default to FLAC
	}

	// Build nugs-dl command - nugs-dl expects URLs as positional arguments
	// Based on REFERENCE_CODE/README.md, container IDs map to release URLs
	containerURL := fmt.Sprintf("https://play.nugs.net/release/%d", download.ContainerID)
	cmd := exec.Command("./nugs-dl",
		"--format", formatNum,
		"--outpath", dm.downloadPath,
		containerURL)

	cmd.Dir = "/home/jmagar/code/nugs"

	// Log the command being executed for debugging
	log.Printf("Executing download command: %s (args: %v) in directory: %s",
		cmd.Path, cmd.Args, cmd.Dir)

	// Start the command
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start download command: %v", err)
		return fmt.Errorf("failed to start download: %v", err)
	}

	// Monitor the download process
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Update progress periodically (simplified - in real implementation, parse nugs-dl output)
	progressTicker := time.NewTicker(5 * time.Second)
	defer progressTicker.Stop()

	progress := 10
	for {
		select {
		case <-job.Cancel:
			if err := cmd.Process.Kill(); err != nil {
				log.Printf("Warning: failed to kill download process: %v", err)
			}
			return fmt.Errorf("download cancelled")

		case <-progressTicker.C:
			if progress < 90 {
				progress += 10
				if err := dm.JobManager.UpdateJob(job.ID, func(j *models.Job) {
					j.Progress = progress
					j.Message = fmt.Sprintf("Downloading... %d%%", progress)
				}); err != nil {
					log.Printf("Warning: failed to update job progress: %v", err)
				}
			}

		case err := <-done:
			if err != nil {
				log.Printf("Download command failed for container %d: %v", download.ContainerID, err)
				return fmt.Errorf("download command failed: %v", err)
			}
			log.Printf("Download command completed successfully for container %d", download.ContainerID)

			// Calculate file size and update record
			filePath := filepath.Join(dm.downloadPath, fmt.Sprintf("%s_%d.%s",
				strings.ReplaceAll(download.ArtistName, " ", "_"),
				download.ContainerID,
				download.Format))

			fileSize := dm.getFileSize(filePath)

			if _, err := dm.DB.Exec(`
				UPDATE downloads 
				SET file_path = ?, file_size = ?, downloaded_at = datetime('now')
				WHERE id = ?
			`, filePath, fileSize, download.ID); err != nil {
				log.Printf("Warning: failed to update download file info: %v", err)
			}

			return nil
		}
	}
}

func (dm *DownloadManager) updateDownloadStatus(downloadID int, status models.DownloadStatus, errorMsg string) {
	if errorMsg != "" {
		if _, err := dm.DB.Exec(`
			UPDATE downloads 
			SET status = ?, error_message = ?
			WHERE id = ?
		`, status, errorMsg, downloadID); err != nil {
			log.Printf("Warning: failed to update download status with error: %v", err)
		}
	} else {
		if _, err := dm.DB.Exec(`
			UPDATE downloads 
			SET status = ?
			WHERE id = ?
		`, status, downloadID); err != nil {
			log.Printf("Warning: failed to update download status: %v", err)
		}
	}
}

func (dm *DownloadManager) getFileSize(filePath string) int64 {
	// Simplified file size calculation
	// In real implementation, check actual file size
	return 0
}

func (dm *DownloadManager) GetDownloadStats() (*models.DownloadStats, error) {
	stats := &models.DownloadStats{
		FormatBreakdown:  make(map[string]int64),
		QualityBreakdown: make(map[string]int64),
	}

	// Get overall stats
	err := dm.DB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COUNT(CASE WHEN status IN ('pending', 'queued') THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'downloading' THEN 1 END) as in_progress,
			COALESCE(SUM(size_mb), 0) / 1024.0 as total_gb
		FROM downloads
	`).Scan(&stats.TotalDownloads, &stats.CompletedDownloads, &stats.FailedDownloads,
		&stats.PendingDownloads, &stats.InProgressDownloads, &stats.TotalSizeGB)

	if err != nil {
		return nil, err
	}

	// Calculate additional stats
	stats.QueueLength = stats.PendingDownloads
	stats.ActiveDownloads = stats.InProgressDownloads
	stats.AverageSpeedMbps = 0.0 // Placeholder - would need actual speed tracking

	// Get format breakdown
	rows, err := dm.DB.Query(`
		SELECT format, COUNT(*) 
		FROM downloads 
		GROUP BY format
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var format string
			var count int64
			if rows.Scan(&format, &count) == nil {
				stats.FormatBreakdown[format] = count
			}
		}
	}

	// Get quality breakdown
	rows, err = dm.DB.Query(`
		SELECT quality, COUNT(*) 
		FROM downloads 
		GROUP BY quality
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var quality string
			var count int64
			if rows.Scan(&quality, &count) == nil {
				stats.QualityBreakdown[quality] = count
			}
		}
	}

	return stats, nil
}

func (dm *DownloadManager) CancelDownload(downloadID int) error {
	// Check if download is active
	if active, ok := dm.activeDownloads.Load(downloadID); ok {
		activeDownload := active.(*ActiveDownload)
		select {
		case activeDownload.CancelChan <- true:
			dm.updateDownloadStatus(downloadID, models.DownloadStatusCancelled, "Cancelled by user")
			return nil
		default:
		}
	}

	// If not active, just update status
	result, err := dm.DB.Exec(`
		UPDATE downloads 
		SET status = 'cancelled' 
		WHERE id = ? AND status IN ('pending', 'in_progress')
	`, downloadID)

	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("download cannot be cancelled (not pending or in progress)")
	}

	// Remove from queue
	if _, err := dm.DB.Exec("DELETE FROM download_queue WHERE download_id = ?", downloadID); err != nil {
		log.Printf("Warning: failed to remove download from queue: %v", err)
	}

	return nil
}
