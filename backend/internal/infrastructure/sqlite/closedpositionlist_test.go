package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

var _ interfaces.ClosedPositionLister = (*PositionRepository)(nil)

func seedClosed(t *testing.T, repo *PositionRepository, id, symbol string, closeTime int64, fullyClosed bool) {
	t.Helper()
	closed := samplePosition()
	closed.PositionID, closed.Symbol, closed.CloseTime, closed.FullyClosed = id, symbol, closeTime, fullyClosed
	closed.OpenTime = 1
	if err := repo.UpsertClosedPosition(context.Background(), closed); err != nil {
		t.Fatalf("Upsert %s: %v", id, err)
	}
}

func listIDs(t *testing.T, repo *PositionRepository, input position.ClosedPositionQueryInput) []string {
	t.Helper()
	query, err := position.NewClosedPositionQuery(input)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	listed, err := repo.ListClosedPositions(context.Background(), query)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ids := []string{}
	for _, item := range listed {
		ids = append(ids, item.PositionID)
	}
	return ids
}

func sameIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func newListRepo(t *testing.T) *PositionRepository {
	now := time.UnixMilli(1_800_000_000_000)
	return NewPositionRepository(openMemory(t), fixedClock(&now))
}

func i64(value int64) *int64 { return &value }
func intp(value int) *int    { return &value }

func TestListClosed_SchliesstPlatzhalterUndCloseTimeNullAus(t *testing.T) {
	repo := newListRepo(t)
	seedClosed(t, repo, "1", "SUI-USDT", 5000, true)
	seedClosed(t, repo, "2", "SUI-USDT", 6000, false)
	seedClosed(t, repo, "3", "SUI-USDT", 0, true)
	if got := listIDs(t, repo, position.ClosedPositionQueryInput{}); !sameIDs(got, []string{"1"}) {
		t.Errorf("ids = %v, erwartet [1]", got)
	}
}

func TestListClosed_SortiertAbsteigendNachCloseTime(t *testing.T) {
	repo := newListRepo(t)
	seedClosed(t, repo, "1", "SUI-USDT", 1000, true)
	seedClosed(t, repo, "2", "SUI-USDT", 3000, true)
	seedClosed(t, repo, "3", "SUI-USDT", 2000, true)
	if got := listIDs(t, repo, position.ClosedPositionQueryInput{}); !sameIDs(got, []string{"2", "3", "1"}) {
		t.Errorf("ids = %v", got)
	}
}

func TestListClosed_FromUndToSindInklusiv(t *testing.T) {
	repo := newListRepo(t)
	for index, closeTime := range []int64{999, 1000, 2000, 2001} {
		seedClosed(t, repo, string(rune('1'+index)), "SUI-USDT", closeTime, true)
	}
	got := listIDs(t, repo, position.ClosedPositionQueryInput{From: i64(1000), To: i64(2000)})
	if !sameIDs(got, []string{"3", "2"}) {
		t.Errorf("ids = %v, erwartet [3 2]", got)
	}
}

func TestListClosed_FiltertNachSymbol(t *testing.T) {
	repo := newListRepo(t)
	seedClosed(t, repo, "1", "SUI-USDT", 1000, true)
	seedClosed(t, repo, "2", "BTC-USDT", 2000, true)
	if got := listIDs(t, repo, position.ClosedPositionQueryInput{Symbol: "BTC-USDT"}); !sameIDs(got, []string{"2"}) {
		t.Errorf("ids = %v", got)
	}
}

func TestListClosed_LimitGenauUndEinsDarunter(t *testing.T) {
	repo := newListRepo(t)
	for index := int64(1); index <= 3; index++ {
		seedClosed(t, repo, string(rune('0'+index)), "SUI-USDT", index*1000, true)
	}
	if got := listIDs(t, repo, position.ClosedPositionQueryInput{Limit: intp(3)}); len(got) != 3 {
		t.Errorf("limit 3: %v", got)
	}
	if got := listIDs(t, repo, position.ClosedPositionQueryInput{Limit: intp(2)}); !sameIDs(got, []string{"3", "2"}) {
		t.Errorf("limit 2: %v", got)
	}
}

func TestListClosed_LeereTabelleLiefertLeereListe(t *testing.T) {
	repo := newListRepo(t)
	query, _ := position.NewClosedPositionQuery(position.ClosedPositionQueryInput{})
	listed, err := repo.ListClosedPositions(context.Background(), query)
	if err != nil || listed == nil || len(listed) != 0 {
		t.Errorf("listed=%v err=%v, erwartet leer und nicht nil", listed, err)
	}
}

func TestListClosed_MapptAlleFelder(t *testing.T) {
	repo := newListRepo(t)
	want := samplePosition()
	if err := repo.UpsertClosedPosition(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	query, _ := position.NewClosedPositionQuery(position.ClosedPositionQueryInput{})
	listed, err := repo.ListClosedPositions(context.Background(), query)
	if err != nil || len(listed) != 1 || listed[0] != want {
		t.Errorf("listed=%+v err=%v, erwartet %+v", listed, err, want)
	}
}
