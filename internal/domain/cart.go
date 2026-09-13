package domain

import "time"

// CartItem represents one prompt in a user's shopping cart.
//
// Design notes:
//   - No Quantity field: a prompt can only be purchased once per user, so the
//     cart is a *set* of prompts, not a list of quantities.
//   - (user_id, prompt_id) is UNIQUE at the DB level to prevent duplicate
//     rows even under concurrent requests.
//   - Deletes on user_id are hard-cascaded by GORM. Deletes on prompt_id are
//     soft (Prompt uses a DeletedAt column, not a real DELETE), so cart items
//     whose prompt was later removed/paused remain in the cart and are
//     surfaced to the UI as "unavailable".
type CartItem struct {
	ID string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	// UserID is explicitly typed uuid to match users.id. There is no
	// `User User` relation field on this struct for GORM to infer the type
	// from at AutoMigrate time (unlike PromptID below, which is covered by
	// the Prompt relation), so without this explicit tag the column would
	// silently default to text/varchar — the same class of bug that broke
	// the author dashboard views chart (see PromptView.PromptID).
	UserID   string    `json:"user_id" gorm:"not null;type:uuid;index;uniqueIndex:idx_cart_user_prompt"`
	PromptID string    `json:"prompt_id" gorm:"not null;type:uuid;index;uniqueIndex:idx_cart_user_prompt"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships (populated via Preload).
	Prompt Prompt `json:"prompt" gorm:"foreignKey:PromptID"`
}

// CartItemView is the shape returned to the client. It deliberately exposes
// only the prompt summary — never content/instructions — because the user
// hasn't purchased it yet.
type CartItemView struct {
	PromptID         string    `json:"prompt_id"`
	Title            string    `json:"title"`
	CoverImage       string    `json:"cover_image"`
	SellerName       string    `json:"seller_name"`
	Category         string    `json:"category"`
	Price            int64     `json:"price"`
	DiscountPrice    int64     `json:"discount_price"`
	FinalPrice       int64     `json:"final_price"`
	Unavailable      bool      `json:"unavailable"`
	UnavailableMsg   string    `json:"unavailable_msg,omitempty"`
	AlreadyPurchased bool      `json:"already_purchased"`
	AddedAt          time.Time `json:"added_at"`
}

// CartResponse is the full payload for GET /cart.
type CartResponse struct {
	Data        []CartItemView `json:"data"`
	TotalAmount int64          `json:"total_amount"`
	Count       int            `json:"count"`
}