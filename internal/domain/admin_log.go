package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type AdminLog struct {
	ID         string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AdminID    string    `json:"admin_id" gorm:"not null;index"`
	Action     string    `json:"action" gorm:"size:100;not null"`
	TargetType string    `json:"target_type" gorm:"size:50"`
	TargetID   string    `json:"target_id"`
	Details    JSONMap   `json:"details" gorm:"type:jsonb"`
	IP         string    `json:"ip" gorm:"size:45"`
	UserAgent  string    `json:"user_agent" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
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