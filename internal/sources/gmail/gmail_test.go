package gmail_test

import (
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/sources/gmail"
)

func TestExtractMessageID(t *testing.T) {
	input := []byte(`{
	  "messages": [
	    {"id": "19d100ee48e14953", "threadId": "19d100ee48e14953"}
	  ],
	  "resultSizeEstimate": 1
	}`)
	id, err := gmail.ExtractMessageID(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "19d100ee48e14953" {
		t.Errorf("expected 19d100ee48e14953, got %q", id)
	}
}

func TestExtractMessageID_Empty(t *testing.T) {
	input := []byte(`{"messages": [], "resultSizeEstimate": 0}`)
	_, err := gmail.ExtractMessageID(input)
	if err == nil {
		t.Fatal("expected error for empty messages, got nil")
	}
}

func TestExtractDocID(t *testing.T) {
	emailBody := `Dear Colleague,
Open Weekly Report
<https://docs.google.com/document/d/1FGG0-VOGVBoRFaLOsnZ9ApWmcBue_hJZay1xtN3aKig/edit?usp=drivesdk>
Best regards`

	docID, err := gmail.ExtractDocID([]byte(emailBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if docID != "1FGG0-VOGVBoRFaLOsnZ9ApWmcBue_hJZay1xtN3aKig" {
		t.Errorf("expected doc ID, got %q", docID)
	}
}

func TestExtractDocID_NoURL(t *testing.T) {
	_, err := gmail.ExtractDocID([]byte("no links here"))
	if err == nil {
		t.Fatal("expected error when no doc URL found, got nil")
	}
}
