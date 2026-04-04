package gmail

import (
	"encoding/base64"
	"testing"
)

func TestDecodeGmailRawMessage_StdBase64(t *testing.T) {
	mime := "Subject: x\n\nhttps://docs.google.com/document/d/abc123_doc/edit\n"
	enc := base64.StdEncoding.EncodeToString([]byte(mime))
	got, err := decodeGmailRawMessage(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	docID, err := ExtractDocID(got)
	if err != nil {
		t.Fatalf("ExtractDocID: %v", err)
	}
	if docID != "abc123_doc" {
		t.Errorf("doc id: want abc123_doc, got %q", docID)
	}
}

func TestDecodeGmailRawMessage_RawURL(t *testing.T) {
	mime := "x"
	enc := base64.RawURLEncoding.EncodeToString([]byte(mime))
	_, err := decodeGmailRawMessage(enc)
	if err != nil {
		t.Fatalf("decode raw url: %v", err)
	}
}
