// Package clock enthält die Systemuhr als Adapter für den Clock-Port.
package clock

import "time"

// SystemClock liefert die echte aktuelle Zeit.
type SystemClock struct{}

// Now liefert die aktuelle Zeit.
func (SystemClock) Now() time.Time {
	return time.Now()
}
