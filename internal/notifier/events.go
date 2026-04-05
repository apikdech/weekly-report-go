package notifier

import "time"

// NotificationEvent is the base interface for all notification events
type NotificationEvent interface {
	Type() string
	Timestamp() time.Time
}

// StartEvent is emitted when report generation begins
type StartEvent struct {
	WeekRange string
	EventTime time.Time
}

func (e *StartEvent) Type() string         { return "start" }
func (e *StartEvent) Timestamp() time.Time { return e.EventTime }

// FailedEvent is emitted when pipeline fails
type FailedEvent struct {
	WeekRange string
	Error     error
	EventTime time.Time
}

func (e *FailedEvent) Type() string         { return "failed" }
func (e *FailedEvent) Timestamp() time.Time { return e.EventTime }

// FinishedEvent is emitted when report is successfully uploaded
type FinishedEvent struct {
	WeekRange  string
	DocID      string
	DocURL     string
	ReportPath string
	EventTime  time.Time
}

func (e *FinishedEvent) Type() string         { return "finished" }
func (e *FinishedEvent) Timestamp() time.Time { return e.EventTime }

// EventHandler handles notification events
type EventHandler interface {
	Handle(event NotificationEvent)
	Supports(eventType string) bool
}

// EventEmitter dispatches events to registered handlers
type EventEmitter struct {
	handlers []EventHandler
}

// NewEventEmitter creates a new event emitter
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		handlers: make([]EventHandler, 0),
	}
}

// Register adds a handler to the emitter
func (e *EventEmitter) Register(handler EventHandler) {
	e.handlers = append(e.handlers, handler)
}

// Emit sends an event to all registered handlers that support it
func (e *EventEmitter) Emit(event NotificationEvent) {
	for _, handler := range e.handlers {
		if handler.Supports(event.Type()) {
			handler.Handle(event)
		}
	}
}
