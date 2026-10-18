package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"
)

const waitLimit = 2 * time.Second

func receive[T any](t *testing.T, channel <-chan T, what string) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(waitLimit):
		t.Fatalf("Timeout: %s", what)
		panic("unreachable")
	}
}

func startScheduler(t *testing.T, config Config) (cancel context.CancelFunc, done <-chan struct{}) {
	t.Helper()
	ctx, cancelFunc := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() { New(config).Run(ctx); close(finished) }()
	t.Cleanup(func() { cancelFunc(); <-finished })
	return cancelFunc, finished
}

func TestScheduler_LaeuftSofortUndBeiJedemTick(t *testing.T) {
	runs := make(chan struct{}, 10)
	ticks := make(chan struct{})
	startScheduler(t, Config{Ticks: ticks, Job: func(context.Context) error { runs <- struct{}{}; return nil }})

	receive(t, runs, "Lauf direkt beim Start")
	for tick := 1; tick <= 3; tick++ {
		ticks <- struct{}{}
		receive(t, runs, "Lauf nach Tick")
	}
}

func TestScheduler_ZweiterTickWaehrendLaufWirdUebersprungen(t *testing.T) {
	started := make(chan struct{}, 10)
	release := make(chan struct{})
	skipped := make(chan struct{}, 10)
	ticks := make(chan struct{}, 10)
	startScheduler(t, Config{
		Ticks:  ticks,
		OnSkip: func() { skipped <- struct{}{} },
		Job: func(context.Context) error {
			started <- struct{}{}
			<-release
			return nil
		},
	})
	receive(t, started, "erster Lauf")
	ticks <- struct{}{}
	ticks <- struct{}{}
	receive(t, skipped, "erster übersprungener Tick")
	receive(t, skipped, "zweiter übersprungener Tick")

	close(release)
	select {
	case <-started:
		t.Fatal("übersprungene Ticks dürfen keinen Nachlauf auslösen")
	case <-time.After(50 * time.Millisecond):
	}
	ticks <- struct{}{}
	receive(t, started, "Lauf nach Ende des ersten Laufs")
}

func TestScheduler_JobFehlerWirdGemeldetUndLaufWeiter(t *testing.T) {
	boom := errors.New("boom")
	errs := make(chan error, 10)
	runs := make(chan struct{}, 10)
	ticks := make(chan struct{})
	startScheduler(t, Config{
		Ticks:   ticks,
		OnError: func(err error) { errs <- err },
		Job:     func(context.Context) error { runs <- struct{}{}; return boom },
	})
	receive(t, runs, "Start")
	if err := receive(t, errs, "Fehlermeldung"); !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
	ticks <- struct{}{}
	receive(t, runs, "Lauf nach Fehler")
}

func TestScheduler_ShutdownBrichtLaufendenJobUeberKontextAbUndWartet(t *testing.T) {
	started := make(chan struct{})
	jobEnded := make(chan struct{})
	cancel, done := startScheduler(t, Config{
		Ticks: make(chan struct{}),
		Job: func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			close(jobEnded)
			return ctx.Err()
		},
	})
	receive(t, started, "Job gestartet")
	cancel()
	receive(t, done, "Run kehrt zurück")
	select {
	case <-jobEnded:
	default:
		t.Error("Run kehrte zurück, bevor der Job beendet war")
	}
}

func TestScheduler_ShutdownOhneLaufendenJobUndKeinNeuerLaufDanach(t *testing.T) {
	runs := make(chan struct{}, 10)
	ticks := make(chan struct{}, 1)
	cancel, done := startScheduler(t, Config{Ticks: ticks, Job: func(context.Context) error { runs <- struct{}{}; return nil }})
	receive(t, runs, "Start")
	cancel()
	receive(t, done, "Run kehrt zurück")
	if len(runs) != 0 {
		t.Errorf("unerwartete Läufe nach Shutdown: %d", len(runs))
	}
}

func TestScheduler_BereitsAbgebrochenerKontextStartetKeinenJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ran := false
	finished := make(chan struct{})
	go func() {
		New(Config{Ticks: make(chan struct{}), Job: func(context.Context) error { ran = true; return nil }}).Run(ctx)
		close(finished)
	}()
	receive(t, finished, "Run kehrt zurück")
	if ran {
		t.Error("Job darf bei abgebrochenem Kontext nicht laufen")
	}
}
