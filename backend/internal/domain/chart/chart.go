// Package chart enthält den fachlichen Kern des Positions-Charts: Kerzen, Schutzorders und Abstände.
package chart

import "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"

// CandleInterval ist ein von der API unterstütztes Kerzenintervall.
type CandleInterval string

// Unterstützte Intervalle laut docs/openapi.yaml.
const (
	Interval5m  CandleInterval = "5m"
	Interval15m CandleInterval = "15m"
	Interval1h  CandleInterval = "1h"
	Interval4h  CandleInterval = "4h"
	Interval1d  CandleInterval = "1d"
)

// Standardwerte und Grenzen der Abfrage.
const (
	DefaultInterval = Interval15m
	DefaultLimit    = 300
	MinLimit        = 1
	MaxLimit        = 1000
)

// distanceScale ist die Anzahl der Nachkommastellen des Abstandsverhältnisses.
const distanceScale = 8

// ParseCandleInterval prüft das Intervall exakt (Groß-/Kleinschreibung zählt).
func ParseCandleInterval(raw string) (CandleInterval, error) {
	switch interval := CandleInterval(raw); interval {
	case Interval5m, Interval15m, Interval1h, Interval4h, Interval1d:
		return interval, nil
	}
	return "", &UnsupportedIntervalError{Value: raw}
}

// ParseLimit prüft, dass das Limit zwischen MinLimit und MaxLimit liegt.
func ParseLimit(limit int) (int, error) {
	if limit < MinLimit || limit > MaxLimit {
		return 0, &LimitOutOfRangeError{Value: limit}
	}
	return limit, nil
}

// ParseTriggerPrice prüft, dass der Auslösepreis eine gültige Dezimalzahl größer 0 ist.
func ParseTriggerPrice(raw string) (string, error) {
	// Subtraktion von 0 prüft das Format; das Vorzeichen des Ergebnisses ist das der Zahl.
	difference, err := decimal.Subtract(raw, "0")
	if err != nil {
		return "", err
	}
	if difference == "0" || difference[0] == '-' {
		return "", &TriggerPriceMustBePositiveError{Value: raw}
	}
	return raw, nil
}

// DistanceRatio liefert (trigger - entry) / entry mit genau 8 Nachkommastellen.
func DistanceRatio(entry, trigger string) (string, error) {
	difference, err := decimal.Subtract(trigger, entry)
	if err != nil {
		return "", err
	}
	ratio, err := decimal.Divide(difference, entry, distanceScale)
	if err != nil {
		return "", err
	}
	return padFraction(ratio, distanceScale), nil
}

// padFraction füllt die Nachkommastellen mit Nullen auf die feste Länge auf.
func padFraction(value string, scale int) string {
	integerPart, fractionPart := value, ""
	for index := 0; index < len(value); index++ {
		if value[index] == '.' {
			integerPart, fractionPart = value[:index], value[index+1:]
			break
		}
	}
	for len(fractionPart) < scale {
		fractionPart += "0"
	}
	return integerPart + "." + fractionPart
}

// Candle ist eine Kerze; Preise und Volumen bleiben exakte Dezimal-Strings, OpenTime ist Epoch-ms.
type Candle struct {
	Symbol   string
	Interval CandleInterval
	OpenTime int64
	Open     string
	High     string
	Low      string
	Close    string
	Volume   string
}

// ProtectiveKind unterscheidet Stop-Loss und Take-Profit.
type ProtectiveKind string

// Arten der Schutzorders.
const (
	StopLoss   ProtectiveKind = "STOP_LOSS"
	TakeProfit ProtectiveKind = "TAKE_PROFIT"
)

// ProtectiveOrder ist eine offene SL- oder TP-Order einer Position. IDs bleiben Strings (über 2^53).
type ProtectiveOrder struct {
	OrderID      string
	PositionID   string
	Symbol       string
	Kind         ProtectiveKind
	TriggerPrice string
	Quantity     string
	Status       string
}

// ProtectiveOrdersSnapshot sind die Schutzorders zu einem Zeitpunkt (FetchedAt in Epoch-ms).
type ProtectiveOrdersSnapshot struct {
	Orders    []ProtectiveOrder
	FetchedAt int64
}
