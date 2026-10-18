package position

// ClosedPosition ist eine geschlossene Position mit primitiven Feldern. Beträge bleiben Strings.
type ClosedPosition struct {
	PositionID     string
	Symbol         string
	Side           PositionSide
	Isolated       bool
	OpenTime       int64
	CloseTime      int64
	AvgOpenPrice   string
	AvgClosePrice  string
	RealisedProfit string
	NetProfit      string
	PositionAmount string
	ClosedAmount   string
	Leverage       float64
	Commission     string
	TotalFunding   string
	FullyClosed    bool
	Raw            string
}

// ClosedPositionInput ist die primitive Eingabe der Factory. Side wird über das VO geprüft.
type ClosedPositionInput struct {
	PositionID     string
	Symbol         string
	Side           string
	Isolated       bool
	OpenTime       int64
	CloseTime      int64
	AvgOpenPrice   string
	AvgClosePrice  string
	RealisedProfit string
	NetProfit      string
	PositionAmount string
	ClosedAmount   string
	Leverage       float64
	Commission     string
	TotalFunding   string
	FullyClosed    bool
	Raw            string
}

// NewClosedPosition prüft erst die Zeiten, dann UpdateBeforeOpen, dann die Felder (wie OpenPosition).
func NewClosedPosition(input ClosedPositionInput) (ClosedPosition, error) {
	openTime, err := NewEpochMillis(input.OpenTime)
	if err != nil {
		return ClosedPosition{}, err
	}
	closeTime, err := NewEpochMillis(input.CloseTime)
	if err != nil {
		return ClosedPosition{}, err
	}
	if closeTime.Value() < openTime.Value() {
		return ClosedPosition{}, &UpdateBeforeOpenError{
			PositionID: input.PositionID,
			OpenTime:   openTime.Value(),
			UpdateTime: closeTime.Value(),
		}
	}

	positionID, err := NewPositionID(input.PositionID)
	if err != nil {
		return ClosedPosition{}, err
	}
	symbol, err := NewTradingSymbol(input.Symbol)
	if err != nil {
		return ClosedPosition{}, err
	}
	side, err := NewPositionSide(input.Side)
	if err != nil {
		return ClosedPosition{}, err
	}

	fields := fieldValidator{}
	created := ClosedPosition{
		PositionID:     positionID.Value(),
		Symbol:         symbol.Value(),
		Side:           side,
		Isolated:       input.Isolated,
		OpenTime:       openTime.Value(),
		CloseTime:      closeTime.Value(),
		AvgOpenPrice:   fields.amount(input.AvgOpenPrice),
		AvgClosePrice:  fields.amount(input.AvgClosePrice),
		RealisedProfit: fields.amount(input.RealisedProfit),
		NetProfit:      fields.amount(input.NetProfit),
		PositionAmount: fields.amount(input.PositionAmount),
		ClosedAmount:   fields.amount(input.ClosedAmount),
		Leverage:       fields.leverage(input.Leverage),
		Commission:     fields.amount(input.Commission),
		TotalFunding:   fields.amount(input.TotalFunding),
		FullyClosed:    input.FullyClosed,
		Raw:            input.Raw,
	}
	if fields.firstError != nil {
		return ClosedPosition{}, fields.firstError
	}
	return created, nil
}
