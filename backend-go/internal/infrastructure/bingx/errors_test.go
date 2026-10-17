package bingx_test

import (
	"strings"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
)

func TestAPIError_ErrorKuerztDieMeldungNicht(t *testing.T) {
	message := strings.Repeat("Z", 500)
	apiError := &bingx.APIError{Code: 109400, Msg: message, Path: "/p"}

	text := apiError.Error()

	if !strings.Contains(text, message) {
		t.Errorf("Msg nicht vollständig im Error()-Text (Länge %d)", len(text))
	}
	if strings.Contains(text, "…") {
		t.Errorf("Error() darf nicht kürzen: %q", text)
	}
}
