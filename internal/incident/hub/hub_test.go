package hub

import (
	"testing"
	"time"
)

func TestHubPublishesAndClosesSubscribers(t *testing.T) {
	h := New()
	stream, unsubscribe := h.Subscribe()
	h.Publish(Event{Type: "incident.created"})
	select {
	case event := <-stream:
		if event.Type != "incident.created" || event.Timestamp.IsZero() {
			t.Fatalf("unexpected event: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
	unsubscribe()
	h.Close()
	select {
	case _, open := <-stream:
		if open {
			t.Fatal("subscriber channel should be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not close")
	}
}
