package domain

import (
	"time"
)

type Prompt struct {
	ID           string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SellerID     string    `json:"seller_id" gorm:"not null;index"`
	
	// Basic info
	Title        string    `json:"title" gorm:"not null;size:255"`
	Description  string    `json:"description" gorm:"type:text"`
	Category     string    `json:"category" gorm:"index;size:100"`
	SubCategory  string    `json:"sub_category" gorm:"size:100"`
	Tags         []string  `json:"tags" gorm:"type:text[]"`
	
	// Content
	Content      string    `json:"content" gorm:"type:text;not null"`
	DemoOutput   string    `json:"demo_output" gorm:"type:text"`
	Instructions string    `json:"instructions" gorm:"type:text"`
	
	// Media
	CoverImage   string    `json:"cover_image" gorm:"size:500"`
	Images       []string  `json:"images" gorm:"type:text[]"`
	
	// Pricing
	Price        int64     `json:"price" gorm:"not null;index"`
	DiscountPrice int64    `json:"discount_price"`
	
	// Status & stats
	Status       string    `json:"status" gorm:"default:'pending';index;size:20"`
	Views        int       `json:"views" gorm:"default:0"`
	SalesCount   int       `json:"sales_count" gorm:"default:0"`
	Rating       float64   `json:"rating" gorm:"default:0"`
	ReviewCount  int       `json:"review_count" gorm:"default:0"`
	
	// Additional
	Difficulty   string    `json:"difficulty" gorm:"size:20"`
	Language     string    `json:"language" gorm:"size:20;default:'english'"`
	Version      string    `json:"version" gorm:"size:10;default:'1.0.0'"`
	
	// SEO
	Slug         string    `json:"slug" gorm:"unique;index;size:255"`
	MetaTitle    string    `json:"meta_title" gorm:"size:60"`
	MetaDescription string `json:"meta_description" gorm:"size:160"`
	MetaKeywords []string  `json:"meta_keywords" gorm:"type:text[]"`
	
	// Timestamps
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	PublishedAt  *time.Time `json:"published_at"`
	DeletedAt    *time.Time `json:"-" gorm:"index"`
}