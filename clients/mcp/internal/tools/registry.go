// Package tools definiert die MCP-Tools des Journals (nur lesend).
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
)

// ErrInvalidArguments meldet ungültige Tool-Argumente (vor dem HTTP-Aufruf).
var ErrInvalidArguments = errors.New("ungültige Argumente")

// Getter ist der Teil des API-Clients, den die Tools brauchen.
type Getter interface {
	Get(ctx context.Context, path string, query url.Values) (json.RawMessage, error)
}

// Tool ist ein MCP-Tool.
type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	Call        func(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
}

// Registry hält alle Tools in fester Reihenfolge.
type Registry struct{ list []Tool }

const (
	emptySchema  = `{"type":"object","properties":{},"additionalProperties":false}`
	closedSchema = `{"type":"object","properties":{` +
		`"from":{"type":"integer","minimum":0,"description":"Beginn des Zeitraums, Unix-Millisekunden (Schliesszeitpunkt)"},` +
		`"to":{"type":"integer","minimum":0,"description":"Ende des Zeitraums, Unix-Millisekunden, nicht kleiner als from"},` +
		`"symbol":{"type":"string","description":"Handelspaar, z.B. SUI-USDT"},` +
		`"limit":{"type":"integer","minimum":1,"maximum":500,"description":"Maximale Anzahl Positionen (1 bis 500)"}},` +
		`"additionalProperties":false}`
	noteSchema = `{"type":"object","properties":{` +
		`"positionId":{"type":"string","pattern":"^[0-9]{1,20}$","description":"Position-ID als Ziffernfolge (bis 20 Stellen), als String"}},` +
		`"required":["positionId"],"additionalProperties":false}`
)

// NewRegistry registriert die vier Journal-Tools.
func NewRegistry(api Getter) *Registry {
	simple := func(path string) func(context.Context, json.RawMessage) (json.RawMessage, error) {
		return func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
			if _, err := parseObject(args); err != nil {
				return nil, err
			}
			return api.Get(ctx, path, nil)
		}
	}
	return &Registry{list: []Tool{
		{
			Name:        "get_open_positions",
			Description: "Liefert die aktuell offenen Positionen mit Entry, Stop-Loss, Take-Profit-Stufen, Margin, unrealisiertem PnL und Liquidationsabstand.",
			InputSchema: json.RawMessage(emptySchema),
			Call:        simple("/positions/open"),
		},
		{
			Name:        "get_closed_positions",
			Description: "Liefert geschlossene Positionen mit R-Ergebnis, BE- und TP1-Level. Optional nach Zeitraum (from/to in Unix-Millisekunden), Symbol und Anzahl (limit 1 bis 500) filtern.",
			InputSchema: json.RawMessage(closedSchema),
			Call: func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
				return callClosed(ctx, api, args)
			},
		},
		{
			Name:        "get_account_summary",
			Description: "Liefert die Kontoübersicht: Guthaben, Equity, verfügbare Margin, realisierter PnL für heute, 7 und 30 Tage sowie den laufenden Monat.",
			InputSchema: json.RawMessage(emptySchema),
			Call:        simple("/account/summary"),
		},
		{
			Name:        "get_position_note",
			Description: "Liefert die Journal-Notiz (Setup, Bewertung, Fehler, Notizen) zu einer Position anhand der positionId.",
			InputSchema: json.RawMessage(noteSchema),
			Call: func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
				return callNote(ctx, api, args)
			},
		},
	}}
}

// List liefert alle Tools.
func (r *Registry) List() []Tool { return append([]Tool(nil), r.list...) }

// Lookup sucht ein Tool nach Name.
func (r *Registry) Lookup(name string) (Tool, bool) {
	for _, tool := range r.list {
		if tool.Name == name {
			return tool, true
		}
	}
	return Tool{}, false
}

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidArguments, fmt.Sprintf(format, a...))
}

// parseObject liest Argumente; fehlend oder null zählt als leeres Objekt.
func parseObject(args json.RawMessage) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(args)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return map[string]json.RawMessage{}, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return nil, invalid("Objekt erwartet")
	}
	if fields == nil {
		fields = map[string]json.RawMessage{}
	}
	return fields, nil
}

// intField liest eine ganze Zahl im Bereich [min, max].
func intField(fields map[string]json.RawMessage, name string, min, max int64) (int64, bool, error) {
	raw, ok := fields[name]
	if !ok {
		return 0, false, nil
	}
	value, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil || value < min || value > max {
		return 0, false, invalid("%s muss eine ganze Zahl zwischen %d und %d sein", name, min, max)
	}
	return value, true, nil
}

func callClosed(ctx context.Context, api Getter, args json.RawMessage) (json.RawMessage, error) {
	fields, err := parseObject(args)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	from, hasFrom, err := intField(fields, "from", 0, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	to, hasTo, err := intField(fields, "to", 0, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	if hasFrom && hasTo && to < from {
		return nil, invalid("to darf nicht kleiner als from sein")
	}
	limit, hasLimit, err := intField(fields, "limit", 1, 500)
	if err != nil {
		return nil, err
	}
	if raw, ok := fields["symbol"]; ok {
		var symbol string
		if json.Unmarshal(raw, &symbol) != nil {
			return nil, invalid("symbol muss eine Zeichenkette sein")
		}
		if symbol != "" {
			query.Set("symbol", symbol)
		}
	}
	if hasFrom {
		query.Set("from", strconv.FormatInt(from, 10))
	}
	if hasTo {
		query.Set("to", strconv.FormatInt(to, 10))
	}
	if hasLimit {
		query.Set("limit", strconv.FormatInt(limit, 10))
	}
	return api.Get(ctx, "/positions/closed", query)
}

func callNote(ctx context.Context, api Getter, args json.RawMessage) (json.RawMessage, error) {
	fields, err := parseObject(args)
	if err != nil {
		return nil, err
	}
	var positionID string
	if json.Unmarshal(fields["positionId"], &positionID) != nil || !isPositionID(positionID) {
		return nil, invalid("positionId muss eine Ziffernfolge mit 1 bis 20 Stellen sein")
	}
	return api.Get(ctx, "/positions/"+positionID+"/note", nil)
}

func isPositionID(value string) bool {
	if len(value) < 1 || len(value) > 20 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
