package domain

import (
	"time"
)

type User struct {
	ID           string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Email        string    `json:"email" gorm:"unique;not null;size:255"`
	PasswordHash string    `json:"-" gorm:"not null;size:255"`
	FullName     string    `json:"full_name" gorm:"size:100"`
	Expertise    string    `json:"expertise" gorm:"size:100"`
	Bio          string    `json:"bio" gorm:"type:text"`
	Rating       float64   `json:"rating" gorm:"default:0"`
	TotalSales   int       `json:"total_sales" gorm:"default:0"`
	WalletBalance int64    `json:"wallet_balance" gorm:"default:0"`
	Role         string    `json:"role" gorm:"default:'user';size:20"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}