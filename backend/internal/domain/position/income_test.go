package position

import (
	"errors"
	"testing"
)

func validIncomeInput() IncomeInput {
	return IncomeInput{
		TranID: "8000000000000000002", Symbol: "ONDO-USDT", IncomeType: "REALIZED_PNL",
		Amount: "1.7000", Time: 1780018017000, Asset: "USDT", TradeID: "7000000000000000002", Raw: `{"tranId":"8000000000000000002"}`,
	}
}

func TestNewIncome_HappyPath(t *testing.T) {
	input := validIncomeInput()
	got, err := NewIncome(input)
	if err != nil {
		t.Fatalf("Fehler: %v", err)
	}
	if got != Income(input) {
		t.Errorf("got %+v, want %+v", got, input)
	}
}

func TestNewIncome_LeereTradeIdIstErlaubt(t *testing.T) {
	input := validIncomeInput()
	input.TradeID = ""
	if _, err := NewIncome(input); err != nil {
		t.Fatalf("leere tradeId (Funding) muss erlaubt sein: %v", err)
	}
}

func TestNewIncome_Fehler(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*IncomeInput)
		check  func(error) bool
	}{
		{"leere tranId", func(i *IncomeInput) { i.TranID = "" }, func(err error) bool { var e *InvalidTranIDError; return errors.As(err, &e) }},
		{"leerer incomeType", func(i *IncomeInput) { i.IncomeType = "" }, func(err error) bool { var e *InvalidIncomeTypeError; return errors.As(err, &e) }},
		{"Betrag kein Dezimal", func(i *IncomeInput) { i.Amount = "abc" }, func(err error) bool { var e *InvalidDecimalAmountError; return errors.As(err, &e) }},
		{"Betrag leer", func(i *IncomeInput) { i.Amount = "" }, func(err error) bool { var e *InvalidDecimalAmountError; return errors.As(err, &e) }},
		{"Zeit 0", func(i *IncomeInput) { i.Time = 0 }, func(err error) bool { var e *InvalidTimestampError; return errors.As(err, &e) }},
		{"Symbol ungültig", func(i *IncomeInput) { i.Symbol = "ONDO" }, func(err error) bool { var e *InvalidTradingSymbolError; return errors.As(err, &e) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validIncomeInput()
			tt.mutate(&input)
			_, err := NewIncome(input)
			if err == nil || !tt.check(err) {
				t.Fatalf("falscher Fehler: %v", err)
			}
			var domainErr PositionDomainError
			if !errors.As(err, &domainErr) {
				t.Errorf("muss PositionDomainError sein: %v", err)
			}
		})
	}
}
