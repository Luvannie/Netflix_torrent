package events

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestBusSubscribeAndPublish(t *testing.T) {
	bus := NewBus()
	msgs, cancel := bus.Subscribe("/topic/test")
	defer cancel()

	testPayload := map[string]string{"message": "hello"}
	if err := bus.Publish("/topic/test", testPayload); err != nil {
		t.Fatalf("Publish error = %v", err)
	}

	select {
	case msg := <-msgs:
		if msg.Destination != "/topic/test" {
			t.Errorf("Destination = %q", msg.Destination)
		}
		var got map[string]string
		if err := json.Unmarshal(msg.Body, &got); err != nil {
			t.Fatalf("Unmarshal error = %v", err)
		}
		if got["message"] != "hello" {
			t.Errorf("message = %q", got["message"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for message")
	}
}

func TestBusPublishMultipleSubscribers(t *testing.T) {
	bus := NewBus()
	msgs1, cancel1 := bus.Subscribe("/topic/multi")
	msgs2, cancel2 := bus.Subscribe("/topic/multi")
	defer cancel1()
	defer cancel2()

	payload := map[string]int{"count": 2}
	if err := bus.Publish("/topic/multi", payload); err != nil {
		t.Fatalf("Publish error = %v", err)
	}

	received := 0
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		select {
		case <-msgs1:
			received++
		case <-time.After(100 * time.Millisecond):
		}
		wg.Done()
	}()

	go func() {
		select {
		case <-msgs2:
			received++
		case <-time.After(100 * time.Millisecond):
		}
		wg.Done()
	}()

	wg.Wait()
	if received != 2 {
		t.Errorf("received = %d, want 2", received)
	}
}

func TestBusUnsubscribe(t *testing.T) {
	bus := NewBus()
	_, cancel := bus.Subscribe("/topic/cancel")
	cancel()

	time.Sleep(10 * time.Millisecond)

	if err := bus.Publish("/topic/cancel", map[string]string{"msg": "ignored"}); err != nil {
		t.Fatalf("Publish error = %v", err)
	}

	select {
	case <-time.After(50 * time.Millisecond):
	}
}

func TestMessageJSON(t *testing.T) {
	msg := Message{
		Destination: "/topic/search/jobs/5",
		Body:        []byte(`{"type":"SearchCompletedEvent","jobId":5}`),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var decoded struct {
		Destination string `json:"destination"`
		Body        []byte `json:"body"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if decoded.Destination != "/topic/search/jobs/5" {
		t.Errorf("Destination = %q", decoded.Destination)
	}

	var bodyDecoded map[string]interface{}
	if err := json.Unmarshal(decoded.Body, &bodyDecoded); err != nil {
		t.Fatalf("Body unmarshal error = %v", err)
	}
	if bodyDecoded["type"] != "SearchCompletedEvent" {
		t.Errorf("type = %v", bodyDecoded["type"])
	}
}
