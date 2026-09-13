package payment

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

// MockProvider is a development-only gateway simulator. It behaves like a
// real provider:
//   - InitPayment returns a unique authority and a URL pointing at our
//     local mock-gateway.html page.
//   - VerifyPayment always succeeds (unless the caller sent the "fail" flag
//     via the gateway page, in which case the callback URL is what drives
//     the failure — the mock itself doesn't know about it).
//
// Do NOT use this in production. The presence of PAYMENT_PROVIDER=mock in
// a production environment should be considered a configuration error.
type MockProvider struct {
	// GatewayURL is the base URL of the local mock page (e.g.
	// http://localhost:5500/mock-gateway.html). The authority is appended
	// as a query parameter when a payment is started.
	GatewayURL string
}

func NewMockProvider() *MockProvider {
	url := os.Getenv("PAYMENT_GATEWAY_URL")
	if url == "" {
		url = "http://localhost:5500/mock-gateway.html"
	}
	return &MockProvider{GatewayURL: url}
}

func (m *MockProvider) Name() string { return "mock" }

// InitPayment generates a UUID-based authority and returns a URL pointing
// at the local mock page with the authority and amount pre-filled.
//
// The "amount" and "callback" query params are intentionally included so
// the mock page can display them and, after the user clicks, redirect the
// browser to the real callback endpoint.
func (m *MockProvider) InitPayment(ctx context.Context, amount int64, description, callbackURL string) (*InitResult, error) {
	authority := "mock-" + uuid.New().String()

	sep := "?"
	if strings.Contains(m.GatewayURL, "?") {
		sep = "&"
	}

	url := fmt.Sprintf(
		"%s%sauthority=%s&amount=%d&callback=%s",
		m.GatewayURL, sep,
		authority, amount,
		urlEncode(callbackURL),
	)

	return &InitResult{
		Authority:  authority,
		GatewayURL: url,
	}, nil
}

// VerifyPayment always succeeds. The mock never talks to an external
// service; the decision to call Verify at all is what matters, because
// that's exactly what a real provider integration does.
func (m *MockProvider) VerifyPayment(ctx context.Context, authority string, amount int64) (*VerifyResult, error) {
	if authority == "" {
		return &VerifyResult{Success: false, ErrorMsg: "empty authority"}, nil
	}
	if !strings.HasPrefix(authority, "mock-") {
		return &VerifyResult{Success: false, ErrorMsg: "authority was not issued by this provider"}, nil
	}
	return &VerifyResult{
		Success: true,
		RefID:   "mockref-" + strings.TrimPrefix(authority, "mock-"),
	}, nil
}

// urlEncode is a tiny helper so we don't pull in net/url just for one call.
// It encodes only the characters that would break a query string; the
// callback URL is our own and always looks like http://host/path.
func urlEncode(s string) string {
	replacer := strings.NewReplacer(
		":", "%3A",
		"/", "%2F",
		"?", "%3F",
		"&", "%26",
		"=", "%3D",
	)
	return replacer.Replace(s)
}