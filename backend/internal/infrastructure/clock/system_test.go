package clock_test

import (
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/clock"
)

func TestSystemClock_LiefertDieAktuelleZeit(t *testing.T) {
	var systemClock interfaces.Clock = clock.SystemClock{}

	before := time.Now()
	now := systemClock.Now()
	after := time.Now()

	if now.Before(before) || now.After(after) {
		t.Errorf("Now = %v, erwartet zwischen %v und %v", now, before, after)
	}
	if now.UnixMilli() <= 0 {
		t.Errorf("UnixMilli = %d, erwartet > 0", now.UnixMilli())
	}
}
