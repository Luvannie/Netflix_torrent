package events

import (
	"encoding/json"
	"sync"
)

type Bus struct {
	mu          sync.RWMutex
	subscribers map[string][]*subscriber
}

type subscriber struct {
	ch     chan Message
	closed bool
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[string][]*subscriber),
	}
}

func (b *Bus) Subscribe(destination string) (<-chan Message, func()) {
	b.mu.Lock()
	ch := make(chan Message, 1)
	sub := &subscriber{ch: ch}
	b.subscribers[destination] = append(b.subscribers[destination], sub)
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		subs := b.subscribers[destination]
		for i, s := range subs {
			if s == sub {
				b.subscribers[destination] = append(subs[:i], subs[i+1:]...)
				s.closed = true
				break
			}
		}
		b.mu.Unlock()
	}

	return ch, cancel
}

func (b *Bus) Publish(destination string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := Message{
		Destination: destination,
		Body:        body,
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, s := range b.subscribers[destination] {
		if s.closed {
			continue
		}
		select {
		case s.ch <- msg:
		default:
		}
	}

	return nil
}

type Message struct {
	Destination string `json:"destination"`
	Body        []byte `json:"body"`
}
