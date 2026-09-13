package domain

// CreateReviewRequest is the payload for POST /prompts/:id/reviews.
// Rating must be 1–5; Comment is capped at 250 characters.
type CreateReviewRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

// RejectReviewRequest carries the optional reason text when an admin
// rejects a review.
type RejectReviewRequest struct {
	Reason string `json:"reason"`
}