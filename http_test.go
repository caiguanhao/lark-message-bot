package main

import (
	"encoding/json"
	"testing"

	"github.com/caiguanhao/larkslim"
)

func TestHandleMessagePayloadIgnoresOtherEvents(t *testing.T) {
	h := &httpHandler{}
	payload, err := json.Marshal(struct {
		Header struct {
			EventType string `json:"event_type"`
		} `json:"header"`
	}{})
	if err != nil {
		t.Fatal(err)
	}
	if err := h.handleMessagePayload(payload); err != nil {
		t.Fatal(err)
	}
}

func TestMessageReceiveEventText(t *testing.T) {
	event := larkslim.MessageReceiveEvent{}
	event.Event.Message.Content = `{"text":"help"}`
	if got := event.Text(); got != "help" {
		t.Fatalf("Text() = %q, want help", got)
	}
}
