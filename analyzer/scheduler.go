package analyzer

import (
	"fmt"
	"sync"

	"nofx/chaos"
	"nofx/kernel"
	"nofx/logger"
	"nofx/mcp"
	"nofx/store"

	"github.com/robfig/cron/v3"
)

// Scheduler manages analyst execution schedules
type Scheduler struct {
	store          *store.Store
	analyzerEngine *chaos.AnalyzerEngine
	cron           *cron.Cron
	jobs           map[string]cron.EntryID
	mu             sync.Mutex
	mcpClient      mcp.AIClient // We need this to run analysis
}

// NewScheduler creates a new scheduler
func NewScheduler(s *store.Store, engine *chaos.AnalyzerEngine, mcpClient mcp.AIClient) *Scheduler {
	return &Scheduler{
		store:          s,
		analyzerEngine: engine,
		cron:           cron.New(cron.WithSeconds()), // Enable seconds for precision if needed
		jobs:           make(map[string]cron.EntryID),
		mcpClient:      mcpClient,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.cron.Start()
	logger.Infof("🕒 Analyst Scheduler started")
	s.SyncJobs()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
	logger.Infof("🛑 Analyst Scheduler stopped")
}

// SyncJobs synchronizes scheduled jobs with database state
func (s *Scheduler) SyncJobs() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	profiles, err := s.store.AnalystProfile().GetAll()
	if err != nil {
		return fmt.Errorf("failed to fetch analyst profiles: %w", err)
	}

	activeJobIDs := make(map[string]bool)

	for _, profile := range profiles {
		// Skip if disabled or no schedule
		if !profile.IsEnabled || profile.CronSchedule == "" {
			if jobID, exists := s.jobs[profile.ID]; exists {
				s.cron.Remove(jobID)
				delete(s.jobs, profile.ID)
				logger.Infof("Removed scheduled job for analyst %s", profile.Name)
			}
			continue
		}

		activeJobIDs[profile.ID] = true

		// Check if already running (optimization: check schedule change if needed)
		if _, exists := s.jobs[profile.ID]; exists {
			// For simplicity, we don't check if schedule changed, just keep it.
			// To support schedule updates, we should compare and recreate if changed.
			// Here assuming SyncJobs is called after updates or periodically.
			// Let's remove and re-add to be safe and simple for now.
			s.cron.Remove(s.jobs[profile.ID])
		}

		// Add job
		// We need to capture profile content for the closure
		p := profile
		jobID, err := s.cron.AddFunc(p.CronSchedule, func() {
			s.runJob(p)
		})

		if err != nil {
			logger.Errorf("Failed to schedule analyst %s with schedule %s: %v", p.Name, p.CronSchedule, err)
			continue
		}

		s.jobs[p.ID] = jobID
		logger.Infof("Scheduled analyst %s with schedule %s", p.Name, p.CronSchedule)
	}

	// Cleanup removed profiles
	for id, jobID := range s.jobs {
		if !activeJobIDs[id] {
			s.cron.Remove(jobID)
			delete(s.jobs, id)
			logger.Infof("Removed obsolete job for analyst ID %s", id)
		}
	}

	return nil
}

func (s *Scheduler) runJob(profile *store.AnalystProfile) {
	logger.Infof("🚀 Starting scheduled analysis for %s", profile.Name)

	// Build context (Simplified)
	// In a real scenario, we need to construct a proper context similar to AutoTrader
	// For now, we create a minimal context.
	// We might need to inject the StrategyConfig from somewhere if it's not in AnalyzerEngine
	// AnalyzerEngine has config, but we need to create a kernel.Context

	// TODO: Retrieve real account state if needed, or just use empty for pure market analysis
	// Chaos mode relies on account balance for risk control, but Analyzer might just need market data.
	// However, BuildUserPrompt uses account info.
	// Let's create a dummy context for now, assuming fetchMarketData will populate it.

	ctx := &kernel.Context{
		// Account: ... (Optional for pure analysis?)
		// Positions: ... (Need to fetch from DB if we want to analyze current positions)
		CandidateCoins: []kernel.CandidateCoin{}, // AnalyzerEngine logic needs to be robust to empty
	}

	// Fetch positions from DB to populate context
	// This ensures the analyst sees current holdings
	// We need a way to get the default trader ID or iterate all.
	// For simplicity, we assume single user/trader mode or pick the first one.
	// Or we can load from config.

	// Converting store.AnalystProfile to chaos.AnalystProfile
	chaosProfile := chaos.AnalystProfile{
		ID:           profile.ID,
		Name:         profile.Name,
		SystemPrompt: profile.SystemPrompt,
		ModelID:      profile.ModelID,
	}

	// Execute
	// Use a timeout context for the job
	// jobCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	// defer cancel()

	session, err := s.analyzerEngine.RunSentinel(ctx, s.mcpClient, chaosProfile)
	if err != nil {
		logger.Errorf("❌ Analysis job failed for %s: %v", profile.Name, err)
		return
	}

	logger.Infof("✅ Analysis job completed for %s. Session ID: %s", profile.Name, session.SessionID)
}

// TriggerManually triggers an analyst immediately
func (s *Scheduler) TriggerManually(profileID string) error {
	profile, err := s.store.AnalystProfile().GetByID(profileID)
	if err != nil {
		return err
	}

	go s.runJob(profile)
	return nil
}
