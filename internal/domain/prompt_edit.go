package domain

import (
	"time"

	"gorm.io/datatypes"
)

const (
	EditProposalPending  = "pending"
	EditProposalApproved = "approved"
	EditProposalRejected = "rejected"
)

// PromptEditProposal stores a proposed set of changes for an existing prompt.
// The prompt itself is untouched until an admin approves the proposal; this
// keeps an approved prompt visible to buyers while edits are pending review.
//
// Changes are stored as a sparse JSON object — only the fields the author
// actually modified are present. On approval, those fields are applied to
// the prompt via a partial update.
type PromptEditProposal struct {
	ID string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	// PromptID/SellerID explicitly typed uuid. Likely already correct via
	// GORM inferring the type from the Prompt/Seller relation fields below,
	// but made explicit here so it doesn't silently depend on those
	// relation fields staying present (see PromptView.PromptID for the bug
	// this class of implicit typing caused).
	PromptID string `json:"prompt_id" gorm:"not null;type:uuid;index"`
	SellerID string `json:"seller_id" gorm:"not null;type:uuid;index"`

	Changes datatypes.JSON `json:"changes" gorm:"type:jsonb;not null"`

	Status string `json:"status" gorm:"default:'pending';index;size:20"`

	// AdminNote is an optional message from the reviewer, shown to the
	// author if the proposal is rejected.
	AdminNote string `json:"admin_note" gorm:"type:text"`

	ReviewedAt *time.Time `json:"reviewed_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Prompt Prompt `json:"prompt" gorm:"foreignKey:PromptID"`
	Seller User   `json:"seller" gorm:"foreignKey:SellerID"`
}