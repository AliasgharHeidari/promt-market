package domain

import (
	"time"
)

type Order struct {
	ID         string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	BuyerID    string    `json:"buyer_id" gorm:"not null;index"`
	PromptID   string    `json:"prompt_id" gorm:"not null;index"`
	Amount     int64     `json:"amount" gorm:"not null"`
	Commission int64     `json:"commission"`
	Status     string    `json:"status" gorm:"default:'pending';index;size:50"`
	PaymentID  string    `json:"payment_id" gorm:"index;size:255"`
	CreatedAt  time.Time `json:"created_at"`
}