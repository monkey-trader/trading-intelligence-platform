package account

import "time"

// Window ist ein Kalendermonat als Halboffenes Intervall [StartMs, EndExclusiveMs) in Epoch-ms.
type Window struct {
	StartMs        int64
	EndExclusiveMs int64
	Label          string // "YYYY-MM"
}

// MonthWindow liefert den Kalendermonat von nowMs in der Zeitzone loc.
func MonthWindow(nowMs int64, loc *time.Location) Window {
	local := time.UnixMilli(nowMs).In(loc)
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 1, 0)
	return Window{StartMs: start.UnixMilli(), EndExclusiveMs: end.UnixMilli(), Label: start.Format("2006-01")}
}
