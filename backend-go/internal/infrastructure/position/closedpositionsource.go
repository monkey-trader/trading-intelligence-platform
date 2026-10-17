package position

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
)

// closedPositionHistoryPath ist der BingX-Endpunkt für die Positions-Historie.
const closedPositionHistoryPath = "/openApi/swap/v1/trade/positionHistory"

// Konstanten der Abfrage.
const (
	// maxQueryRangeMillis ist die maximale Abfragespanne in Millisekunden (siehe Domain).
	maxQueryRangeMillis = int64(domainposition.MaxQueryRange / time.Millisecond)
	// closedPositionPageSize ist die Seitengröße; eine kürzere Seite beendet das Paging.
	closedPositionPageSize = 100
	// maxClosedPositionPages ist die feste Seitenobergrenze gegen Endlosschleifen.
	maxClosedPositionPages = 1000
)

// Sentinel-Fehler der Source, per errors.Is erkennbar.
var (
	// ErrQueryRangeTooLarge: Der Bereich ist größer als 89 Tage.
	ErrQueryRangeTooLarge = errors.New("abfragebereich größer als 89 Tage")
	// ErrPageLimitExceeded: Das Seitenlimit wurde erreicht.
	ErrPageLimitExceeded = errors.New("seitenlimit überschritten")
)

// BingXClosedPositionSource fragt die Positions-Historie live bei BingX ab.
type BingXClosedPositionSource struct {
	client bingx.GetClient
}

// NewBingXClosedPositionSource baut die Source.
func NewBingXClosedPositionSource(client bingx.GetClient) *BingXClosedPositionSource {
	return &BingXClosedPositionSource{client: client}
}

// FetchClosedPositions ruft GET /openApi/swap/v1/trade/positionHistory auf und übersetzt Fehler.
// Bricht ab, wenn der Bereich größer als 89 Tage ist oder mehr als 1000 Seiten nötig wären. pageIndex zählt ab 0.
func (source *BingXClosedPositionSource) FetchClosedPositions(ctx context.Context, symbol string, from, to int64) ([]domainposition.ClosedPosition, error) {
	if from > to {
		return nil, &domainposition.InvalidQueryRangeError{From: from, To: to}
	}
	if to-from > maxQueryRangeMillis {
		return nil, fmt.Errorf("%w: von %d bis %d, erlaubt sind %d Tage", ErrQueryRangeTooLarge, from, to, int(domainposition.MaxQueryRange.Hours()/24))
	}

	var collected []domainposition.ClosedPosition
	for pageIndex := 0; pageIndex < maxClosedPositionPages; pageIndex++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		data, err := source.client.Get(ctx, closedPositionHistoryPath, []bingx.QueryParam{
			{Name: "symbol", Value: symbol},
			{Name: "startTs", Value: strconv.FormatInt(from, 10)},
			{Name: "endTs", Value: strconv.FormatInt(to, 10)},
			{Name: "pageIndex", Value: strconv.Itoa(pageIndex)},
			{Name: "pageSize", Value: strconv.Itoa(closedPositionPageSize)},
		})
		if err != nil {
			return nil, toClosedUnavailable(err)
		}
		page, err := mapClosedPositionsResponse(data)
		if err != nil {
			return nil, toClosedUnavailable(err)
		}
		collected = append(collected, page...)
		if len(page) < closedPositionPageSize {
			return collected, nil
		}
	}
	return nil, fmt.Errorf("%w: mehr als %d Seiten für %s von %d bis %d, abgebrochen", ErrPageLimitExceeded, maxClosedPositionPages, symbol, from, to)
}

// toClosedUnavailable übersetzt bekannte Fehler in *ClosedPositionsUnavailableError; unbekannte gehen unverändert durch.
func toClosedUnavailable(err error) error {
	reason, known := classifyUnavailable(err)
	if !known {
		return err
	}
	return &domainposition.ClosedPositionsUnavailableError{Reason: reason, Cause: err}
}
