package position

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
)

var _ interfaces.IncomeSource = (*BingXIncomeSource)(nil)

type recordedCall struct {
	path   string
	params map[string]string
}

// incomeClientFake antwortet der Reihe nach und merkt sich die Aufrufe.
type incomeClientFake struct {
	responses []json.RawMessage
	err       error
	calls     []recordedCall
}

func (fake *incomeClientFake) Get(_ context.Context, path string, params []bingx.QueryParam) (json.RawMessage, error) {
	call := recordedCall{path: path, params: map[string]string{}}
	for _, param := range params {
		call.params[param.Name] = param.Value
	}
	fake.calls = append(fake.calls, call)
	if fake.err != nil {
		return nil, fake.err
	}
	index := len(fake.calls) - 1
	if index >= len(fake.responses) {
		return json.RawMessage(`[]`), nil
	}
	return fake.responses[index], nil
}

// incomePageJSON erzeugt count Einträge mit Zeiten firstTime, firstTime+1, … und eindeutigen tranIds ab idOffset.
func incomePageJSON(count int, firstTime int64, idOffset int) json.RawMessage {
	out := "["
	for i := 0; i < count; i++ {
		if i > 0 {
			out += ","
		}
		out += fmt.Sprintf(`{"symbol":"SUI-USDT","incomeType":"FUNDING_FEE","income":"0.1","asset":"USDT","time":%d,"tranId":"%d","tradeId":""}`,
			firstTime+int64(i), idOffset+i)
	}
	return json.RawMessage(out + "]")
}

func TestIncomeSource_EineSeiteMitRichtigenParametern(t *testing.T) {
	fake := &incomeClientFake{responses: []json.RawMessage{incomePageJSON(999, 1000, 1)}}
	entries, err := NewBingXIncomeSource(fake).FetchIncome(context.Background(), 1000, 5000)
	if err != nil || len(entries) != 999 {
		t.Fatalf("entries=%d err=%v", len(entries), err)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("999 Einträge sind keine volle Seite: %d Aufrufe, erwartet 1", len(fake.calls))
	}
	call := fake.calls[0]
	if call.path != "/openApi/swap/v2/user/income" || call.params["startTime"] != "1000" ||
		call.params["endTime"] != "5000" || call.params["limit"] != "1000" {
		t.Errorf("Aufruf falsch: %+v", call)
	}
	if _, hasSymbol := call.params["symbol"]; hasSymbol {
		t.Error("Income ist kontoweit, kein symbol-Parameter")
	}
}

func TestIncomeSource_VollePageBlaettertMitMaxTime(t *testing.T) {
	fake := &incomeClientFake{responses: []json.RawMessage{incomePageJSON(1000, 2000, 1), incomePageJSON(3, 3500, 5000)}}
	entries, err := NewBingXIncomeSource(fake).FetchIncome(context.Background(), 1000, 9000)
	if err != nil || len(entries) != 1003 {
		t.Fatalf("entries=%d err=%v", len(entries), err)
	}
	if len(fake.calls) != 2 {
		t.Fatalf("Aufrufe %d, erwartet 2", len(fake.calls))
	}
	// Seite 1: Zeiten 2000..2999, nächster startTime inklusiv 2999.
	if want := strconv.Itoa(2999); fake.calls[1].params["startTime"] != want {
		t.Errorf("startTime Seite 2 = %s, erwartet %s", fake.calls[1].params["startTime"], want)
	}
	if fake.calls[1].params["endTime"] != "9000" {
		t.Errorf("endTime muss gleich bleiben: %s", fake.calls[1].params["endTime"])
	}
}

func TestIncomeSource_VolleSeiteAmFensterEndeStoppt(t *testing.T) {
	// max(time) == to: ein Folgeaufruf holt Reste mit dieser Zeit, die leere Antwort beendet das Paging.
	fake := &incomeClientFake{responses: []json.RawMessage{incomePageJSON(1000, 4001, 1)}}
	_, err := NewBingXIncomeSource(fake).FetchIncome(context.Background(), 1000, 5000)
	if err != nil {
		t.Fatalf("Fehler: %v", err)
	}
	if len(fake.calls) != 2 {
		t.Errorf("Aufrufe %d, erwartet 2 (nächster Start 5000, Antwort leer)", len(fake.calls))
	}
}

func TestIncomeSource_VolleSeiteOhneFortschrittIstFehler(t *testing.T) {
	// 1000 Einträge mit identischem Zeitstempel = startTime: nicht blätterbar, kein stiller Datenverlust.
	same := make([]string, 1000)
	for i := range same {
		same[i] = fmt.Sprintf(`{"symbol":"SUI-USDT","incomeType":"FUNDING_FEE","income":"0.1","asset":"USDT","time":2000,"tranId":"%d","tradeId":""}`, i)
	}
	page := json.RawMessage("[" + strings.Join(same, ",") + "]")
	fake := &incomeClientFake{responses: []json.RawMessage{page}}
	if _, err := NewBingXIncomeSource(fake).FetchIncome(context.Background(), 2000, 9000); err == nil {
		t.Fatal("Fehler erwartet")
	}
}

func TestIncomeSource_DedupliziertUeberSeitengrenze(t *testing.T) {
	// Seite 2 wiederholt den letzten Eintrag von Seite 1 (tranId 1000 mit Zeit 2999).
	fake := &incomeClientFake{responses: []json.RawMessage{incomePageJSON(1000, 2000, 1), incomePageJSON(2, 2999, 1000)}}
	entries, err := NewBingXIncomeSource(fake).FetchIncome(context.Background(), 1000, 9000)
	if err != nil || len(entries) != 1001 {
		t.Fatalf("entries=%d err=%v", len(entries), err)
	}
}

func TestIncomeSource_NullDataIstLeer(t *testing.T) {
	fake := &incomeClientFake{responses: []json.RawMessage{json.RawMessage(`null`)}}
	entries, err := NewBingXIncomeSource(fake).FetchIncome(context.Background(), 1, 2)
	if err != nil || len(entries) != 0 {
		t.Errorf("entries=%v err=%v", entries, err)
	}
}

func TestIncomeSource_Fehler(t *testing.T) {
	t.Run("Client-Fehler wird durchgereicht", func(t *testing.T) {
		boom := errors.New("boom")
		_, err := NewBingXIncomeSource(&incomeClientFake{err: boom}).FetchIncome(context.Background(), 1, 2)
		if !errors.Is(err, boom) {
			t.Errorf("err=%v", err)
		}
	})
	t.Run("von nach bis", func(t *testing.T) {
		_, err := NewBingXIncomeSource(&incomeClientFake{}).FetchIncome(context.Background(), 5, 2)
		var rangeErr *domainposition.InvalidQueryRangeError
		if !errors.As(err, &rangeErr) {
			t.Errorf("err=%v", err)
		}
	})
	t.Run("abgebrochener Kontext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := NewBingXIncomeSource(&incomeClientFake{}).FetchIncome(ctx, 1, 2)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err=%v", err)
		}
	})
	t.Run("kaputte Antwort", func(t *testing.T) {
		fake := &incomeClientFake{responses: []json.RawMessage{json.RawMessage(`{"x":1}`)}}
		_, err := NewBingXIncomeSource(fake).FetchIncome(context.Background(), 1, 2)
		var unexpectedErr *bingx.UnexpectedResponseError
		if !errors.As(err, &unexpectedErr) {
			t.Errorf("err=%v", err)
		}
	})
}
