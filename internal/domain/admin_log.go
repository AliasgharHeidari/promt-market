package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type AdminLog struct {
	ID string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	// AdminID explicitly typed uuid to match users.id. This struct has no
	// `Admin User` relation field, so GORM had nothing to infer the type
	// from at AutoMigrate time and this previously defaulted to
	// text/varchar — the same latent bug class as PromptView.PromptID.
	// Nothing currently joins admin_logs.admin_id against users.id, so this
	// was silent, but would break the moment such a join was added.
	AdminID string `json:"admin_id" gorm:"not null;type:uuid;index"`

	Action     string `json:"action" gorm:"size:100;not null"`
	TargetType string `json:"target_type" gorm:"size:50"`

	// TargetID intentionally stays untyped (no type:uuid): depending on
	// TargetType this can reference rows in different tables (prompts,
	// orders, reviews, ...), so it can't be pinned to a single foreign-key
	// type. If admin_logs is ever joined against a specific target table,
	// cast explicitly at the query site (e.g. target_id::uuid) rather than
	// typing this column.
	TargetID string `json:"target_id"`

	Details   JSONMap   `json:"details" gorm:"type:jsonb"`
	IP        string    `json:"ip" gorm:"size:45"`
	UserAgent string    `json:"user_agent" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
}

type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal(map[string]interface{}{})
	}
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}