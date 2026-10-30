package main

import (
	"context"
	"time"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	appprice "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/price"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/clock"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/config"
)

// priceBrokerBuffer ist der Puffer je Abonnent; ein langsamer Client verpasst nur alte Kurse.
const priceBrokerBuffer = 4

// newPriceStream verdrahtet GET /prices/stream: Broker, Use Case und Poller. Ohne BingX-Zugang ist alles nil
// (Route nicht registriert). run blockiert bis zum Abbruch des Contexts.
func newPriceStream(
	lookupEnv func(string) string,
	openPositions func(context.Context) (appposition.OpenPositionsOverview, error),
) (subscribe func() (<-chan appprice.PriceSnapshot, func()), run func(context.Context), err error) {
	bingXConfig, configured, err := config.ReadBingXConfig(lookupEnv)
	if err != nil || !configured {
		return nil, nil, err
	}
	client := bingx.NewClient(bingx.ClientOptions{
		APIKey:         bingXConfig.APIKey,
		APISecret:      bingXConfig.APISecret,
		BaseURL:        bingXConfig.BaseURL,
		MaxRetries:     bingXMaxRetries,
		RequestTimeout: bingXRequestTimeout,
		RequestPause:   bingXRequestPause,
		InitialBackoff: bingXInitialBackoff,
	})
	broker := appprice.NewBroker[appprice.PriceSnapshot](priceBrokerBuffer)
	useCase := appprice.NewStreamPricesUseCase(appprice.StreamPricesConfig{
		Source:        bingx.NewTickerSource(client, clock.SystemClock{}),
		OpenPositions: openPositions,
		Broker:        broker,
		Clock:         clock.SystemClock{},
		Sleep:         sleepWithContext,
		Jitter:        appprice.DefaultJitter,
	})
	return broker.Subscribe, useCase.Run, nil
}

func sleepWithContext(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
