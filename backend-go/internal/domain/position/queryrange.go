package position

import (
	"fmt"
	"time"
)

// MaxQueryRange ist die maximale Zeitspanne pro positionHistory-Abfrage (89 Tage).
// BingX erlaubt höchstens 90 Tage, sonst Fehler 109400.
const MaxQueryRange = 89 * 24 * time.Hour

// InvalidQueryRangeError meldet: Der Abfragebereich ist ungültig (from liegt nach to).
type InvalidQueryRangeError struct {
	From int64
	To   int64
}

func (err *InvalidQueryRangeError) Error() string {
	return fmt.Sprintf("Ungültiger Zeitbereich: from %d liegt nach to %d.", err.From, err.To)
}

func (*InvalidQueryRangeError) positionDomainError() {}
