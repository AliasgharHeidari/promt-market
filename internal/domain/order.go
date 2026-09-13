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
	ID       string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	BuyerID  string `json:"buyer_id" gorm:"not null;index"`
	PromptID string `json:"prompt_id" gorm:"not null;index"`

	// Snapshot fields. Captured at checkout time so a later price/title edit
	// by the seller doesn't change what the buyer paid or sees in history.
	Amount        int64  `json:"amount" gorm:"not null"`
	Commission    int64  `json:"commission"`
	PromptTitle   string `json:"prompt_title" gorm:"size:255"`
	PromptCover   string `json:"prompt_cover" gorm:"size:500"`

	Status string `json:"status" gorm:"default:'pending';index;size:50"`

	// PaymentID links the order to the payment attempt that created it.
	// Empty for legacy rows; new orders always set it.
	PaymentID string `json:"payment_id" gorm:"index;size:255"`

	CreatedAt time.Time  `json:"created_at"`
	PaidAt    *time.Time `json:"paid_at"`

	// Relationships. Prompt is soft-deleted on removal, so historical orders
	// keep a resolvable FK — see PromptRepository.Delete for details.
	Buyer  User   `json:"buyer" gorm:"foreignKey:BuyerID"`
	Prompt Prompt `json:"prompt" gorm:"foreignKey:PromptID"`
}