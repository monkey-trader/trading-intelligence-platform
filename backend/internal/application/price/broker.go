// Package price enthält die Use Cases für Live-Kurse.
package price

import "sync"

// Broker verteilt Werte an Abonnenten und merkt sich den letzten Wert.
type Broker[T any] struct {
	mutex       sync.Mutex
	buffer      int
	subscribers map[int]chan T
	nextID      int
	last        T
	hasLast     bool
}

// NewBroker legt einen Broker mit gepuffertem Kanal (buffer Werte) je Abonnent an.
func NewBroker[T any](buffer int) *Broker[T] {
	return &Broker[T]{buffer: buffer, subscribers: map[int]chan T{}}
}

// Subscribe liefert einen Kanal (sofort mit dem letzten Wert, falls vorhanden) und eine Abmeldefunktion,
// die den Kanal schließt (idempotent).
func (broker *Broker[T]) Subscribe() (<-chan T, func()) {
	broker.mutex.Lock()
	defer broker.mutex.Unlock()
	size := broker.buffer
	if size < 1 {
		size = 1
	}
	channel := make(chan T, size)
	if broker.hasLast {
		channel <- broker.last
	}
	id := broker.nextID
	broker.nextID++
	broker.subscribers[id] = channel
	var once sync.Once
	return channel, func() {
		once.Do(func() {
			broker.mutex.Lock()
			defer broker.mutex.Unlock()
			delete(broker.subscribers, id)
			close(channel)
		})
	}
}

// Publish verteilt value an alle Abonnenten, blockiert nie (voller Puffer: Wert für diesen Abonnenten verwerfen).
func (broker *Broker[T]) Publish(value T) {
	broker.mutex.Lock()
	defer broker.mutex.Unlock()
	broker.last, broker.hasLast = value, true
	for _, channel := range broker.subscribers {
		select {
		case channel <- value:
		default:
		}
	}
}
