package domain

import "time"

// PromptView is one row per (prompt, day). Views are aggregated with an
// UPSERT so a busy prompt produces at most one row per day instead of one
// row per hit. This makes monthly/yearly aggregations cheap.
type PromptView struct {
	ID       string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	PromptID string    `json:"prompt_id" gorm:"not null;index:idx_prompt_date,unique"`
	Date     time.Time `json:"date" gorm:"type:date;not null;index:idx_prompt_date,unique"`
	Count    int       `json:"count" gorm:"default:0"`
}