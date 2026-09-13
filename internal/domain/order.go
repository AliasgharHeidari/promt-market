package domain

import "time"

// Order statuses. Kept as strings (not typed) so JSON and DB storage stay
// simple and the DB can be inspected without a migration for new values.
const (
	OrderStatusPending  = "pending"
	OrderStatusPaid     = "paid"
	OrderStatusFailed   = "failed"
	OrderStatusRefunded = "refunded"
)

// Order is one prompt purchased by one buyer. A user's checkout may produce
// multiple orders (one per cart item) that all link to the same payment.
type Order struct {
	ID string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	// BuyerID/PromptID are explicitly typed uuid to match users.id /
	// prompts.id. Previously these relied on GORM inferring the type from
	// the Buyer/Prompt relationship fields below at AutoMigrate time —
	// that inference happens to work today, but it's an implicit
	// dependency: removing/renaming the relation field would silently
	// flip the column back to text on a fresh migration. Being explicit
	// removes that fragility.
	BuyerID  string `json:"buyer_id" gorm:"not null;type:uuid;index"`
	PromptID string `json:"prompt_id" gorm:"not null;type:uuid;index"`

	// Snapshot fields. Captured at checkout time so a later price/title edit
	// by the seller doesn't change what the buyer paid or sees in history.
	Amount      int64  `json:"amount" gorm:"not null"`
	Commission  int64  `json:"commission"`
	PromptTitle string `json:"prompt_title" gorm:"size:255"`
	PromptCover string `json:"prompt_cover" gorm:"size:500"`

	Status string `json:"status" gorm:"default:'pending';index;size:50"`

	// PaymentID links the order to the payment attempt that created it.
	// FIXED: was previously `size:255` (varchar), which does not match
	// payments.id (uuid). Order has no `Payment Payment` relation field,
	// so GORM had no relation to infer the type from and defaulted to a
	// plain varchar column — the same class of bug as PromptView.PromptID.
	// This was latent (no query currently joins orders.payment_id against
	// payments.id) but would fail the moment one did. Nullable via
	// omitempty-style zero value ("") for legacy rows created before this
	// field existed; new orders always set it.
	PaymentID string `json:"payment_id" gorm:"type:uuid;index"`

	CreatedAt time.Time  `json:"created_at"`
	PaidAt    *time.Time `json:"paid_at"`

	// Relationships. Prompt is soft-deleted on removal, so historical orders
	// keep a resolvable FK — see PromptRepository.Delete for details.
	Buyer  User   `json:"buyer" gorm:"foreignKey:BuyerID"`
	Prompt Prompt `json:"prompt" gorm:"foreignKey:PromptID"`
}