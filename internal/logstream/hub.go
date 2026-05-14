package logstream

import (
	"sync"
	"time"
)

// EventType is the discriminator for WS payloads.
type EventType string

const (
	EventWorkerRegistered EventType = "worker.registered"
	EventWorkerOffline    EventType = "worker.offline"
	EventTaskClaimed      EventType = "task.claimed"
	EventTaskHeartbeat    EventType = "task.heartbeat"
	EventTaskCompleted    EventType = "task.completed"
	EventTaskFailed       EventType = "task.failed"
	EventTaskRequeued     EventType = "task.requeued"
	EventJobUpdated       EventType = "job.updated"
	EventStatsSnapshot    EventType = "stats.snapshot"
)

// Event is the over-the-wire payload broadcast to every connected client.
type Event struct {
	Type    EventType `json:"type"`
	Payload any       `json:"payload,omitempty"`
	At      time.Time `json:"at"`
}

// Hub is a many-to-many fanout for Event values. Goroutine-safe.
//
// Producers call Broadcast() from any code path (queue, reaper, etc).
// Consumers (WS handlers) call Subscribe() to get a buffered channel and
// Unsubscribe() when done — usually via defer.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[chan Event]struct{}
	bufSize     int
}

func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[chan Event]struct{}),
		bufSize:     32,
	}
}

// Subscribe returns a new buffered channel that receives every subsequent
// Broadcast. Slow consumers are dropped silently — Hub will close their
// channel rather than blocking the producer.
func (h *Hub) Subscribe() chan Event {
	ch := make(chan Event, h.bufSize)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe stops delivery and closes the channel. Safe to call multiple
// times.
func (h *Hub) Unsubscribe(ch chan Event) {
	h.mu.Lock()
	if _, ok := h.subscribers[ch]; ok {
		delete(h.subscribers, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// Broadcast delivers ev to every subscriber non-blocking. Subscribers whose
// buffer is full have their channel dropped to avoid stalling other clients.
func (h *Hub) Broadcast(t EventType, payload any) {
	ev := Event{Type: t, Payload: payload, At: time.Now()}
	var stale []chan Event
	h.mu.RLock()
	for ch := range h.subscribers {
		select {
		case ch <- ev:
		default:
			stale = append(stale, ch)
		}
	}
	h.mu.RUnlock()
	if len(stale) > 0 {
		h.mu.Lock()
		for _, ch := range stale {
			if _, ok := h.subscribers[ch]; ok {
				delete(h.subscribers, ch)
				close(ch)
			}
		}
		h.mu.Unlock()
	}
}

// Count returns the number of currently-subscribed clients.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}
