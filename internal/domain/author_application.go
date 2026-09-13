package domain

import "time"

// AuthorApplication represents a user's request to become a verified
// prompt author. Created only after the user's phone number has been
// verified (see User.PhoneVerified). Holds the KYC data (national ID +
// ID card/birth-certificate photo) and expertise submitted by the user,
// and is reviewed by an admin before the user's Role is promoted.
type AuthorApplication struct {
	ID     string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID string `json:"user_id" gorm:"not null;index;type:uuid"`

	Expertise  string `json:"expertise" gorm:"size:150;not null"`
	NationalID string `json:"national_id" gorm:"size:20;not null"`

	// Server-local path to the uploaded ID card / birth certificate image.
	// Not exposed as a public URL directly; served through an authenticated
	// admin-only endpoint (see AdminHandler.GetAuthorApplicationDocument).
	IDDocumentPath string `json:"-" gorm:"column:id_document_path;size:500;not null"`

	// pending | approved | rejected
	Status string `json:"status" gorm:"size:20;not null;default:'pending';index"`

	RejectionReason string `json:"rejection_reason,omitempty" gorm:"type:text"`

	ReviewedBy *string    `json:"reviewed_by,omitempty" gorm:"type:uuid"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AuthorApplication) TableName() string {
	return "author_applications"
}