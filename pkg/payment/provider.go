package payment

import "context"

// InitResult is returned by a provider when a new payment is started.
// The frontend redirects the user to GatewayURL with Authority already
// embedded (or as a query param for gateways that need it separately).
type InitResult struct {
	Authority  string
	GatewayURL string
}

// VerifyResult is the outcome of verifying a payment after the user
// returns from the gateway.
type VerifyResult struct {
	Success  bool
	RefID    string
	ErrorMsg string
}

// Provider abstracts an online payment gateway. The Mock implementation is
// used in development and replaced by a real adapter (Zarinpal, IDPay, ...)
// in production. No code outside this package needs to know which one is
// active.
type Provider interface {
	// Name returns a short identifier stored on each Payment row so the
	// provider that created it can be identified later.
	Name() string

	// InitPayment asks the gateway to start a payment for the given amount
	// (in Toman) and returns an authority + the URL the user must visit.
	// description and callbackURL are forwarded to the gateway so it can
	// show a meaningful page and call us back when the user is done.
	InitPayment(ctx context.Context, amount int64, description, callbackURL string) (*InitResult, error)

	// VerifyPayment confirms that a payment identified by authority was
	// actually completed at the gateway. It must be called after the user
	// returns; trusting the callback alone is a security hole.
	VerifyPayment(ctx context.Context, authority string, amount int64) (*VerifyResult, error)
}