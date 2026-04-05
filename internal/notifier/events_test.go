package notifier

import (
	"errors"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func TestStartEvent_ImplementsInterface(t *testing.T) {
	event := &StartEvent{
		WeekRange: "22 March 2026 - 28 March 2026",
		EventTime: time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC),
	}

	var _ NotificationEvent = event

	if event.Type() != EventTypeStart {
		t.Errorf("expected Type() %q, got %q", EventTypeStart, event.Type())
	}

	if !event.Timestamp().Equal(time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC)) {
		t.Errorf("Timestamp() returned unexpected value")
	}
}

func TestProcessingEvent_ImplementsInterface(t *testing.T) {
	event := &ProcessingEvent{
		WeekRange: "22 March 2026 - 28 March 2026",
		Stage:     "Rendering markdown report",
		EventTime: time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC),
	}

	var _ NotificationEvent = event

	if event.Type() != EventTypeProcessing {
		t.Errorf("expected Type() %q, got %q", EventTypeProcessing, event.Type())
	}

	if !event.Timestamp().Equal(time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC)) {
		t.Errorf("Timestamp() returned unexpected value")
	}
}

func TestFailedEvent_ImplementsInterface(t *testing.T) {
	event := &FailedEvent{
		WeekRange: "22 March 2026 - 28 March 2026",
		Error:     errTest,
		EventTime: time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC),
	}

	var _ NotificationEvent = event

	if event.Type() != EventTypeFailed {
		t.Errorf("expected Type() %q, got %q", EventTypeFailed, event.Type())
	}

	if !event.Timestamp().Equal(time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC)) {
		t.Errorf("Timestamp() returned unexpected value")
	}
}

func TestFinishedEvent_ImplementsInterface(t *testing.T) {
	event := &FinishedEvent{
		WeekRange:  "22 March 2026 - 28 March 2026",
		DocID:      "abc123",
		DocURL:     "https://docs.google.com/document/d/abc123/edit",
		ReportPath: "/tmp/report.md",
		EventTime:  time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC),
	}

	var _ NotificationEvent = event

	if event.Type() != EventTypeFinished {
		t.Errorf("expected Type() %q, got %q", EventTypeFinished, event.Type())
	}

	if !event.Timestamp().Equal(time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC)) {
		t.Errorf("Timestamp() returned unexpected value")
	}
}

// mockHandler is a test implementation of EventHandler
type mockHandler struct {
	supportedTypes []EventType
	handledEvents  []NotificationEvent
}

func (m *mockHandler) Handle(event NotificationEvent) {
	m.handledEvents = append(m.handledEvents, event)
}

func (m *mockHandler) Supports(eventType EventType) bool {
	for _, t := range m.supportedTypes {
		if t == eventType {
			return true
		}
	}
	return false
}

func TestEventEmitter_RegisterAndEmit(t *testing.T) {
	emitter := NewEventEmitter()
	handler := &mockHandler{supportedTypes: []EventType{EventTypeStart}}

	emitter.Register(handler)

	event := &StartEvent{
		WeekRange: "22 March 2026 - 28 March 2026",
		EventTime: time.Now(),
	}

	emitter.Emit(event)

	if len(handler.handledEvents) != 1 {
		t.Errorf("expected 1 handled event, got %d", len(handler.handledEvents))
	}
}

func TestEventEmitter_MultipleHandlers(t *testing.T) {
	emitter := NewEventEmitter()
	startHandler := &mockHandler{supportedTypes: []EventType{EventTypeStart}}
	allHandler := &mockHandler{supportedTypes: []EventType{EventTypeStart, EventTypeFailed, EventTypeFinished}}

	emitter.Register(startHandler)
	emitter.Register(allHandler)

	startEvent := &StartEvent{WeekRange: "Week 1", EventTime: time.Now()}
	failedEvent := &FailedEvent{WeekRange: "Week 1", Error: errTest, EventTime: time.Now()}

	emitter.Emit(startEvent)
	emitter.Emit(failedEvent)

	// startHandler should only receive start event
	if len(startHandler.handledEvents) != 1 {
		t.Errorf("startHandler: expected 1 event, got %d", len(startHandler.handledEvents))
	}

	// allHandler should receive both events
	if len(allHandler.handledEvents) != 2 {
		t.Errorf("allHandler: expected 2 events, got %d", len(allHandler.handledEvents))
	}
}
