package gchat_test

import (
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
	"github.com/apikdech/gws-weekly-report/internal/sources/gchat"
)

var sampleResponse = []byte(`{
  "messages": [
    {
      "name": "spaces/AAQAE4zqbX4/messages/aaa",
      "createTime": "2026-03-22T08:00:00.000000Z",
      "sender": {"name": "users/other", "type": "BOT"},
      "text": "old message"
    },
    {
      "name": "spaces/AAQAE4zqbX4/messages/bbb",
      "createTime": "2026-03-27T08:43:10.336799Z",
      "sender": {"name": "users/102650500894334129637", "type": "BOT"},
      "text": "latest metrics text"
    },
    {
      "name": "spaces/AAQAE4zqbX4/messages/ccc",
      "createTime": "2026-03-25T10:00:00.000000Z",
      "sender": {"name": "users/102650500894334129637", "type": "BOT"},
      "text": "earlier metrics text"
    }
  ]
}`)

func TestPickLatestBySender_Found(t *testing.T) {
	text, err := gchat.PickLatestBySender(sampleResponse, "users/102650500894334129637")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "latest metrics text" {
		t.Errorf("got %q, want %q", text, "latest metrics text")
	}
}

func TestPickLatestBySender_NoMatch(t *testing.T) {
	text, err := gchat.PickLatestBySender(sampleResponse, "users/nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty string for no match, got %q", text)
	}
}

func TestPickLatestBySender_EmptyMessages(t *testing.T) {
	data := []byte(`{"messages":[]}`)
	text, err := gchat.PickLatestBySender(data, "users/102650500894334129637")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}

func TestPickLatestBySender_InvalidJSON(t *testing.T) {
	_, err := gchat.PickLatestBySender([]byte(`not json`), "users/x")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestSource_ContributeWithoutFetch(t *testing.T) {
	s := gchat.NewSource(nil, "test-space", "users/test")
	report := &pipeline.ReportData{}
	err := s.Contribute(report)
	if err == nil {
		t.Fatal("expected error when Contribute called before Fetch, got nil")
	}
}

func TestPickLatestBySender_UnparseableTimestampSkipped(t *testing.T) {
	data := []byte(`{
  "messages": [
    {
      "name": "spaces/X/messages/bad",
      "createTime": "not-a-time",
      "sender": {"name": "users/bot"},
      "text": "bad time message"
    },
    {
      "name": "spaces/X/messages/good",
      "createTime": "2026-03-27T08:00:00Z",
      "sender": {"name": "users/bot"},
      "text": "good message"
    }
  ]
}`)
	text, err := gchat.PickLatestBySender(data, "users/bot")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "good message" {
		t.Errorf("got %q, want %q", text, "good message")
	}
}
