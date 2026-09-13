package domain

import "time"

// Payment statuses.
const (
	PaymentStatusPending  = "pending"
	PaymentStatusPaid     = "paid"
	PaymentStatusFailed   = "failed"
	PaymentStatusCanceled = "canceled"
)

// Payment represents a single payment attempt by a user. One payment can
// cover multiple orders (the user may have several prompts in their cart).
//
// Design:
//   - authority is issued by the gateway (or Mock) at InitPayment time and
//     is unique per attempt. It is the idempotency key for the callback.
//   - ref_id is the gateway's tracking code after a successful verify.
//   - amount is captured at checkout time, not recomputed later, so the
//     user pays exactly what the cart showed them.
type Payment struct {
	ID     string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID string `json:"user_id" gorm:"not null;index"`

	Amount int64 `json:"amount" gorm:"not null"`

	// Authority is issued by the provider; globally unique per attempt.
	// Used as the lookup key when the user returns from the gateway.
	Authority string `json:"authority" gorm:"uniqueIndex;size:255"`

	// RefID is the provider's reference after a successful verify.
	RefID string `json:"ref_id" gorm:"size:255"`

	Status string `json:"status" gorm:"default:'pending';index;size:20"`

	// Provider records which gateway was used (e.g. "mock", "zarinpal").
	// Kept on the row so a historical payment is still interpretable after
	// the active provider is switched in config.
	Provider string `json:"provider" gorm:"size:50"`

	// FailureReason holds the provider's error message if the payment failed
	// or was canceled. Empty for successful payments.
	FailureReason string `json:"failure_reason" gorm:"type:text"`

	CreatedAt time.Time  `json:"created_at"`
	PaidAt    *time.Time `json:"paid_at"`

	// Relationships
	User User `json:"user" gorm:"foreignKey:UserID"`
}