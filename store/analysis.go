package store

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// AnalysisStore analysis data storage
type AnalysisStore struct {
	db *gorm.DB
}

// AnalysisSessionDB GORM model for analysis_sessions table
type AnalysisSessionDB struct {
	SessionID             string    `gorm:"primaryKey;column:session_id;size:64"`
	TriggerType           string    `gorm:"column:trigger_type;size:32;not null"` // scheduled, manual, roundtable
	MarketContextSnapshot string    `gorm:"column:market_context_snapshot;type:text"`
	FinalSummary          string    `gorm:"column:final_summary;type:text"`
	CreatedAt             time.Time `gorm:"column:created_at;not null;index:idx_analysis_sessions_created_at,sort:desc"`
}

func (AnalysisSessionDB) TableName() string { return "analysis_sessions" }

// AnalysisRecordDB GORM model for analysis_records table
type AnalysisRecordDB struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	SessionID   string    `gorm:"column:session_id;size:64;not null;index:idx_analysis_records_session_id"`
	AnalystID   string    `gorm:"column:analyst_id;size:64;not null;index:idx_analysis_records_analyst_id"`
	Sentiment   string    `gorm:"column:sentiment;size:32"` // bullish, bearish, neutral
	Reasoning   string    `gorm:"column:reasoning;type:text"`
	Tags        string    `gorm:"column:tags;type:text"` // JSON array of tags
	RawResponse string    `gorm:"column:raw_response;type:text"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
}

func (AnalysisRecordDB) TableName() string { return "analysis_records" }

// AnalysisSession external API struct
type AnalysisSession struct {
	SessionID             string           `json:"session_id"`
	TriggerType           string           `json:"trigger_type"`
	MarketContextSnapshot string           `json:"market_context_snapshot"`
	FinalSummary          string           `json:"final_summary"`
	CreatedAt             time.Time        `json:"created_at"`
	Records               []AnalysisRecord `json:"records,omitempty"`
}

// AnalysisRecord external API struct
type AnalysisRecord struct {
	ID          int64     `json:"id"`
	SessionID   string    `json:"session_id"`
	AnalystID   string    `json:"analyst_id"`
	Sentiment   string    `json:"sentiment"`
	Reasoning   string    `json:"reasoning"`
	Tags        []string  `json:"tags"`
	RawResponse string    `json:"raw_response"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewAnalysisStore creates new AnalysisStore
func NewAnalysisStore(db *gorm.DB) *AnalysisStore {
	return &AnalysisStore{db: db}
}

// initTables initializes analysis tables
func (s *AnalysisStore) initTables() error {
	// For PostgreSQL with existing table, skip AutoMigrate check logic could be added here similar to DecisionStore
	// But standard AutoMigrate is usually safe
	return s.db.AutoMigrate(&AnalysisSessionDB{}, &AnalysisRecordDB{})
}

// CreateSession creates a new analysis session
func (s *AnalysisStore) CreateSession(session *AnalysisSession) error {
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}

	dbSession := &AnalysisSessionDB{
		SessionID:             session.SessionID,
		TriggerType:           session.TriggerType,
		MarketContextSnapshot: session.MarketContextSnapshot,
		FinalSummary:          session.FinalSummary,
		CreatedAt:             session.CreatedAt,
	}

	return s.db.Create(dbSession).Error
}

// CreateRecord creates a new analysis record
func (s *AnalysisStore) CreateRecord(record *AnalysisRecord) error {
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}

	tagsJSON, _ := json.Marshal(record.Tags)

	dbRecord := &AnalysisRecordDB{
		SessionID:   record.SessionID,
		AnalystID:   record.AnalystID,
		Sentiment:   record.Sentiment,
		Reasoning:   record.Reasoning,
		Tags:        string(tagsJSON),
		RawResponse: record.RawResponse,
		CreatedAt:   record.CreatedAt,
	}

	if err := s.db.Create(dbRecord).Error; err != nil {
		return err
	}
	record.ID = dbRecord.ID
	return nil
}

// GetSession gets a session by ID with its records
func (s *AnalysisStore) GetSession(sessionID string) (*AnalysisSession, error) {
	var dbSession AnalysisSessionDB
	if err := s.db.First(&dbSession, "session_id = ?", sessionID).Error; err != nil {
		return nil, err
	}

	var dbRecords []AnalysisRecordDB
	if err := s.db.Where("session_id = ?", sessionID).Find(&dbRecords).Error; err != nil {
		return nil, err
	}

	session := &AnalysisSession{
		SessionID:             dbSession.SessionID,
		TriggerType:           dbSession.TriggerType,
		MarketContextSnapshot: dbSession.MarketContextSnapshot,
		FinalSummary:          dbSession.FinalSummary,
		CreatedAt:             dbSession.CreatedAt,
		Records:               make([]AnalysisRecord, len(dbRecords)),
	}

	for i, r := range dbRecords {
		var tags []string
		json.Unmarshal([]byte(r.Tags), &tags)
		session.Records[i] = AnalysisRecord{
			ID:          r.ID,
			SessionID:   r.SessionID,
			AnalystID:   r.AnalystID,
			Sentiment:   r.Sentiment,
			Reasoning:   r.Reasoning,
			Tags:        tags,
			RawResponse: r.RawResponse,
			CreatedAt:   r.CreatedAt,
		}
	}

	return session, nil
}

// GetLatestSessions gets latest N sessions (without records content to be light)
func (s *AnalysisStore) GetLatestSessions(limit int) ([]*AnalysisSession, error) {
	var dbSessions []AnalysisSessionDB
	if err := s.db.Order("created_at DESC").Limit(limit).Find(&dbSessions).Error; err != nil {
		return nil, err
	}

	sessions := make([]*AnalysisSession, len(dbSessions))
	for i, db := range dbSessions {
		sessions[i] = &AnalysisSession{
			SessionID:             db.SessionID,
			TriggerType:           db.TriggerType,
			MarketContextSnapshot: db.MarketContextSnapshot,
			FinalSummary:          db.FinalSummary,
			CreatedAt:             db.CreatedAt,
		}
	}
	return sessions, nil
}
