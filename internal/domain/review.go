package domain

import (
	"time"
)

// Review statuses. A review starts as pending and only becomes visible
// to the public once an admin approves it.
const (
	ReviewStatusPending  = "pending"
	ReviewStatusApproved = "approved"
	ReviewStatusRejected = "rejected"
)

type Review struct {
	ID string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	// UserID/PromptID explicitly typed uuid. These were likely already
	// correct via GORM inferring the type from the User/Prompt relation
	// fields below, but that's an implicit dependency — made explicit here
	// so it can't silently regress if the relation fields are ever removed.
	UserID   string `json:"user_id" gorm:"not null;type:uuid;index"`
	PromptID string `json:"prompt_id" gorm:"not null;type:uuid;index"`

	// OrderID links the review to a specific purchase. Set for paid prompts.
	// Empty for free prompts, which don't require an order.
	//
	// FIXED: this struct has no `Order Order` relation field, so GORM had
	// nothing to infer the type from and this previously defaulted to
	// text/varchar — a mismatch against orders.id (uuid). Latent (nothing
	// currently joins reviews.order_id against orders.id) but the same bug
	// class as PromptView.PromptID. Nullable: stays empty for free-prompt
	// reviews that have no associated order.
	OrderID string `json:"order_id" gorm:"type:uuid;index"`

	Rating  int    `json:"rating" gorm:"not null;check:rating >= 1 AND rating <= 5"`
	Comment string `json:"comment" gorm:"type:text;not null"`

	Status string `json:"status" gorm:"default:'pending';index;size:20"`

	// RejectionReason is filled by an admin when rejecting a review.
	// Shown to the review's author in their own view.
	RejectionReason string `json:"rejection_reason" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships (populated by Preload).
	User   User   `json:"user" gorm:"foreignKey:UserID"`
	Prompt Prompt `json:"prompt" gorm:"foreignKey:PromptID"`
}

// ReviewWithUser is a DTO for public review listings that only exposes
// safe user fields (no email, no password hash, etc.).
type ReviewWithUser struct {
	ID        string    `json:"id"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		FullName string `json:"full_name"`
	} `json:"user"`
}