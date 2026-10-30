package price_test

import (
	"sync"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/price"
)

func receive(t *testing.T, channel <-chan int) int {
	t.Helper()
	select {
	case value, ok := <-channel:
		if !ok {
			t.Fatal("Kanal unerwartet geschlossen")
		}
		return value
	case <-time.After(time.Second):
		t.Fatal("kein Wert empfangen")
	}
	return 0
}

func TestBrokerFansOutToAllSubscribers(t *testing.T) {
	broker := price.NewBroker[int](4)
	first, _ := broker.Subscribe()
	second, _ := broker.Subscribe()

	broker.Publish(7)

	if got := receive(t, first); got != 7 {
		t.Fatalf("first got %d", got)
	}
	if got := receive(t, second); got != 7 {
		t.Fatalf("second got %d", got)
	}
}

func TestBrokerDoesNotBlockOnSlowSubscriber(t *testing.T) {
	broker := price.NewBroker[int](1)
	slow, _ := broker.Subscribe()
	fast, _ := broker.Subscribe()

	done := make(chan struct{})
	go func() {
		for value := 1; value <= 10; value++ {
			broker.Publish(value) // slow liest nie
			receive(t, fast)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blockiert wegen langsamem Abonnenten")
	}
	if got := receive(t, slow); got != 1 {
		t.Fatalf("langsamer Abonnent behält den ersten gepufferten Wert, got %d", got)
	}
}

func TestBrokerUnsubscribeClosesChannel(t *testing.T) {
	broker := price.NewBroker[int](2)
	channel, unsubscribe := broker.Subscribe()

	unsubscribe()
	unsubscribe() // idempotent

	select {
	case _, ok := <-channel:
		if ok {
			t.Fatal("Kanal sollte geschlossen und leer sein")
		}
	case <-time.After(time.Second):
		t.Fatal("Kanal nicht geschlossen")
	}
	broker.Publish(1) // darf nach Abmeldung nicht panicken
}

func TestBrokerNewSubscriberReceivesLatestSnapshot(t *testing.T) {
	broker := price.NewBroker[int](2)
	broker.Publish(1)
	broker.Publish(2)

	channel, _ := broker.Subscribe()

	if got := receive(t, channel); got != 2 {
		t.Fatalf("want letzter Stand 2, got %d", got)
	}
}

func TestBrokerWithoutSnapshotSendsNothingOnSubscribe(t *testing.T) {
	broker := price.NewBroker[int](2)
	channel, _ := broker.Subscribe()

	select {
	case value := <-channel:
		t.Fatalf("unerwarteter Wert %d", value)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestBrokerIsRaceFreeUnderConcurrency(_ *testing.T) {
	broker := price.NewBroker[int](2)
	var group sync.WaitGroup
	for range 8 {
		group.Add(2)
		go func() {
			defer group.Done()
			_, unsubscribe := broker.Subscribe()
			unsubscribe()
		}()
		go func() {
			defer group.Done()
			for value := range 50 {
				broker.Publish(value)
			}
		}()
	}
	group.Wait()
}
