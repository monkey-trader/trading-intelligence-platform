// Package interfaces enthält Ports des Shared-Kerns.
package interfaces

import "time"

// Clock liefert die aktuelle Zeit, damit Tests eine feste Zeit injizieren können.
type Clock interface {
	Now() time.Time
}
