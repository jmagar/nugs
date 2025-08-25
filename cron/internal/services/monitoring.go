package services

import (
	"database/sql"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jmagar/nugs/cron/internal/models"
)

type MonitoringService struct {
	DB         *sql.DB
	JobManager *models.JobManager
}

func NewMonitoringService(db *sql.DB, jobManager *models.JobManager) *MonitoringService {
	return &MonitoringService{
		DB:         db,
		JobManager: jobManager,
	}
}

func (s *MonitoringService) CreateMonitor(req *models.MonitorRequest) (*models.MonitorResponse, error) {
	// Validate artist exists
	var artistName string
	err := s.DB.QueryRow(`SELECT name FROM artists WHERE id = ?`, req.ArtistID).Scan(&artistName)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.MonitorResponse{
				Success: false,
				Error:   "Artist not found",
			}, nil
		}
		return nil, err
	}

	// Check if monitor already exists for this user and artist
	var existingID int
	err = s.DB.QueryRow(`SELECT id FROM monitors WHERE user_id = 1 AND artist_id = ?`, req.ArtistID).Scan(&existingID)
	if err == nil {
		return &models.MonitorResponse{
			Success: false,
			Error:   "Monitor already exists for this artist",
		}, nil
	}

	// Set defaults and create settings JSON
	if req.CheckInterval == 0 {
		req.CheckInterval = 60 // 1 hour default (minutes)
	}

	settings := fmt.Sprintf(`{
		"check_interval": %d,
		"notify_new_shows": %t,
		"notify_show_updates": %t
	}`, req.CheckInterval, req.NotifyNewShows, req.NotifyShowUpdates)

	// Create monitor
	result, err := s.DB.Exec(`
		INSERT INTO monitors (user_id, artist_id, status, settings, shows_found, alerts_sent, created_at, updated_at)
		VALUES (1, ?, 'active', ?, 0, 0, datetime('now'), datetime('now'))
	`, req.ArtistID, settings)

	if err != nil {
		return &models.MonitorResponse{
			Success: false,
			Error:   "Failed to create monitor",
		}, err
	}

	monitorID, _ := result.LastInsertId()

	return &models.MonitorResponse{
		Success:   true,
		MonitorID: int(monitorID),
		Message:   fmt.Sprintf("Monitor created for %s", artistName),
	}, nil
}

func (s *MonitoringService) UpdateMonitor(monitorID int, req *models.MonitorUpdateRequest) error {
	updates := []string{}
	args := []interface{}{}

	if req.Status != nil {
		updates = append(updates, "status = ?")
		args = append(args, *req.Status)
	}

	if req.CheckInterval != nil {
		updates = append(updates, "check_interval = ?")
		args = append(args, *req.CheckInterval)
	}

	if req.NotifyNewShows != nil {
		updates = append(updates, "notify_new_shows = ?")
		args = append(args, *req.NotifyNewShows)
	}

	if req.NotifyShowUpdates != nil {
		updates = append(updates, "notify_show_updates = ?")
		args = append(args, *req.NotifyShowUpdates)
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	updates = append(updates, "updated_at = datetime('now')")
	args = append(args, monitorID)

	query := fmt.Sprintf("UPDATE artist_monitors SET %s WHERE id = ?", strings.Join(updates, ", "))

	result, err := s.DB.Exec(query, args...)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("monitor not found")
	}

	return nil
}

func (s *MonitoringService) DeleteMonitor(monitorID int) error {
	result, err := s.DB.Exec("DELETE FROM artist_monitors WHERE id = ?", monitorID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("monitor not found")
	}

	// Also delete related alerts
	if _, err := s.DB.Exec("DELETE FROM monitor_alerts WHERE monitor_id = ?", monitorID); err != nil {
		log.Printf("Warning: failed to delete monitor alerts: %v", err)
	}

	return nil
}

func (s *MonitoringService) CheckAllMonitors() *models.Job {
	job := s.JobManager.CreateJob(models.JobTypeMonitorCheck)
	go s.runMonitoringCheck(job)
	return job
}

