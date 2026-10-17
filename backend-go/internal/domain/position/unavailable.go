package position

// UnavailableReason nennt, warum die offenen Positionen gerade nicht geliefert werden können.
type UnavailableReason string

// Gründe laut docs/openapi.yaml (Problem-Typen).
const (
	ReasonNotConfigured   UnavailableReason = "not-configured"
	ReasonRateLimited     UnavailableReason = "rate-limited"
	ReasonUnreachable     UnavailableReason = "unreachable"
	ReasonUpstreamError   UnavailableReason = "upstream-error"
	ReasonInvalidResponse UnavailableReason = "invalid-response"
)

// OpenPositionsUnavailableError meldet: Adapter übersetzen ihre Fehler hierhin. Cause nur fürs Log.
type OpenPositionsUnavailableError struct {
	Reason UnavailableReason
	Cause  error
}

func (err *OpenPositionsUnavailableError) Error() string {
	return "Offene Positionen nicht verfügbar: " + string(err.Reason)
}

// Unwrap liefert die Ursache.
func (err *OpenPositionsUnavailableError) Unwrap() error {
	return err.Cause
}
