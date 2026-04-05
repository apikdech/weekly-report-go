package notifier

import "time"

// EventType identifies a notification event kind for handler routing.
type EventType string

const (
	EventTypeStart      EventType = "start"
	EventTypeProcessing EventType = "processing"
	EventTypeFailed     EventType = "failed"
	EventTypeFinished   EventType = "finished"
)

// NotificationEvent is the base interface for all notification events
type NotificationEvent interface {
	Type() EventType
	Timestamp() time.Time
}

// StartEvent is emitted when report generation begins
type StartEvent struct {
	WeekRange string
	EventTime time.Time
}

func (e *StartEvent) Type() EventType      { return EventTypeStart }
func (e *StartEvent) Timestamp() time.Time { return e.EventTime }

// ProcessingEvent is emitted between major pipeline phases (e.g. after data collection, before upload).
type ProcessingEvent struct {
	WeekRange string
	Stage     string
	EventTime time.Time
}

func (e *ProcessingEvent) Type() EventType      { return EventTypeProcessing }
func (e *ProcessingEvent) Timestamp() time.Time { return e.EventTime }

// FailedEvent is emitted when pipeline fails
type FailedEvent struct {
	WeekRange string
	Error     error
	EventTime time.Time
}

func (e *FailedEvent) Type() EventType      { return EventTypeFailed }
func (e *FailedEvent) Timestamp() time.Time { return e.EventTime }

// FinishedEvent is emitted when report is successfully uploaded
type FinishedEvent struct {
	WeekRange  string
	DocID      string
	DocURL     string
	ReportPath string
	EventTime  time.Time
}

func (e *FinishedEvent) Type() EventType      { return EventTypeFinished }
func (e *FinishedEvent) Timestamp() time.Time { return e.EventTime }

// EventHandler handles notification events
type EventHandler interface {
	Handle(event NotificationEvent)
	Supports(eventType EventType) bool
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
