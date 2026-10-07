package postgres

import (
	"context"
	"time"

	"gorm.io/gorm/clause"
)

// SettingsRepository persists runtime setting overrides.
type SettingsRepository struct{ db *DB }

func NewSettingsRepository(db *DB) *SettingsRepository { return &SettingsRepository{db: db} }

func (r *SettingsRepository) All(ctx context.Context) (map[string]string, error) {
	var models []settingModel
	if err := r.db.session(ctx).Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make(map[string]string, len(models))
	for _, m := range models {
		out[m.Key] = m.Value
	}
	return out, nil
}

func (r *SettingsRepository) Upsert(ctx context.Context, values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	now := time.Now().UTC()
	rows := make([]settingModel, 0, len(values))
	for k, v := range values {
		rows = append(rows, settingModel{Key: k, Value: v, UpdatedAt: now})
	}
	return translate(r.db.session(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&rows).Error)
}
