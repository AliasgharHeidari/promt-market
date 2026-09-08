package domain

import (
	"time"
)

type Review struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	OrderID   string    `json:"order_id" gorm:"not null;index"`
	UserID    string    `json:"user_id" gorm:"not null;index"`
	PromptID  string    `json:"prompt_id" gorm:"not null;index"`
	Rating    int       `json:"rating" gorm:"check:rating >= 1 AND rating <= 5"`
	Comment   string    `json:"comment" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
}