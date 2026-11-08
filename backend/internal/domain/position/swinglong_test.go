package position_test

import (
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

func TestSwingLongSpan_Covers(t *testing.T) {
	closed := position.SwingLongSpan{PositionID: "1", Symbol: "BTC-USDT", OpenTime: 1000, CloseTime: 2000}
	open := position.SwingLongSpan{PositionID: "2", Symbol: "BTC-USDT", OpenTime: 1000}
	testCases := []struct {
		name   string
		span   position.SwingLongSpan
		symbol string
		time   int64
		want   bool
	}{
		{"geschlossen: genau Eröffnung", closed, "BTC-USDT", 1000, true},
		{"geschlossen: eins davor", closed, "BTC-USDT", 999, false},
		{"geschlossen: genau Schluss", closed, "BTC-USDT", 2000, true},
		{"geschlossen: eins danach", closed, "BTC-USDT", 2001, false},
		{"geschlossen: anderes Symbol", closed, "ETH-USDT", 1500, false},
		{"offen: genau Eröffnung", open, "BTC-USDT", 1000, true},
		{"offen: eins davor", open, "BTC-USDT", 999, false},
		{"offen: weit danach", open, "BTC-USDT", 9_999_999_999_999, true},
		{"offen: anderes Symbol", open, "ETH-USDT", 5000, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := testCase.span.Covers(testCase.symbol, testCase.time); got != testCase.want {
				t.Errorf("Covers = %v, erwartet %v", got, testCase.want)
			}
		})
	}
}
