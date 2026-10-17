package position

// OpenPosition ist eine aktuell offene Position mit primitiven Feldern. Beträge bleiben Strings.
type OpenPosition struct {
	PositionID       string
	Symbol           string
	Side             PositionSide
	Isolated         bool
	PositionAmount   string
	AvgOpenPrice     string
	MarkPrice        string
	Leverage         float64
	InitialMargin    string
	Margin           string
	PositionValue    string
	UnrealizedProfit string
	RealisedProfit   string
	PnlRatio         string
	// LiquidationPrice "0" bedeutet keine Liquidation.
	LiquidationPrice string
	OpenTime         int64
	UpdateTime       int64
}

// OpenPositionsSnapshot sind die offenen Positionen zu einem Zeitpunkt (FetchedAt in Epoch-ms).
type OpenPositionsSnapshot struct {
	Positions []OpenPosition
	FetchedAt int64
}

// OpenPositionInput ist die primitive Eingabe der Factory. Side wird über das VO geprüft.
type OpenPositionInput struct {
	PositionID       string
	Symbol           string
	Side             string
	Isolated         bool
	PositionAmount   string
	AvgOpenPrice     string
	MarkPrice        string
	Leverage         float64
	InitialMargin    string
	Margin           string
	PositionValue    string
	UnrealizedProfit string
	RealisedProfit   string
	PnlRatio         string
	LiquidationPrice string
	OpenTime         int64
	UpdateTime       int64
}

// NewOpenPosition prüft erst die Zeiten, dann UpdateBeforeOpen, dann die Felder (wie TS).
func NewOpenPosition(input OpenPositionInput) (OpenPosition, error) {
	openTime, err := NewEpochMillis(input.OpenTime)
	if err != nil {
		return OpenPosition{}, err
	}
	updateTime, err := NewEpochMillis(input.UpdateTime)
	if err != nil {
		return OpenPosition{}, err
	}
	if updateTime.Value() < openTime.Value() {
		return OpenPosition{}, &UpdateBeforeOpenError{
			PositionID: input.PositionID,
			OpenTime:   openTime.Value(),
			UpdateTime: updateTime.Value(),
		}
	}

	positionID, err := NewPositionID(input.PositionID)
	if err != nil {
		return OpenPosition{}, err
	}
	symbol, err := NewTradingSymbol(input.Symbol)
	if err != nil {
		return OpenPosition{}, err
	}
	side, err := NewPositionSide(input.Side)
	if err != nil {
		return OpenPosition{}, err
	}

	// Reihenfolge wie in OpenPositionFactory.ts; der erste Fehler gewinnt.
	fields := fieldValidator{}
	created := OpenPosition{
		PositionID:       positionID.Value(),
		Symbol:           symbol.Value(),
		Side:             side,
		Isolated:         input.Isolated,
		PositionAmount:   fields.amount(input.PositionAmount),
		AvgOpenPrice:     fields.amount(input.AvgOpenPrice),
		MarkPrice:        fields.amount(input.MarkPrice),
		Leverage:         fields.leverage(input.Leverage),
		InitialMargin:    fields.amount(input.InitialMargin),
		Margin:           fields.amount(input.Margin),
		PositionValue:    fields.amount(input.PositionValue),
		UnrealizedProfit: fields.amount(input.UnrealizedProfit),
		RealisedProfit:   fields.amount(input.RealisedProfit),
		PnlRatio:         fields.amount(input.PnlRatio),
		LiquidationPrice: fields.amount(input.LiquidationPrice),
		OpenTime:         openTime.Value(),
		UpdateTime:       updateTime.Value(),
	}
	if fields.firstError != nil {
		return OpenPosition{}, fields.firstError
	}
	return created, nil
}

// fieldValidator prüft Felder der Reihe nach über ihre VOs und merkt sich den ersten Fehler.
// Go wertet Composite-Literale in Quelltext-Reihenfolge aus, darum bleibt die Prüfreihenfolge erhalten.
type fieldValidator struct {
	firstError error
}

func (validator *fieldValidator) amount(rawAmount string) string {
	if validator.firstError != nil {
		return ""
	}
	amount, err := NewDecimalAmount(rawAmount)
	if err != nil {
		validator.firstError = err
		return ""
	}
	return amount.Value()
}

func (validator *fieldValidator) leverage(rawLeverage float64) float64 {
	if validator.firstError != nil {
		return 0
	}
	leverage, err := NewLeverage(rawLeverage)
	if err != nil {
		validator.firstError = err
		return 0
	}
	return leverage.Value()
}
