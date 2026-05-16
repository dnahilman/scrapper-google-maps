package logstream

import (
	"encoding/json"
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
	EventLog              EventType = "log"
	EventPlaceScraped     EventType = "place.scraped"
)

// LogEntry is the payload for EventLog events.
type LogEntry struct {
	Level   string         `json:"level"`
	Time    string         `json:"time"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// HubWriter implements io.Writer so it can be added to zerolog as a secondary
// output. Each Write call is expected to be one JSON log line; non-JSON lines
// are silently ignored.
type HubWriter struct {
	hub *Hub
}

func NewHubWriter(hub *Hub) *HubWriter { return &HubWriter{hub: hub} }

func (w *HubWriter) Write(p []byte) (int, error) {
	var raw map[string]any
	if err := json.Unmarshal(p, &raw); err != nil {
		return len(p), nil
	}
	entry := LogEntry{
		Level:   strField(raw, "level"),
		Time:    strField(raw, "time"),
		Message: strField(raw, "message"),
	}
	fields := make(map[string]any, len(raw))
	for k, v := range raw {
		if k != "level" && k != "time" && k != "message" {
			fields[k] = v
		}
	}
	if len(fields) > 0 {
		entry.Fields = fields
	}
	w.hub.Broadcast(EventLog, entry)
	return len(p), nil
}

func strField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

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
