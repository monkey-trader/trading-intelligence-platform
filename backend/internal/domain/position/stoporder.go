package position

// StopOrderRecord ist eine historische STOP-Order (beliebiger Status) einer Position.
type StopOrderRecord struct {
	PositionID   string
	TriggerPrice string
	Time         int64 // Epoch-ms der Order
}

// EarliestInitialStopLoss liefert den Trigger der zeitlich frühesten Order auf der Verlustseite des Entry
// (LONG unter, SHORT über dem Entry; gleich Entry zählt nicht). nil, wenn keine passt.
func EarliestInitialStopLoss(side PositionSide, entry string, records []StopOrderRecord) (*string, error) {
	entryValue, err := parseExact(entry)
	if err != nil {
		return nil, err
	}
	var earliest *StopOrderRecord
	for index := range records {
		value, err := parseExact(records[index].TriggerPrice)
		if err != nil {
			return nil, err
		}
		comparison := value.Cmp(entryValue)
		onLossSide := (side == Long && comparison < 0) || (side != Long && comparison > 0)
		if onLossSide && (earliest == nil || records[index].Time < earliest.Time) {
			earliest = &records[index]
		}
	}
	if earliest == nil {
		return nil, nil
	}
	return copyString(&earliest.TriggerPrice), nil
}
