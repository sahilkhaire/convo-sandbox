package core

import (
	"encoding/json"
	"sync"
)

type EventType string

const (
	EventNewMessage  EventType = "new_message"
	EventDelivery    EventType = "delivery"
	EventDataCleared EventType = "data_cleared"
)

type SSEEvent struct {
	Type EventType       `json:"type"`
	Data json.RawMessage `json:"data"`
}

type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan SSEEvent]struct{}
}

func NewSSEHub() *SSEHub {
	return &SSEHub{clients: make(map[chan SSEEvent]struct{})}
}

func (h *SSEHub) Subscribe() chan SSEEvent {
	ch := make(chan SSEEvent, 32)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *SSEHub) Unsubscribe(ch chan SSEEvent) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *SSEHub) Broadcast(eventType EventType, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	ev := SSEEvent{Type: eventType, Data: payload}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- ev:
		default:
		}
	}
}
