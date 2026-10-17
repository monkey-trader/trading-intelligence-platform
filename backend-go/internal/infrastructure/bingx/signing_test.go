package bingx_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
)

// Fälle portiert aus backend/src/infrastructure/bingx/signing.test.ts.

func referenceSignature(query, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(query))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestSignQuery(t *testing.T) {
	testCases := []struct {
		name     string
		query    string
		secret   string
		expected string
	}{
		{
			name:     "bekannter HMAC-SHA256-Testvektor",
			query:    "The quick brown fox jumps over the lazy dog",
			secret:   "key",
			expected: "f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8",
		},
		{
			name:     "BingX-Query-String",
			query:    "symbol=ONDO-USDT&startTs=1780000000000&endTs=1780100000000&timestamp=1780100000123",
			secret:   "test-secret-abc",
			expected: referenceSignature("symbol=ONDO-USDT&startTs=1780000000000&endTs=1780100000000&timestamp=1780100000123", "test-secret-abc"),
		},
		{
			name:     "leerer Query",
			query:    "",
			secret:   "secret",
			expected: referenceSignature("", "secret"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := bingx.SignQuery(testCase.query, testCase.secret); got != testCase.expected {
				t.Errorf("SignQuery = %q, erwartet %q", got, testCase.expected)
			}
		})
	}
}

func TestSignQuery_VerschiedeneQueriesVerschiedeneSignaturen(t *testing.T) {
	first := bingx.SignQuery("timestamp=1", "secret")
	second := bingx.SignQuery("timestamp=2", "secret")
	if first == "" || first == second {
		t.Errorf("Signaturen %q und %q, erwartet verschieden und nicht leer", first, second)
	}
}
