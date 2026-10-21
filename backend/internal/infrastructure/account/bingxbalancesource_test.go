package account

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/account"
	accountinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/account/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
)

// Erwartet: NewBingXBalanceSource(client bingx.GetClient) *BingXBalanceSource mit FetchBalance(ctx).
var _ accountinterfaces.BalanceSource = (*BingXBalanceSource)(nil)

type clientFake struct {
	response json.RawMessage
	err      error
	path     string
	params   []bingx.QueryParam
}

func (fake *clientFake) Get(_ context.Context, path string, params []bingx.QueryParam) (json.RawMessage, error) {
	fake.path, fake.params = path, params
	return fake.response, fake.err
}

func fixtureData(t *testing.T) json.RawMessage {
	t.Helper()
	content, err := os.ReadFile("../../../../fixtures/bingx/balance.json")
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func TestBalanceSource_Fixture(t *testing.T) {
	fake := &clientFake{response: fixtureData(t)}
	balance, err := NewBingXBalanceSource(fake).FetchBalance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if balance != (account.Balance{Balance: "1234.5678", Equity: "1259.5678", AvailableMargin: "1150.0678", UnrealizedProfit: "25.0000"}) {
		t.Errorf("balance = %+v", balance)
	}
	if fake.path != "/openApi/swap/v2/user/balance" {
		t.Errorf("Pfad = %s", fake.path)
	}
}

func TestBalanceSource_Mapping(t *testing.T) {
	testCases := []struct {
		name, data string
		expected   account.Balance
	}{
		{"Zahlen statt Strings", `{"balance":{"balance":1000.5,"equity":1010,"availableMargin":900.25}}`,
			account.Balance{Balance: "1000.5", Equity: "1010", AvailableMargin: "900.25"}},
		{"availableMargin fehlt, availableBalance als Kandidat", `{"balance":{"balance":"5","equity":"6","availableBalance":"4"}}`,
			account.Balance{Balance: "5", Equity: "6", AvailableMargin: "4"}},
		{"Extra-Felder werden ignoriert", `{"balance":{"balance":"5","equity":"6","availableMargin":"4","foo":{"x":1}}}`,
			account.Balance{Balance: "5", Equity: "6", AvailableMargin: "4"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := NewBingXBalanceSource(&clientFake{response: json.RawMessage(testCase.data)}).FetchBalance(context.Background())
			if err != nil || got != testCase.expected {
				t.Errorf("got=%+v err=%v, erwartet %+v", got, err, testCase.expected)
			}
		})
	}
}

func TestBalanceSource_Fehler(t *testing.T) {
	t.Run("Client-Fehler wird durchgereicht", func(t *testing.T) {
		boom := errors.New("boom")
		_, err := NewBingXBalanceSource(&clientFake{err: boom}).FetchBalance(context.Background())
		if !errors.Is(err, boom) {
			t.Errorf("err=%v", err)
		}
	})
	for name, data := range map[string]string{
		"data ist Array":           `[]`,
		"data ist null":            `null`,
		"balance-Objekt fehlt":     `{}`,
		"balance ist Zahl":         `{"balance":5}`,
		"balance-Feld fehlt":       `{"balance":{"equity":"1","availableMargin":"1"}}`,
		"equity fehlt":             `{"balance":{"balance":"1","availableMargin":"1"}}`,
		"Betrag keine Dezimalzahl": `{"balance":{"balance":"abc","equity":"1","availableMargin":"1"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewBingXBalanceSource(&clientFake{response: json.RawMessage(data)}).FetchBalance(context.Background())
			var unexpected *bingx.UnexpectedResponseError
			if !errors.As(err, &unexpected) {
				t.Errorf("err=%v, erwartet *bingx.UnexpectedResponseError", err)
			}
		})
	}
}
