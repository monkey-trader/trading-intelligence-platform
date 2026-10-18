package position

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
)

var _ interfaces.ContractSource = (*BingXContractSource)(nil)

type contractClientFake struct {
	data  json.RawMessage
	err   error
	paths []string
}

func (fake *contractClientFake) Get(_ context.Context, path string, _ []bingx.QueryParam) (json.RawMessage, error) {
	fake.paths = append(fake.paths, path)
	return fake.data, fake.err
}

func TestContractSource_Tabelle(t *testing.T) {
	fixture, err := os.ReadFile("../../../../fixtures/bingx/contracts.sample.json")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		data    string
		want    []string
		wantErr bool
	}{
		{"Fixture: nur USDT, USDC fällt weg", string(fixture), []string{"BTC-USDT", "NCCONICKEL2USD-USDT"}, false},
		{"dedupliziert und sortiert", `[{"symbol":"ETH-USDT","currency":"USDT"},{"symbol":"BTC-USDT","currency":"USDT"},{"symbol":"ETH-USDT","currency":"USDT"}]`,
			[]string{"BTC-USDT", "ETH-USDT"}, false},
		{"Eintrag ohne Symbol wird ignoriert", `[{"currency":"USDT"},{"symbol":"","currency":"USDT"},{"symbol":"BTC-USDT","currency":"USDT"}]`,
			[]string{"BTC-USDT"}, false},
		{"leere Liste", `[]`, []string{}, false},
		{"null", `null`, []string{}, false},
		{"kein Array", `{"a":1}`, nil, true},
		{"kaputtes JSON", `[`, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &contractClientFake{data: json.RawMessage(tt.data)}
			got, err := NewBingXContractSource(fake).FetchContracts(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) || (len(got) > 0 && !reflect.DeepEqual(got, tt.want)) {
				t.Errorf("Symbole = %v, erwartet %v", got, tt.want)
			}
			if len(fake.paths) != 1 || fake.paths[0] != "/openApi/swap/v2/quote/contracts" {
				t.Errorf("Pfad = %v", fake.paths)
			}
		})
	}
}

func TestContractSource_ClientFehlerWirdDurchgereicht(t *testing.T) {
	rateLimit := &bingx.RateLimitError{}
	_, err := NewBingXContractSource(&contractClientFake{err: rateLimit}).FetchContracts(context.Background())
	var target *bingx.RateLimitError
	if !errors.As(err, &target) {
		t.Errorf("err = %v, erwartet RateLimitError (direkt oder als Cause)", err)
	}
}
