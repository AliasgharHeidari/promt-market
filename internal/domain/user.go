package domain

import (
	"time"
)

// Role values:
//   - "user"          : normal buyer, cannot create prompts
//   - "prompt_author"  : verified seller, can create prompts
//   - "admin"          : admin panel access
type User struct {
	ID            string  `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Email         string  `json:"email" gorm:"unique;not null;size:255"`
	PasswordHash  string  `json:"-" gorm:"not null;size:255"`
	FullName      string  `json:"full_name" gorm:"size:100"`
	Expertise     string  `json:"expertise" gorm:"size:100"`
	Bio           string  `json:"bio" gorm:"type:text"`
	Rating        float64 `json:"rating" gorm:"default:0"`
	TotalSales    int     `json:"total_sales" gorm:"default:0"`
	WalletBalance int64   `json:"wallet_balance" gorm:"default:0"`
	Role          string  `json:"role" gorm:"default:'user';size:20"`
	IsActive      bool    `json:"is_active" gorm:"default:true"`
	IsVerified    bool    `json:"is_verified" gorm:"default:false"` // email verified

	Phone         string `json:"phone" gorm:"size:20"`
	PhoneVerified bool   `json:"phone_verified" gorm:"default:false"`

	// National ID captured at author-application time; kept on the user
	// record too (in addition to AuthorApplication) so it's available
	// without a join once the user becomes a prompt_author.
	NationalID string `json:"national_id,omitempty" gorm:"size:20"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsPromptAuthor reports whether the user is allowed to create prompts.
func (u *User) IsPromptAuthor() bool {
	return u.Role == "prompt_author" || u.Role == "admin"
}