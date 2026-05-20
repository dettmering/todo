// Package events provides a minimal in-process pub/sub bus.
// Webhook delivery (Lastenheft 5.11) will hook in here as a subscriber.
package events

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type EventType string

const (
	TaskCreated         EventType = "task.created"
	TaskUpdated         EventType = "task.updated"
	TaskCompleted       EventType = "task.completed"
	TaskDeleted         EventType = "task.deleted"
	TaskRescheduled     EventType = "task.rescheduled"
	TaskDeadlineChanged EventType = "task.deadline_changed"
	ProjectCreated      EventType = "project.created"
	ProjectUpdated      EventType = "project.updated"
	ProjectDeleted      EventType = "project.deleted"
)

type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      any
}

type Handler func(ctx context.Context, e Event)

type Bus struct {
	mu       sync.RWMutex
	handlers []Handler
}

func NewBus() *Bus { return &Bus{} }

func (b *Bus) Subscribe(h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, h)
}

// Publish fans out to handlers asynchronously so HTTP requests are never
// blocked by slow subscribers (e.g., webhook delivery).
func (b *Bus) Publish(ctx context.Context, t EventType, data any) {
	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers...)
	b.mu.RUnlock()

	e := Event{Type: t, Timestamp: time.Now().UTC(), Data: data}
	for _, h := range handlers {
		go func(h Handler) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("event handler panic", "type", t, "panic", r)
				}
			}()
			h(ctx, e)
		}(h)
	}
}