func (s *MonitoringService) CheckArtist(artistID int) (*models.CheckResult, error) {
	// Get current show count
	var currentCount int
	var artistName string
	err := s.DB.QueryRow(`
		SELECT COUNT(s.id), a.name 
		FROM shows s 
		JOIN artists a ON s.artist_id = a.id 
		WHERE s.artist_id = ?
	`, artistID).Scan(&currentCount, &artistName)

	if err != nil {
		return nil, fmt.Errorf("failed to get current show count: %v", err)
	}

	// Get monitor for this artist
	var monitor models.ArtistMonitor
	err = s.DB.QueryRow(`
		SELECT id, total_shows 
		FROM artist_monitors 
		WHERE artist_id = ? AND status = 'active'
	`, artistID).Scan(&monitor.ID, &monitor.TotalShows)

	if err != nil {
		return &models.CheckResult{
			ArtistID:   artistID,
			ArtistName: artistName,
			Success:    false,
			Error:      "No active monitor found for artist",
		}, nil
	}

	startTime := time.Now()

	// Use catalog_manager to refresh this specific artist
	cmd := exec.Command("./bin/catalog_manager", "artist", strconv.Itoa(artistID))
	cmd.Dir = "/home/jmagar/code/nugs/cron"

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &models.CheckResult{
			ArtistID:      artistID,
			ArtistName:    artistName,
			PreviousCount: monitor.TotalShows,
			CurrentCount:  currentCount,
			Success:       false,
			Error:         fmt.Sprintf("catalog_manager failed: %v", err),
		}, nil
	}

	// Get updated show count
	var newCurrentCount int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM shows WHERE artist_id = ?`, artistID).Scan(&newCurrentCount); err != nil {
		log.Printf("Warning: failed to get updated show count for artist %d: %v", artistID, err)
		newCurrentCount = currentCount // Use original count as fallback
	}

	newShows := newCurrentCount - monitor.TotalShows
	if newShows < 0 {
		newShows = 0
	}

	// Update monitor
	if _, err := s.DB.Exec(`
		UPDATE artist_monitors 
		SET total_shows = ?, new_shows_found = new_shows_found + ?, 
		    last_checked = datetime('now'), updated_at = datetime('now')
		WHERE id = ?
	`, newCurrentCount, newShows, monitor.ID); err != nil {
		log.Printf("Warning: failed to update monitor: %v", err)
	}

	if newShows > 0 {
		if _, err := s.DB.Exec(`
			UPDATE artist_monitors 
			SET last_new_show = datetime('now') 
			WHERE id = ?
		`, monitor.ID); err != nil {
			log.Printf("Warning: failed to update last new show time: %v", err)
		}

		// Create alert for new shows
		s.createAlert(monitor.ID, artistID, 0, models.AlertTypeNewShow,
			fmt.Sprintf("%d new show(s) found for %s", newShows, artistName),
			string(output))
	}

	return &models.CheckResult{
		ArtistID:      artistID,
		ArtistName:    artistName,
		PreviousCount: monitor.TotalShows,
		CurrentCount:  newCurrentCount,
		NewShows:      newShows,
		CheckDuration: time.Since(startTime).String(),
		Success:       true,
	}, nil
}

func (s *MonitoringService) runMonitoringCheck(job *models.Job) {
	startTime := time.Now()

	if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Status = models.JobStatusRunning
		j.StartedAt = startTime
		j.Message = "Starting monitoring check for all active artists..."
	}); err != nil {
		log.Printf("Warning: failed to update job status: %v", err)
		return
	}

	// Get all active monitors
	rows, err := s.DB.Query(`
		SELECT id, artist_id, artist_name, check_interval, last_checked
		FROM artist_monitors 
		WHERE status = 'active'
		ORDER BY last_checked ASC NULLS FIRST
	`)

	if err != nil {
		if updateErr := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
			j.Status = models.JobStatusFailed
			j.Error = err.Error()
			completedAt := time.Now()
			j.CompletedAt = &completedAt
		}); updateErr != nil {
			log.Printf("Warning: failed to update job status: %v", updateErr)
		}
		return
	}
	defer rows.Close()

	var results []models.CheckResult
	processedCount := 0
	successCount := 0

	for rows.Next() {
		var monitorID, artistID, checkInterval int
		var artistName string
		var lastChecked sql.NullString

		err := rows.Scan(&monitorID, &artistID, &artistName, &checkInterval, &lastChecked)
		if err != nil {
			continue
		}

		// Check if enough time has passed since last check
		if lastChecked.Valid {
			lastCheck, err := time.Parse("2006-01-02 15:04:05", lastChecked.String)
			if err == nil {
				nextCheck := lastCheck.Add(time.Duration(checkInterval) * time.Minute)
				if time.Now().Before(nextCheck) {
					continue // Skip this monitor, not time yet
				}
			}
		}

		processedCount++
		if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
			j.Progress = int(float64(processedCount) / 10.0 * 90) // Reserve 10% for final processing
			j.Message = fmt.Sprintf("Checking %s (%d of ?)", artistName, processedCount)
		}); err != nil {
			log.Printf("Warning: failed to update job progress: %v", err)
		}

		result, err := s.CheckArtist(artistID)
		if err == nil && result.Success {
			successCount++
		}
		if result != nil {
			results = append(results, *result)
		}

		// Small delay between checks to avoid overwhelming the system
		time.Sleep(1 * time.Second)
	}

	// Complete the job
	completedAt := time.Now()
	if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Status = models.JobStatusCompleted
		j.Progress = 100
		j.Message = fmt.Sprintf("Monitoring check completed: %d/%d successful", successCount, processedCount)
		j.Result = map[string]interface{}{
			"processed_count": processedCount,
			"success_count":   successCount,
			"results":         results,
			"duration":        time.Since(startTime).String(),
		}
		j.CompletedAt = &completedAt
	}); err != nil {
		log.Printf("Warning: failed to update job completion status: %v", err)
	}
}

func (s *MonitoringService) createAlert(monitorID, artistID, showID int, alertType models.AlertType, message, details string) {
	if _, err := s.DB.Exec(`
		INSERT INTO monitor_alerts (monitor_id, artist_id, show_id, alert_type, message, details, created_at)
		VALUES (?, ?, NULLIF(?, 0), ?, ?, ?, datetime('now'))
	`, monitorID, artistID, showID, alertType, message, details); err != nil {
		log.Printf("Warning: failed to create alert: %v", err)
	}
}

func (s *MonitoringService) GetMonitorStats() (*models.MonitorStats, error) {
	stats := &models.MonitorStats{}

	// Get monitor counts
	err := s.DB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
			COUNT(CASE WHEN status = 'paused' THEN 1 END) as paused
		FROM artist_monitors
	`).Scan(&stats.TotalMonitors, &stats.ActiveMonitors, &stats.PausedMonitors)

	if err != nil {
		return nil, err
	}

	// Get alert counts
	if err := s.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM monitor_alerts 
		WHERE date(created_at) = date('now')
	`).Scan(&stats.TotalAlertsToday); err != nil {
		log.Printf("Warning: failed to get total alerts today: %v", err)
		stats.TotalAlertsToday = 0
	}

	if err := s.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM monitor_alerts 
		WHERE status != 'acknowledged'
	`).Scan(&stats.UnacknowledgedAlerts); err != nil {
		log.Printf("Warning: failed to get unacknowledged alerts: %v", err)
		stats.UnacknowledgedAlerts = 0
	}

	// Get last check time
	var lastCheckStr sql.NullString
	if err := s.DB.QueryRow(`
		SELECT last_check 
		FROM artist_monitors 
		WHERE last_check IS NOT NULL 
		ORDER BY last_check DESC 
		LIMIT 1
	`).Scan(&lastCheckStr); err != nil {
		log.Printf("Warning: failed to get last check time: %v", err)
	}

	if lastCheckStr.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", lastCheckStr.String); err == nil {
			stats.LastCheckTime = &t
		}
	}

	return stats, nil
}

func (s *MonitoringService) CreateBulkMonitors(req *models.BulkMonitorRequest) (*models.BulkMonitorResponse, error) {
	response := &models.BulkMonitorResponse{
		ProcessedCount: len(req.ArtistIDs),
		SuccessCount:   0,
		FailedCount:    0,
		Errors:         []string{},
	}

	for _, artistID := range req.ArtistIDs {
		monitorReq := &models.MonitorRequest{
			ArtistID:          artistID,
			CheckInterval:     req.CheckInterval,
			NotifyNewShows:    req.NotifyNewShows,
			NotifyShowUpdates: req.NotifyShowUpdates,
		}

		result, err := s.CreateMonitor(monitorReq)
		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, fmt.Sprintf("Artist ID %d: %v", artistID, err))
		} else if !result.Success {
			response.FailedCount++
			response.Errors = append(response.Errors, fmt.Sprintf("Artist ID %d: %s", artistID, result.Error))
		} else {
			response.SuccessCount++
		}
	}

	response.Success = response.SuccessCount > 0
	response.Message = fmt.Sprintf("Created %d monitors successfully, %d failed", response.SuccessCount, response.FailedCount)

	return response, nil
}
