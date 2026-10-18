// Package scheduler führt einen Job periodisch aus, ohne dass sich Läufe überlappen.
package scheduler

import "context"

// Config konfiguriert den Scheduler.
type Config struct {
	// Job wird sofort beim Start und danach bei jedem Tick ausgeführt.
	Job func(ctx context.Context) error
	// Ticks liefert die Takte (produktiv time.Ticker.C, im Test ein eigener Channel).
	Ticks <-chan struct{}
	// OnError wird bei Job-Fehlern gerufen (optional).
	OnError func(err error)
	// OnSkip wird gerufen, wenn ein Tick wegen laufendem Job übersprungen wurde (optional).
	OnSkip func()
}

// Scheduler führt den Job aus.
type Scheduler struct{ config Config }

// New baut den Scheduler.
func New(config Config) *Scheduler { return &Scheduler{config: config} }

// Run blockiert bis ctx beendet ist und wartet auf einen laufenden Job.
func (scheduler *Scheduler) Run(ctx context.Context) {
	var running chan struct{} // offen, solange ein Job läuft
	start := func() {
		running = make(chan struct{})
		go func(finished chan struct{}) {
			defer close(finished)
			if err := scheduler.config.Job(ctx); err != nil && scheduler.config.OnError != nil {
				scheduler.config.OnError(err)
			}
		}(running)
	}
	isRunning := func() bool {
		if running == nil {
			return false
		}
		select {
		case <-running:
			return false
		default:
			return true
		}
	}

	if ctx.Err() != nil {
		return
	}
	start()
	for {
		select {
		case <-ctx.Done():
			<-running
			return
		case <-scheduler.config.Ticks:
			if ctx.Err() != nil {
				<-running
				return
			}
			if isRunning() {
				if scheduler.config.OnSkip != nil {
					scheduler.config.OnSkip()
				}
				continue
			}
			start()
		}
	}
}
