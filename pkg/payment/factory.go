package payment

import (
	"errors"
	"os"
)

// NewFromEnv returns the active payment provider based on the
// PAYMENT_PROVIDER env var. Only "mock" is implemented today; adding a
// real gateway means adding a case here plus a new file next to mock.go.
//
// Additional guard: in APP_ENV=production, "mock" is rejected so a
// misconfigured deploy cannot accidentally accept fake payments.
func NewFromEnv() (Provider, error) {
	name := os.Getenv("PAYMENT_PROVIDER")
	if name == "" {
		name = "mock"
	}

	env := os.Getenv("APP_ENV")

	switch name {
	case "mock":
		if env == "production" {
			return nil, errors.New("payment: PAYMENT_PROVIDER=mock is not allowed in production")
		}
		return NewMockProvider(), nil
	default:
		return nil, errors.New("payment: unknown provider: " + name)
	}
}