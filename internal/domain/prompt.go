package domain

import (
	"time"

	"github.com/lib/pq"
)

type Prompt struct {
	ID       string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SellerID string `json:"seller_id" gorm:"not null;index"`

	Seller User `json:"seller" gorm:"foreignKey:SellerID"`

	Title       string         `json:"title" gorm:"not null;size:255"`
	Description string         `json:"description" gorm:"type:text"`
	Category    string         `json:"category" gorm:"index;size:100"`
	SubCategory string         `json:"sub_category" gorm:"size:100"`
	Tags        pq.StringArray `json:"tags" gorm:"type:text[]"`

	Content      string `json:"content" gorm:"type:text;not null"`
	DemoOutput   string `json:"demo_output" gorm:"type:text"`
	Instructions string `json:"instructions" gorm:"type:text"`

	CoverImage string         `json:"cover_image" gorm:"size:500"`
	Images     pq.StringArray `json:"images" gorm:"type:text[]"`

	Price         int64 `json:"price" gorm:"not null;index"`
	DiscountPrice int64 `json:"discount_price"`

	Status      string  `json:"status" gorm:"default:'pending';index;size:20"`
	Views       int     `json:"views" gorm:"default:0"`
	SalesCount  int     `json:"sales_count" gorm:"default:0"`
	Rating      float64 `json:"rating" gorm:"default:0"`
	ReviewCount int     `json:"review_count" gorm:"default:0"`

	// IsPaused temporarily hides the prompt from public listing without
	// changing its Status. Owner can toggle this at any time.
	IsPaused bool `json:"is_paused" gorm:"default:false;index"`

	// HasPendingEdit is true while a PromptEditProposal for this prompt is
	// awaiting admin review. Used to show a badge in the author dashboard.
	HasPendingEdit bool `json:"has_pending_edit" gorm:"default:false;index"`

	RejectionNote string `json:"rejection_note" gorm:"type:text"`

	Difficulty string `json:"difficulty" gorm:"size:20"`
	Language   string `json:"language" gorm:"size:20;default:'english'"`
	Version    string `json:"version" gorm:"size:10;default:'1.0.0'"`

	Slug            string         `json:"slug" gorm:"unique;index;size:255"`
	MetaTitle       string         `json:"meta_title" gorm:"size:60"`
	MetaDescription string         `json:"meta_description" gorm:"size:160"`
	MetaKeywords    pq.StringArray `json:"meta_keywords" gorm:"type:text[]"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	PublishedAt *time.Time `json:"published_at"`
	DeletedAt   *time.Time `json:"-" gorm:"index"`
}