package position

import (
	"errors"
	"strings"
	"testing"
)

func TestLimitErrors_SindDomainErrorsMitKontext(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains []string
	}{
		{"QueryRangeTooLarge", &QueryRangeTooLargeError{From: 10, To: 99}, []string{"10", "99", "89"}},
		{"PageLimitExceeded", &PageLimitExceededError{Symbol: "ADA-USDT", From: 1, To: 2, Pages: 1000}, []string{"ADA-USDT", "1000"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var domainErr PositionDomainError
			if !errors.As(tt.err, &domainErr) {
				t.Fatal("muss PositionDomainError sein")
			}
			for _, part := range tt.contains {
				if !strings.Contains(tt.err.Error(), part) {
					t.Errorf("Meldung %q enthält %q nicht", tt.err.Error(), part)
				}
			}
		})
	}
}
