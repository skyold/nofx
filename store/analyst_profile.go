package store

import (
	"time"

	"gorm.io/gorm"
)

// AnalystProfileDB GORM model for analyst_profiles table
type AnalystProfileDB struct {
	ID           string    `gorm:"primaryKey;column:id;size:64"`
	Name         string    `gorm:"column:name;size:128;not null"`
	SystemPrompt string    `gorm:"column:system_prompt;type:text"`
	ModelID      string    `gorm:"column:model_id;size:64"`
	CronSchedule string    `gorm:"column:cron_schedule;size:64"` // e.g., "*/15 * * * *"
	IsEnabled    bool      `gorm:"column:is_enabled;default:true"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null"`
}

func (AnalystProfileDB) TableName() string { return "analyst_profiles" }

// AnalystProfile external API struct
type AnalystProfile struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	SystemPrompt string    `json:"system_prompt"`
	ModelID      string    `json:"model_id"`
	CronSchedule string    `json:"cron_schedule"`
	IsEnabled    bool      `json:"is_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AnalystProfileStore analyst profile storage
type AnalystProfileStore struct {
	db *gorm.DB
}

// NewAnalystProfileStore creates new AnalystProfileStore
func NewAnalystProfileStore(db *gorm.DB) *AnalystProfileStore {
	return &AnalystProfileStore{db: db}
}

// initTables initializes analyst profile tables
func (s *AnalystProfileStore) initTables() error {
	return s.db.AutoMigrate(&AnalystProfileDB{})
}

// Create creates a new analyst profile
func (s *AnalystProfileStore) Create(profile *AnalystProfile) error {
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now().UTC()
	}
	profile.UpdatedAt = time.Now().UTC()

	dbProfile := &AnalystProfileDB{
		ID:           profile.ID,
		Name:         profile.Name,
		SystemPrompt: profile.SystemPrompt,
		ModelID:      profile.ModelID,
		CronSchedule: profile.CronSchedule,
		IsEnabled:    profile.IsEnabled,
		CreatedAt:    profile.CreatedAt,
		UpdatedAt:    profile.UpdatedAt,
	}

	return s.db.Create(dbProfile).Error
}

// Update updates an existing analyst profile
func (s *AnalystProfileStore) Update(profile *AnalystProfile) error {
	profile.UpdatedAt = time.Now().UTC()

	dbProfile := &AnalystProfileDB{
		ID:           profile.ID,
		Name:         profile.Name,
		SystemPrompt: profile.SystemPrompt,
		ModelID:      profile.ModelID,
		CronSchedule: profile.CronSchedule,
		IsEnabled:    profile.IsEnabled,
		UpdatedAt:    profile.UpdatedAt,
	}

	return s.db.Model(&AnalystProfileDB{}).Where("id = ?", profile.ID).Updates(dbProfile).Error
}

// GetByID gets an analyst profile by ID
func (s *AnalystProfileStore) GetByID(id string) (*AnalystProfile, error) {
	var dbProfile AnalystProfileDB
	if err := s.db.First(&dbProfile, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &AnalystProfile{
		ID:           dbProfile.ID,
		Name:         dbProfile.Name,
		SystemPrompt: dbProfile.SystemPrompt,
		ModelID:      dbProfile.ModelID,
		CronSchedule: dbProfile.CronSchedule,
		IsEnabled:    dbProfile.IsEnabled,
		CreatedAt:    dbProfile.CreatedAt,
		UpdatedAt:    dbProfile.UpdatedAt,
	}, nil
}

// GetAll gets all analyst profiles
func (s *AnalystProfileStore) GetAll() ([]*AnalystProfile, error) {
	var dbProfiles []AnalystProfileDB
	if err := s.db.Find(&dbProfiles).Error; err != nil {
		return nil, err
	}

	profiles := make([]*AnalystProfile, len(dbProfiles))
	for i, db := range dbProfiles {
		profiles[i] = &AnalystProfile{
			ID:           db.ID,
			Name:         db.Name,
			SystemPrompt: db.SystemPrompt,
			ModelID:      db.ModelID,
			CronSchedule: db.CronSchedule,
			IsEnabled:    db.IsEnabled,
			CreatedAt:    db.CreatedAt,
			UpdatedAt:    db.UpdatedAt,
		}
	}
	return profiles, nil
}

// Delete deletes an analyst profile by ID
func (s *AnalystProfileStore) Delete(id string) error {
	return s.db.Delete(&AnalystProfileDB{}, "id = ?", id).Error
}
