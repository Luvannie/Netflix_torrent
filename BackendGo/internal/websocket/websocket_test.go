package websocket

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

func TestWebSocketHubRegister(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	conn := &testConn{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}}
	hub.Register(conn)
	time.Sleep(10 * time.Millisecond)

	if len(hub.clients) != 1 {
		t.Errorf("clients count = %d", len(hub.clients))
	}
}

func TestWebSocketHubUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	conn := &testConn{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}}
	hub.Register(conn)
	time.Sleep(10 * time.Millisecond)
	hub.Unregister(conn)
	time.Sleep(10 * time.Millisecond)

	if len(hub.clients) != 0 {
		t.Errorf("clients count = %d", len(hub.clients))
	}
}

func TestWebSocketMessageBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	conn1 := &testConn{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}, send: make(chan []byte, 1)}
	conn2 := &testConn{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1235}, send: make(chan []byte, 1)}
	hub.Register(conn1)
	hub.Register(conn2)
	time.Sleep(10 * time.Millisecond)

	hub.Broadcast([]byte("hello"))

	select {
	case msg := <-conn1.send:
		if string(msg) != "hello" {
			t.Errorf("conn1 msg = %q", msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("conn1 timeout")
	}

	select {
	case msg := <-conn2.send:
		if string(msg) != "hello" {
			t.Errorf("conn2 msg = %q", msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("conn2 timeout")
	}
}

func TestParseFrame(t *testing.T) {
	frame := ParseFrame("MESSAGE\nsubscription:sub-1\ndestination:/topic/test\ncontent-type:application/json\n\n{\"msg\":\"hello\"}\x00")

	if frame.Command != "MESSAGE" {
		t.Errorf("Command = %q", frame.Command)
	}
	if frame.Headers["subscription"] != "sub-1" {
		t.Errorf("subscription = %q", frame.Headers["subscription"])
	}
	if frame.Headers["destination"] != "/topic/test" {
		t.Errorf("destination = %q", frame.Headers["destination"])
	}
}

func TestFormatFrame(t *testing.T) {
	frame := FormatFrame("MESSAGE", map[string]string{
		"subscription": "sub-1",
		"destination":  "/topic/test",
		"content-type": "application/json",
	}, `{"msg":"hello"}`)

	if frame == "" {
		t.Error("frame is empty")
	}
	if !strings.Contains(frame, "MESSAGE") {
		t.Errorf("frame does not contain MESSAGE")
	}
	if !strings.Contains(frame, "sub-1") {
		t.Errorf("frame does not contain sub-1")
	}
}

func TestDownloadProgressEvent(t *testing.T) {
	msg := Message{
		Type:     "DownloadProgressEvent",
		TaskID:   10,
		Progress: 50,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if decoded.Type != "DownloadProgressEvent" {
		t.Errorf("Type = %q", decoded.Type)
	}
	if decoded.TaskID != 10 {
		t.Errorf("TaskID = %d", decoded.TaskID)
	}
	if decoded.Progress != 50 {
		t.Errorf("Progress = %d", decoded.Progress)
	}
}

func TestSearchCompletedEvent(t *testing.T) {
	msg := Message{
		Type:  "SearchCompletedEvent",
		JobID: 5,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if decoded.Type != "SearchCompletedEvent" {
		t.Errorf("Type = %q", decoded.Type)
	}
	if decoded.JobID != 5 {
		t.Errorf("JobID = %d", decoded.JobID)
	}
}

type testConn struct {
	addr net.Addr
	send chan []byte
}

func (c *testConn) RemoteAddr() net.Addr { return c.addr }
func (c *testConn) Send(msg []byte) error {
	select {
	case c.send <- msg:
	default:
	}
	return nil
}
func (c *testConn) Close() error { return nil }
