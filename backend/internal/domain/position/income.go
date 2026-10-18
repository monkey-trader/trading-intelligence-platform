package position

// Income ist ein Kontoeintrag (Gebühr, Funding, realisierter PnL) mit primitiven Feldern.
type Income struct {
	TranID     string
	Symbol     string
	IncomeType string
	Amount     string
	Time       int64
	Asset      string
	TradeID    string
	Raw        string
}

// IncomeInput ist die primitive Eingabe der Factory.
type IncomeInput = Income

// InvalidTranIDError meldet: Die Transaktions-ID ist leer.
type InvalidTranIDError struct{ Value string }

// InvalidIncomeTypeError meldet: Der Income-Typ ist leer.
type InvalidIncomeTypeError struct{ Value string }

func (err *InvalidTranIDError) Error() string        { return "Ungültige tranId: " + err.Value }
func (err *InvalidIncomeTypeError) Error() string    { return "Ungültiger incomeType: " + err.Value }
func (*InvalidTranIDError) positionDomainError()     {}
func (*InvalidIncomeTypeError) positionDomainError() {}

// NewIncome prüft die Felder. Eine leere TradeID ist erlaubt (Funding hat keine).
func NewIncome(input IncomeInput) (Income, error) {
	if input.TranID == "" {
		return Income{}, &InvalidTranIDError{Value: input.TranID}
	}
	if input.IncomeType == "" {
		return Income{}, &InvalidIncomeTypeError{Value: input.IncomeType}
	}
	symbol, err := NewTradingSymbol(input.Symbol)
	if err != nil {
		return Income{}, err
	}
	amount, err := NewDecimalAmount(input.Amount)
	if err != nil {
		return Income{}, err
	}
	occurredAt, err := NewEpochMillis(input.Time)
	if err != nil {
		return Income{}, err
	}
	created := input
	created.Symbol = symbol.Value()
	created.Amount = amount.Value()
	created.Time = occurredAt.Value()
	return created, nil
}
