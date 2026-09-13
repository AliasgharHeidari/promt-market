package domain

type CreatePromptRequest struct {
	Title         string   `json:"title" validate:"required,min=5,max=255"`
	Description   string   `json:"description" validate:"required,min=20"`
	Category      string   `json:"category" validate:"required"`
	SubCategory   string   `json:"sub_category"`
	Tags          []string `json:"tags"`
	Content       string   `json:"content" validate:"required,min=10"`
	DemoOutput    string   `json:"demo_output"`
	Instructions  string   `json:"instructions"`
	CoverImage    string   `json:"cover_image"`
	Images        []string `json:"images"`
	Price         int64    `json:"price" validate:"required,min=1000"`
	DiscountPrice int64    `json:"discount_price"`
	Difficulty    string   `json:"difficulty" validate:"oneof=beginner intermediate advanced"`
	Language      string   `json:"language" validate:"oneof=english persian both"`
}

type UpdatePromptRequest struct {
	Title         *string   `json:"title"`
	Description   *string   `json:"description"`
	Category      *string   `json:"category"`
	SubCategory   *string   `json:"sub_category"`
	Tags          []string  `json:"tags"`
	Content       *string   `json:"content"`
	DemoOutput    *string   `json:"demo_output"`
	Instructions  *string   `json:"instructions"`
	CoverImage    *string   `json:"cover_image"`
	Images        []string  `json:"images"`
	Price         *int64    `json:"price"`
	DiscountPrice *int64    `json:"discount_price"`
	Difficulty    *string   `json:"difficulty"`
	Language      *string   `json:"language"`
}

type PromptFilter struct {
	Category    string   `query:"category"`
	SubCategory string   `query:"sub_category"`
	Tags        []string `query:"tags"`
	MinPrice    int64    `query:"min_price"`
	MaxPrice    int64    `query:"max_price"`
	Difficulty  string   `query:"difficulty" validate:"omitempty,oneof=beginner intermediate advanced"`
	Language    string   `query:"language" validate:"omitempty,oneof=english persian both"`
	MinRating   float64  `query:"min_rating" validate:"omitempty,min=0,max=5"`
	Search      string   `query:"search"`
	SortBy      string   `query:"sort_by" validate:"omitempty,oneof=price created_at rating sales_count views"`
	SortOrder   string   `query:"sort_order" validate:"omitempty,oneof=asc desc"`
	Status      string   `query:"status" validate:"omitempty,oneof=pending approved rejected draft"`
	Page        int      `query:"page" validate:"min=1"`
	Limit       int      `query:"limit" validate:"min=1,max=100"`
}

// AdminUpdatePromptRequest is the payload for admins to edit any prompt
// while it is under review. All fields are optional (pointer-based) so the
// admin can patch only what they need to correct.
type AdminUpdatePromptRequest struct {
	Title         *string   `json:"title" validate:"omitempty,min=5,max=255"`
	Description   *string   `json:"description" validate:"omitempty,min=20"`
	Category      *string   `json:"category"`
	SubCategory   *string   `json:"sub_category"`
	Tags          []string  `json:"tags"`
	Content       *string   `json:"content" validate:"omitempty,min=10"`
	DemoOutput    *string   `json:"demo_output"`
	Instructions  *string   `json:"instructions"`
	CoverImage    *string   `json:"cover_image"`
	Images        []string  `json:"images"`
	Price         *int64    `json:"price" validate:"omitempty,min=0"`
	DiscountPrice *int64    `json:"discount_price"`
	Difficulty    *string   `json:"difficulty" validate:"omitempty,oneof=beginner intermediate advanced"`
	Language      *string   `json:"language" validate:"omitempty,oneof=english persian both"`
	Status        *string   `json:"status" validate:"omitempty,oneof=pending approved rejected"`
}

// RejectPromptRequest carries the optional reason text when an admin
// rejects a prompt. Reason is stored on Prompt.RejectionNote and shown
// back to the seller.
type RejectPromptRequest struct {
	Note string `json:"note"`
}

// RemoveImageRequest identifies which image to strip from a prompt's
// gallery. Index is zero-based and refers to the position inside
// Prompt.Images at the time the admin submits the form.
type RemoveImageRequest struct {
	Index int `json:"index" validate:"min=0"`
}

// ─────────────────────────────────────────────────────────
//  Phase 2: content gating
// ─────────────────────────────────────────────────────────

// PromptResponse wraps a Prompt with viewer-specific metadata. The handler
// builds one of these per request so a single prompt can be rendered
// differently for buyers vs. everyone else without changing the DB row.
//
// When HasAccess is false, Content / Instructions / DemoOutput are cleared
// by the service — the fields are emitted as empty strings, and the
// frontend uses HasAccess to decide whether to render them or a paywall.
type PromptResponse struct {
	HasAccess bool `json:"has_access"`

	// AccessReason explains why HasAccess is true. Useful for the frontend
	// to render a different banner ("you own this", "this prompt is free",
	// "you are the author", etc.) without extra queries.
	//
	// Values: "purchased" | "free" | "owner" | "admin" | ""
	AccessReason string `json:"access_reason,omitempty"`

	Prompt
}

// ─────────────────────────────────────────────────────────
//  Filter helpers
// ─────────────────────────────────────────────────────────

func (f *PromptFilter) SetDefaults() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.SortBy == "" {
		f.SortBy = "created_at"
	}
	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}
	if f.Status == "" {
		f.Status = "approved"
	}
}

func (f *PromptFilter) GetOffset() int {
	return (f.Page - 1) * f.Limit
}