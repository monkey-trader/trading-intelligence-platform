package position

import (
	"context"
	"fmt"
	"strconv"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
)

const (
	incomePath = "/openApi/swap/v2/user/income"
	// incomePageSize ist das BingX-Maximum; eine kürzere Seite beendet das Paging.
	incomePageSize = 1000
)

// BingXIncomeSource fragt Income live bei BingX ab.
type BingXIncomeSource struct{ client bingx.GetClient }

// NewBingXIncomeSource baut die Source.
func NewBingXIncomeSource(client bingx.GetClient) *BingXIncomeSource {
	return &BingXIncomeSource{client: client}
}

// FetchIncome blättert per startTime=max(time) (inklusiv, damit Einträge mit gleichem Zeitstempel über die
// Seitengrenze nicht verloren gehen) und dedupliziert über tranId, bis eine Seite nicht voll ist.
// Kommt eine volle Seite ohne Fortschritt (max(time) == startTime), bricht es mit einem Fehler ab.
func (source *BingXIncomeSource) FetchIncome(ctx context.Context, from, to int64) ([]domainposition.Income, error) {
	if from > to {
		return nil, &domainposition.InvalidQueryRangeError{From: from, To: to}
	}
	var collected []domainposition.Income
	seen := map[string]struct{}{}
	for pageStart := from; pageStart <= to; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		data, err := source.client.Get(ctx, incomePath, []bingx.QueryParam{
			{Name: "startTime", Value: strconv.FormatInt(pageStart, 10)},
			{Name: "endTime", Value: strconv.FormatInt(to, 10)},
			{Name: "limit", Value: strconv.Itoa(incomePageSize)},
		})
		if err != nil {
			return nil, err
		}
		page, err := mapIncomeResponse(data)
		if err != nil {
			return nil, err
		}
		for _, entry := range page {
			if _, duplicate := seen[entry.TranID]; duplicate {
				continue
			}
			seen[entry.TranID] = struct{}{}
			collected = append(collected, entry)
		}
		if len(page) < incomePageSize {
			break
		}
		nextStart := latestIncomeTime(page)
		if nextStart <= pageStart {
			return nil, fmt.Errorf("income: volle Seite ohne Fortschritt bei startTime=%d, Paging nicht möglich", pageStart)
		}
		pageStart = nextStart
	}
	return collected, nil
}

// latestIncomeTime liefert die größte Zeit einer nicht leeren Seite.
func latestIncomeTime(page []domainposition.Income) int64 {
	latest := page[0].Time
	for _, entry := range page {
		latest = max(latest, entry.Time)
	}
	return latest
}
