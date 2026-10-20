package chart_test

import (
	"context"
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
	chartinfra "github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/chart"
)

func TestStopOrderHistory_RateLimitWirdAlsPortFehlerGemeldet(t *testing.T) {
	_, err := chartinfra.NewBingXStopOrderHistorySource(&fakeGetClient{err: &bingx.RateLimitError{}}).
		FetchStopOrders(context.Background(), "BTC-USDT", historyFrom, historyFrom+1)
	var limited *interfaces.RateLimitedError
	if !errors.As(err, &limited) {
		t.Fatalf("err = %v, erwartet *interfaces.RateLimitedError", err)
	}
	var rateLimit *bingx.RateLimitError
	if !errors.As(err, &rateLimit) {
		t.Errorf("Ursache muss erhalten bleiben: %v", err)
	}
}
