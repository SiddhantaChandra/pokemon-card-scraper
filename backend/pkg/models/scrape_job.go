package models

import (
	"time"
)

// ScrapeJobStatus represents the current status of a scraping job
type ScrapeJobStatus string

const (
	StatusPending    ScrapeJobStatus = "pending"
	StatusRunning    ScrapeJobStatus = "running"
	StatusCompleted  ScrapeJobStatus = "completed"
	StatusFailed     ScrapeJobStatus = "failed"
	StatusCancelled  ScrapeJobStatus = "cancelled"
)

// ScrapeJob represents a scraping operation with its metadata and progress
type ScrapeJob struct {
	ID             string          `json:"id" db:"id"`
	Status         ScrapeJobStatus `json:"status" db:"status"`
	StartedAt      *time.Time      `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time      `json:"completed_at" db:"completed_at"`
	ItemsScraped   int             `json:"items_scraped" db:"items_scraped"`
	ItemsUpdated   int             `json:"items_updated" db:"items_updated"`
	ItemsAdded     int             `json:"items_added" db:"items_added"`
	ErrorMessage   string          `json:"error_message,omitempty" db:"error_message"`
	TotalPages     int             `json:"total_pages" db:"total_pages"`
	CurrentPage    int             `json:"current_page" db:"current_page"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

// ScrapeStats represents overall scraping statistics
type ScrapeStats struct {
	TotalJobs          int       `json:"total_jobs"`
	SuccessfulJobs     int       `json:"successful_jobs"`
	FailedJobs         int       `json:"failed_jobs"`
	LastSuccessfulRun  *time.Time `json:"last_successful_run"`
	LastFailedRun      *time.Time `json:"last_failed_run"`
	TotalCardsScraped  int       `json:"total_cards_scraped"`
	TotalCardsInStock  int       `json:"total_cards_in_stock"`
	AverageJobDuration float64   `json:"average_job_duration_minutes"`
}

// NewScrapeJob creates a new scrape job with pending status
func NewScrapeJob() *ScrapeJob {
	now := time.Now()
	return &ScrapeJob{
		ID:        generateJobID(),
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Start marks the job as running and sets the start time
func (sj *ScrapeJob) Start() {
	now := time.Now()
	sj.Status = StatusRunning
	sj.StartedAt = &now
	sj.UpdatedAt = now
}

// Complete marks the job as completed and sets the completion time
func (sj *ScrapeJob) Complete() {
	now := time.Now()
	sj.Status = StatusCompleted
	sj.CompletedAt = &now
	sj.UpdatedAt = now
}

// Fail marks the job as failed with an error message
func (sj *ScrapeJob) Fail(errorMsg string) {
	now := time.Now()
	sj.Status = StatusFailed
	sj.ErrorMessage = errorMsg
	sj.CompletedAt = &now
	sj.UpdatedAt = now
}

// Cancel marks the job as cancelled
func (sj *ScrapeJob) Cancel() {
	now := time.Now()
	sj.Status = StatusCancelled
	sj.CompletedAt = &now
	sj.UpdatedAt = now
}

// Duration returns the duration of the job if it has started
func (sj *ScrapeJob) Duration() *time.Duration {
	if sj.StartedAt == nil {
		return nil
	}
	
	var endTime time.Time
	if sj.CompletedAt != nil {
		endTime = *sj.CompletedAt
	} else {
		endTime = time.Now()
	}
	
	duration := endTime.Sub(*sj.StartedAt)
	return &duration
}

// IsActive returns true if the job is currently running
func (sj *ScrapeJob) IsActive() bool {
	return sj.Status == StatusRunning || sj.Status == StatusPending
}

// Progress returns the current progress as a percentage (0-100)
func (sj *ScrapeJob) Progress() float64 {
	if sj.TotalPages == 0 {
		return 0
	}
	return float64(sj.CurrentPage) / float64(sj.TotalPages) * 100
}

// generateJobID creates a unique job ID
func generateJobID() string {
	return time.Now().Format("20060102-150405") + "-" + randomString(6)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
} 