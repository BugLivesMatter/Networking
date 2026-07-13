package hub

import (
	"sync"
	"time"
)

type Event struct {
	Type       string      `json:"type"`
	IncidentID string      `json:"incidentId"`
	Data       interface{} `json:"data,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[chan Event]struct{}
	closed      bool
}

func New() *Hub {
	return &Hub{subscribers: make(map[chan Event]struct{})}
}

func (h *Hub) Subscribe() (<-chan Event, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	channel := make(chan Event, 32)
	if h.closed {
		close(channel)
		return channel, func() {}
	}
	h.subscribers[channel] = struct{}{}
	var once sync.Once
	return channel, func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			if _, ok := h.subscribers[channel]; ok {
				delete(h.subscribers, channel)
				close(channel)
			}
		})
	}
}

func (h *Hub) Publish(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.closed {
		return
	}
	for channel := range h.subscribers {
		select {
		case channel <- event:
		default:
			// A slow browser must not block incident writes. It will refresh the
			// canonical state after the next delivered notification.
		}
	}
}

func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for channel := range h.subscribers {
		close(channel)
		delete(h.subscribers, channel)
	}
}
