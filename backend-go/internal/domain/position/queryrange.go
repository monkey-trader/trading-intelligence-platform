package position

import "time"

// MaxQueryRange ist die maximale Zeitspanne pro positionHistory-Abfrage (89 Tage).
// BingX erlaubt höchstens 90 Tage, sonst Fehler 109400.
const MaxQueryRange = 89 * 24 * time.Hour
