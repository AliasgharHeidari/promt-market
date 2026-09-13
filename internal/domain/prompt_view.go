package domain

import "time"

// PromptView is one row per (prompt, day). Views are aggregated with an
// UPSERT so a busy prompt produces at most one row per day instead of one
// row per hit. This makes monthly/yearly aggregations cheap.
type PromptView struct {
	ID string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	// PromptID must be explicitly typed uuid to match prompts.id. Without
	// a `Prompt Prompt gorm:"foreignKey:PromptID"` relationship field on
	// this struct, GORM's AutoMigrate has no way to infer the column type
	// from the referenced table and defaults a plain `string` field to
	// text/varchar — which then fails with "operator does not exist:
	// uuid = text" the first time this column is compared/joined against
	// prompts.id. This was the exact cause of the author-dashboard 500s.
	PromptID string `json:"prompt_id" gorm:"not null;type:uuid;index:idx_prompt_date,unique"`

	Date  time.Time `json:"date" gorm:"type:date;not null;index:idx_prompt_date,unique"`
	Count int       `json:"count" gorm:"default:0"`
}