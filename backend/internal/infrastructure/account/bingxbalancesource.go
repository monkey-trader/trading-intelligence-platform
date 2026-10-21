// Package account enthält die BingX-Anbindung der Kontosicht.
package account

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/account"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
	infraposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/position"
)

const balancePath = "/openApi/swap/v2/user/balance"

// BingXBalanceSource liest die Kontowerte über den signierten Client.
type BingXBalanceSource struct{ client bingx.GetClient }

// NewBingXBalanceSource baut die Source.
func NewBingXBalanceSource(client bingx.GetClient) *BingXBalanceSource {
	return &BingXBalanceSource{client: client}
}

// FetchBalance ruft die Balance ab; bekannte BingX-Fehler werden zu *OpenPositionsUnavailableError.
func (source *BingXBalanceSource) FetchBalance(ctx context.Context) (account.Balance, error) {
	data, err := source.client.Get(ctx, balancePath, nil)
	if err != nil {
		return account.Balance{}, infraposition.ToUnavailable(err)
	}
	balance, err := mapBalance(data)
	if err != nil {
		return account.Balance{}, infraposition.ToUnavailable(err)
	}
	return balance, nil
}

func mapBalance(data json.RawMessage) (account.Balance, error) {
	var envelope struct {
		Balance map[string]json.RawMessage `json:"balance"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil || envelope.Balance == nil {
		return account.Balance{}, &bingx.UnexpectedResponseError{Message: "BingX-Balance: Objekt balance fehlt."}
	}
	balance, err := amountOf(envelope.Balance, "balance")
	if err != nil {
		return account.Balance{}, err
	}
	equity, err := amountOf(envelope.Balance, "equity")
	if err != nil {
		return account.Balance{}, err
	}
	margin, err := amountOf(envelope.Balance, "availableMargin", "availableBalance")
	if err != nil {
		return account.Balance{}, err
	}
	// Optional: ohne das Feld bleibt der Wert leer, die Kontoansicht funktioniert trotzdem.
	unrealized, _ := amountOf(envelope.Balance, "unrealizedProfit")
	return account.Balance{Balance: balance, Equity: equity, AvailableMargin: margin, UnrealizedProfit: unrealized}, nil
}

// amountOf nimmt das erste vorhandene Feld (String oder Zahl) und prüft es als Dezimalzahl.
func amountOf(fields map[string]json.RawMessage, keys ...string) (string, error) {
	for _, key := range keys {
		raw, found := fields[key]
		if !found {
			continue
		}
		text := strings.TrimSpace(string(raw))
		if strings.HasPrefix(text, `"`) {
			var unquoted string
			if err := json.Unmarshal(raw, &unquoted); err != nil {
				break
			}
			text = unquoted
		}
		if _, err := position.NewDecimalAmount(text); err != nil {
			return "", &bingx.UnexpectedResponseError{Message: "BingX-Balance: Feld " + key + " ist keine Dezimalzahl."}
		}
		return text, nil
	}
	return "", &bingx.UnexpectedResponseError{Message: "BingX-Balance: Feld " + keys[0] + " fehlt."}
}
