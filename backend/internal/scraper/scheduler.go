package scraper

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
)

// Scheduler manages scraping jobs and their execution
type Scheduler struct {
	scraper        *Scraper
	interval       time.Duration
	running        bool
	currentJob     *models.ScrapeJob
	jobHistory     []*models.ScrapeJob
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	jobCallbacks   map[string]JobCallback
}

// JobCallback defines the interface for job event callbacks
type JobCallback func(job *models.ScrapeJob)

// SchedulerConfig holds configuration for the scheduler
type SchedulerConfig struct {
	ScrapeInterval    time.Duration
	MaxJobHistory     int
	AutoStart         bool
}

// DefaultSchedulerConfig returns sensible defaults
func DefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		ScrapeInterval: 5 * time.Minute, // Scrape every 5 minutes
		MaxJobHistory:  50,              // Keep last 50 jobs
		AutoStart:      false,           // Don't auto-start
	}
}

// NewScheduler creates a new scraping scheduler
func NewScheduler(scraper *Scraper, config *SchedulerConfig) *Scheduler {
	if config == nil {
		config = DefaultSchedulerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		scraper:      scraper,
		interval:     config.ScrapeInterval,
		running:      false,
		jobHistory:   make([]*models.ScrapeJob, 0, config.MaxJobHistory),
		ctx:          ctx,
		cancel:       cancel,
		jobCallbacks: make(map[string]JobCallback),
	}
}

// Start begins the scheduled scraping process
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.running = true
	log.Println("Starting scraping scheduler...")

	// Start the scheduling goroutine
	go s.scheduleLoop()

	return nil
}

// Stop halts the scheduled scraping process
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	log.Println("Stopping scraping scheduler...")
	s.running = false
	s.cancel()

	// Cancel current job if running
	if s.currentJob != nil && s.currentJob.IsActive() {
		s.currentJob.Cancel()
		log.Printf("Cancelled running job: %s", s.currentJob.ID)
	}

	return nil
}

// RunOnce executes a single scraping job immediately
func (s *Scheduler) RunOnce() (*models.ScrapeJob, error) {
	s.mu.Lock()
	if s.currentJob != nil && s.currentJob.IsActive() {
		s.mu.Unlock()
		return nil, fmt.Errorf("a scraping job is already running")
	}
	s.mu.Unlock()

	job := models.NewScrapeJob()
	return s.executeJob(job)
}

// GetCurrentJob returns the currently running job, if any
func (s *Scheduler) GetCurrentJob() *models.ScrapeJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentJob
}

// GetJobHistory returns the history of completed jobs
func (s *Scheduler) GetJobHistory() []*models.ScrapeJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	history := make([]*models.ScrapeJob, len(s.jobHistory))
	copy(history, s.jobHistory)
	return history
}

// GetStats returns overall scraping statistics
func (s *Scheduler) GetStats() *models.ScrapeStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &models.ScrapeStats{
		TotalJobs: len(s.jobHistory),
	}

	var totalDuration time.Duration
	var durationCount int

	for _, job := range s.jobHistory {
		switch job.Status {
		case models.StatusCompleted:
			stats.SuccessfulJobs++
			if stats.LastSuccessfulRun == nil || job.CompletedAt.After(*stats.LastSuccessfulRun) {
				stats.LastSuccessfulRun = job.CompletedAt
			}
		case models.StatusFailed:
			stats.FailedJobs++
			if stats.LastFailedRun == nil || job.CompletedAt.After(*stats.LastFailedRun) {
				stats.LastFailedRun = job.CompletedAt
			}
		}

		stats.TotalCardsScraped += job.ItemsScraped

		// Calculate average duration
		if duration := job.Duration(); duration != nil {
			totalDuration += *duration
			durationCount++
		}
	}

	if durationCount > 0 {
		avgDuration := totalDuration / time.Duration(durationCount)
		stats.AverageJobDuration = avgDuration.Minutes()
	}

	return stats
}

// IsRunning returns whether the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// SetInterval updates the scraping interval
func (s *Scheduler) SetInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = interval
	log.Printf("Scraping interval updated to: %v", interval)
}

