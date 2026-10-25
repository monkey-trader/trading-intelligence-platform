package position_test

import (
	"context"
	"errors"
	"testing"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// Erwartet:
//   (*RiskResolver).WithOverrides(interfaces.PositionInitialStopLossRepository) *RiskResolver
//   (*RiskResolver).ForgetPosition(positionID string)
//   (*GetOpenPositionsUseCase).WithInitialStopLossOverrides(overrides) *GetOpenPositionsUseCase (nach WithRisk)
//   OpenPositionEntry.InitialStopLossSource *string ("MANUAL" | "AUTO" | nil)
//   NewSetInitialStopLossUseCase(entries interfaces.PositionEntryLookup, overrides, resolver *RiskResolver, now func() int64)
//     .Execute(ctx, positionID, rawStopLoss string) (SetInitialStopLossResult{Override position.InitialStopLossOverride; RiskDistance string}, error)
//   NewClearInitialStopLossUseCase(overrides, resolver *RiskResolver).Execute(ctx, positionID string) error
//   Fehler: *chart.PositionNotFoundError, *position.StopLossAtEntryError, *position.StopLossWrongSideError,
//   *decimal.InvalidDecimalError, Quellfehler unverändert durchgereicht (errors.Is).

type fakeOverrides struct {
	stored    map[string]position.InitialStopLossOverride
	getErr    error
	setErr    error
	deleteErr error
	deleted   []string
}

func (repo *fakeOverrides) Set(_ context.Context, override position.InitialStopLossOverride) error {
	if repo.setErr != nil {
		return repo.setErr
	}
	if repo.stored == nil {
		repo.stored = map[string]position.InitialStopLossOverride{}
	}
	repo.stored[override.PositionID] = override
	return nil
}

func (repo *fakeOverrides) Get(_ context.Context, positionID string) (position.InitialStopLossOverride, bool, error) {
	if repo.getErr != nil {
		return position.InitialStopLossOverride{}, false, repo.getErr
	}
	override, found := repo.stored[positionID]
	return override, found, nil
}

func (repo *fakeOverrides) Delete(_ context.Context, positionID string) error {
	repo.deleted = append(repo.deleted, positionID)
	if repo.deleteErr != nil {
		return repo.deleteErr
	}
	delete(repo.stored, positionID)
	return nil
}

var _ interfaces.PositionInitialStopLossRepository = (*fakeOverrides)(nil)

type fakeEntries struct {
	entries map[string]position.PositionEntry
	err     error
}

func (lookup *fakeEntries) FindEntry(_ context.Context, positionID string) (position.PositionEntry, bool, error) {
	if lookup.err != nil {
		return position.PositionEntry{}, false, lookup.err
	}
	entry, found := lookup.entries[positionID]
	return entry, found, nil
}

var _ interfaces.PositionEntryLookup = (*fakeEntries)(nil)

func knownEntries() *fakeEntries {
	return &fakeEntries{entries: map[string]position.PositionEntry{riskPositionID: {Side: position.Long, AvgOpenPrice: "0.9600"}}}
}

const overrideNow = int64(1_791_700_000_000)

func manualOverride(entry, stop string) position.InitialStopLossOverride {
	return position.InitialStopLossOverride{PositionID: riskPositionID, EntryPrice: entry, InitialStopLoss: stop, SetAt: 5}
}

func overrideNowFn() int64 { return overrideNow }

// --- Resolver ---

func TestResolver_OverrideHatVorrangVorGespeichertemRisiko(t *testing.T) {
	risks := &fakeRiskRepo{stored: map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000"}}}
	overrides := &fakeOverrides{stored: map[string]position.InitialStopLossOverride{riskPositionID: manualOverride("0.9500", "0.8000")}}
	resolver := appposition.NewRiskResolver(risks, overrideNowFn).WithOverrides(overrides)

	risk, known := resolver.Resolve(context.Background(), openPosition(nil), nil)
	if !known || risk.InitialStopLoss != "0.8000" || risk.EntryPrice != "0.9500" {
		t.Fatalf("risk=%+v known=%v", risk, known)
	}
	if len(risks.saveCall) != 0 {
		t.Error("position_risk darf durch den Override nicht geschrieben werden")
	}
}

func TestResolver_OverrideVorHistorieUndAktuellemSL(t *testing.T) {
	overrides := &fakeOverrides{stored: map[string]position.InitialStopLossOverride{riskPositionID: manualOverride("0.9600", "0.8000")}}
	resolver := appposition.NewRiskResolver(&fakeRiskRepo{}, overrideNowFn).WithOverrides(overrides)

	risk, known := resolver.Resolve(context.Background(), openPosition(nil), []string{"0.9300"})
	if !known || risk.InitialStopLoss != "0.8000" {
		t.Fatalf("risk=%+v known=%v", risk, known)
	}
}

func TestResolver_OverrideWirdLiveGelesenUndNachLoeschenWiederAuto(t *testing.T) {
	risks := &fakeRiskRepo{stored: map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000"}}}
	overrides := &fakeOverrides{}
	resolver := appposition.NewRiskResolver(risks, overrideNowFn).WithOverrides(overrides)
	ctx := context.Background()

	if risk, _ := resolver.Resolve(ctx, openPosition(nil), nil); risk.InitialStopLoss != "0.9000" {
		t.Fatalf("vor Override: %+v", risk)
	}
	_ = overrides.Set(ctx, manualOverride("0.9600", "0.8000"))
	if risk, _ := resolver.Resolve(ctx, openPosition(nil), nil); risk.InitialStopLoss != "0.8000" {
		t.Fatalf("mit Override (kein Cache): %+v", risk)
	}
	_ = overrides.Delete(ctx, riskPositionID)
	if risk, _ := resolver.Resolve(ctx, openPosition(nil), nil); risk.InitialStopLoss != "0.9000" {
		t.Fatalf("nach Delete: %+v", risk)
	}
}

func TestResolver_OverrideLesefehlerIstUnbekannt(t *testing.T) {
	overrides := &fakeOverrides{getErr: errors.New("db kaputt")}
	resolver := appposition.NewRiskResolver(&fakeRiskRepo{}, overrideNowFn).WithOverrides(overrides)
	if _, known := resolver.Resolve(context.Background(), openPosition(nil), []string{"0.9"}); known {
		t.Error("bei Lesefehler darf das Risiko nicht als bekannt gelten")
	}
}

func TestResolver_ForgetPositionLeertNegativCache(t *testing.T) {
	history := &countingHistory{}
	resolver := appposition.NewRiskResolver(&fakeRiskRepo{}, overrideNowFn).WithHistory(history)
	ctx := context.Background()
	resolver.BeginRun([]string{riskPositionID})
	resolver.Resolve(ctx, openPosition(nil), nil) // keine Treffer -> Negativ-Cache
	resolver.BeginRun([]string{riskPositionID})
	resolver.Resolve(ctx, openPosition(nil), nil)
	if history.calls != 1 {
		t.Fatalf("Negativ-Cache greift nicht: calls=%d", history.calls)
	}
	resolver.ForgetPosition(riskPositionID)
	resolver.BeginRun([]string{riskPositionID})
	resolver.Resolve(ctx, openPosition(nil), nil)
	if history.calls != 2 {
		t.Errorf("nach ForgetPosition muss neu gefragt werden: calls=%d", history.calls)
	}
}

type countingHistory struct{ calls int }

func (history *countingHistory) FetchStopOrders(context.Context, string, int64, int64) ([]position.StopOrderRecord, error) {
	history.calls++
	return nil, nil
}

var _ interfaces.StopOrderHistorySource = (*countingHistory)(nil)

// --- GetOpenPositions: initialStopLossSource und Level ---

func TestGetOpenPositions_QuelleManualMitOverrideEntryUndLevels(t *testing.T) {
	risks := &fakeRiskRepo{}
	overrides := &fakeOverrides{stored: map[string]position.InitialStopLossOverride{riskPositionID: manualOverride("1.0000", "0.9000")}}
	useCase := appposition.NewGetOpenPositionsUseCase(sourceWith(openPosition(nil))).
		WithRisk(risks, &fakeCandles{}, overrideNowFn).WithInitialStopLossOverrides(overrides)

	overview, err := useCase.Execute(context.Background())
	if err != nil || len(overview.Positions) != 1 {
		t.Fatalf("err=%v", err)
	}
	entry := overview.Positions[0]
	if entry.InitialStopLossSource == nil || *entry.InitialStopLossSource != "MANUAL" {
		t.Fatalf("Source = %v", entry.InitialStopLossSource)
	}
	if entry.InitialStopLoss == nil || *entry.InitialStopLoss != "0.9000" || entry.RiskDistance == nil || *entry.RiskDistance != "0.1" {
		t.Errorf("SL=%v Distanz=%v", entry.InitialStopLoss, entry.RiskDistance)
	}
	if len(entry.Levels) != 2 || entry.Levels[0].Price != "1.2" || entry.Levels[1].Price != "1.3" {
		t.Errorf("Level müssen auf Override-Entry 1.0000 beruhen: %+v", entry.Levels)
	}
}

func TestGetOpenPositions_QuelleAutoUndNil(t *testing.T) {
	risks := &fakeRiskRepo{stored: map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000"}}}
	useCase := appposition.NewGetOpenPositionsUseCase(sourceWith(openPosition(nil))).
		WithRisk(risks, &fakeCandles{}, overrideNowFn).WithInitialStopLossOverrides(&fakeOverrides{})
	overview, _ := useCase.Execute(context.Background())
	if source := overview.Positions[0].InitialStopLossSource; source == nil || *source != "AUTO" {
		t.Errorf("bekanntes Risiko ohne Override = %v, erwartet AUTO", source)
	}

	unknown := appposition.NewGetOpenPositionsUseCase(sourceWith(openPosition(nil))).
		WithRisk(&fakeRiskRepo{}, &fakeCandles{}, overrideNowFn).WithInitialStopLossOverrides(&fakeOverrides{})
	overview, _ = unknown.Execute(context.Background())
	if overview.Positions[0].InitialStopLossSource != nil {
		t.Errorf("unbekanntes Risiko muss nil sein: %v", *overview.Positions[0].InitialStopLossSource)
	}
}

// --- SetInitialStopLossUseCase ---

func TestSet_Erfolg(t *testing.T) {
	overrides := &fakeOverrides{}
	useCase := appposition.NewSetInitialStopLossUseCase(knownEntries(), overrides,
		appposition.NewRiskResolver(&fakeRiskRepo{}, overrideNowFn), overrideNowFn)

	result, err := useCase.Execute(context.Background(), riskPositionID, "0.9000")
	if err != nil {
		t.Fatal(err)
	}
	want := position.InitialStopLossOverride{PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000", SetAt: overrideNow}
	if result.Override != want || result.RiskDistance != "0.06" {
		t.Errorf("result = %+v", result)
	}
	if overrides.stored[riskPositionID] != want {
		t.Errorf("nicht gespeichert: %+v", overrides.stored)
	}
}

func TestSet_Fehler(t *testing.T) {
	boom := errors.New("quelle kaputt")
	tests := []struct {
		name   string
		source interfaces.PositionEntryLookup
		id     string
		raw    string
		check  func(error) bool
	}{
		{"Position unbekannt", knownEntries(), "9999", "0.9", func(err error) bool {
			var target *chart.PositionNotFoundError
			return errors.As(err, &target)
		}},
		{"SL gleich Entry", knownEntries(), riskPositionID, "0.96", func(err error) bool {
			var target *position.StopLossAtEntryError
			return errors.As(err, &target)
		}},
		{"falsche Seite", knownEntries(), riskPositionID, "1.0", func(err error) bool {
			var target *position.StopLossWrongSideError
			return errors.As(err, &target)
		}},
		{"keine Zahl", knownEntries(), riskPositionID, "x", func(err error) bool {
			return err != nil && !errors.Is(err, boom) // *decimal.InvalidDecimalError, siehe Domain-Test
		}},
		{"Quellfehler", &fakeEntries{err: boom}, riskPositionID, "0.9", func(err error) bool { return errors.Is(err, boom) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			overrides := &fakeOverrides{}
			useCase := appposition.NewSetInitialStopLossUseCase(test.source, overrides,
				appposition.NewRiskResolver(&fakeRiskRepo{}, overrideNowFn), overrideNowFn)
			_, err := useCase.Execute(context.Background(), test.id, test.raw)
			if !test.check(err) {
				t.Fatalf("err = %v", err)
			}
			if len(overrides.stored) != 0 {
				t.Error("bei Fehler darf nichts gespeichert werden")
			}
		})
	}
}

// --- ClearInitialStopLossUseCase ---

func TestClear_LoeschtIdempotentUndVergisstNegativCache(t *testing.T) {
	overrides := &fakeOverrides{stored: map[string]position.InitialStopLossOverride{riskPositionID: manualOverride("0.96", "0.9")}}
	history := &countingHistory{}
	resolver := appposition.NewRiskResolver(&fakeRiskRepo{}, overrideNowFn).WithHistory(history)
	ctx := context.Background()
	resolver.BeginRun([]string{riskPositionID})
	resolver.Resolve(ctx, openPosition(nil), nil)
	useCase := appposition.NewClearInitialStopLossUseCase(overrides, resolver)

	for range 2 {
		if err := useCase.Execute(ctx, riskPositionID); err != nil {
			t.Fatal(err)
		}
	}
	if _, found := overrides.stored[riskPositionID]; found {
		t.Error("Override noch vorhanden")
	}
	resolver.BeginRun([]string{riskPositionID})
	resolver.Resolve(ctx, openPosition(nil), nil)
	if history.calls != 2 {
		t.Errorf("Negativ-Cache nicht geleert: calls=%d", history.calls)
	}
}

func TestClear_RepositoryFehlerWirdDurchgereicht(t *testing.T) {
	boom := errors.New("db kaputt")
	useCase := appposition.NewClearInitialStopLossUseCase(&fakeOverrides{deleteErr: boom},
		appposition.NewRiskResolver(&fakeRiskRepo{}, overrideNowFn))
	if err := useCase.Execute(context.Background(), riskPositionID); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}