// OnJobStart registers a callback for when jobs start
func (s *Scheduler) OnJobStart(callback JobCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobCallbacks["start"] = callback
}

// OnJobComplete registers a callback for when jobs complete
func (s *Scheduler) OnJobComplete(callback JobCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobCallbacks["complete"] = callback
}

// OnJobFail registers a callback for when jobs fail
func (s *Scheduler) OnJobFail(callback JobCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobCallbacks["fail"] = callback
}

// scheduleLoop runs the main scheduling loop
func (s *Scheduler) scheduleLoop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Println("Scheduling loop terminated")
			return
		case <-ticker.C:
			if s.shouldRunJob() {
				job := models.NewScrapeJob()
				go func() {
					if _, err := s.executeJob(job); err != nil {
						log.Printf("Scheduled job failed: %v", err)
					}
				}()
			}
		}
	}
}

// shouldRunJob determines if a new job should be started
func (s *Scheduler) shouldRunJob() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Don't start if not running
	if !s.running {
		return false
	}

	// Don't start if there's already a job running
	if s.currentJob != nil && s.currentJob.IsActive() {
		return false
	}

	return true
}

// executeJob executes a scraping job
func (s *Scheduler) executeJob(job *models.ScrapeJob) (*models.ScrapeJob, error) {
	s.mu.Lock()
	s.currentJob = job
	s.mu.Unlock()

	// Start the job
	job.Start()
	log.Printf("Starting scraping job: %s", job.ID)

	// Call start callback
	if callback, exists := s.jobCallbacks["start"]; exists {
		callback(job)
	}

	// Execute the actual scraping
	err := s.scraper.ScrapeAll(func(progress ScrapeProgress) {
		s.updateJobProgress(job, progress)
	})

	// Complete or fail the job
	if err != nil {
		job.Fail(err.Error())
		log.Printf("Job failed: %s - %v", job.ID, err)
		
		// Call fail callback
		if callback, exists := s.jobCallbacks["fail"]; exists {
			callback(job)
		}
	} else {
		job.Complete()
		log.Printf("Job completed successfully: %s", job.ID)
		
		// Call complete callback
		if callback, exists := s.jobCallbacks["complete"]; exists {
			callback(job)
		}
	}

	// Add to history
	s.addToHistory(job)

	s.mu.Lock()
	s.currentJob = nil
	s.mu.Unlock()

	return job, err
}

// updateJobProgress updates job progress based on scraping progress
func (s *Scheduler) updateJobProgress(job *models.ScrapeJob, progress ScrapeProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job.CurrentPage = progress.CurrentPage
	job.TotalPages = progress.TotalPages
	job.ItemsScraped = progress.ItemsProcessed
	job.ItemsAdded = progress.ItemsAdded
	job.ItemsUpdated = progress.ItemsUpdated
	job.UpdatedAt = time.Now()
}

// addToHistory adds a completed job to the history
func (s *Scheduler) addToHistory(job *models.ScrapeJob) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobHistory = append(s.jobHistory, job)

	// Trim history if it gets too long
	maxHistory := 50
	if len(s.jobHistory) > maxHistory {
		s.jobHistory = s.jobHistory[len(s.jobHistory)-maxHistory:]
	}
}

// ScrapeProgress represents the progress of a scraping operation
type ScrapeProgress struct {
	CurrentPage     int
	TotalPages      int
	ItemsProcessed  int
	ItemsAdded      int
	ItemsUpdated    int
	StartTime       time.Time
	EstimatedRemaining time.Duration
}

// ProgressPercentage returns the progress as a percentage
func (sp *ScrapeProgress) ProgressPercentage() float64 {
	if sp.TotalPages == 0 {
		return 0
	}
	return float64(sp.CurrentPage) / float64(sp.TotalPages) * 100
}

// ItemsPerSecond returns the processing rate
func (sp *ScrapeProgress) ItemsPerSecond() float64 {
	elapsed := time.Since(sp.StartTime).Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(sp.ItemsProcessed) / elapsed
} 