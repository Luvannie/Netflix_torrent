package websocket

import (
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"
)

type Message struct {
	Type     string `json:"type"`
	TaskID   int64  `json:"taskId,omitempty"`
	Progress int    `json:"progress,omitempty"`
	JobID    int64  `json:"jobId,omitempty"`
}

type Client interface {
	RemoteAddr() net.Addr
	Send(msg []byte) error
	Close() error
}

type Frame struct {
	Command string
	Headers map[string]string
	Body    []byte
}

type Hub struct {
	clients    map[Client]bool
	broadcast  chan []byte
	register   chan Client
	unregister chan Client
	mu         sync.RWMutex
	stop       chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[Client]bool),
		broadcast:  make(chan []byte, 10),
		register:   make(chan Client),
		unregister: make(chan Client),
		stop:       make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				go client.Send(message)
			}
			h.mu.RUnlock()
		case <-h.stop:
			return
		}
	}
}

func (h *Hub) Register(client Client) {
	h.register <- client
}

func (h *Hub) Unregister(client Client) {
	h.unregister <- client
}

func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

func (h *Hub) Stop() {
	close(h.stop)
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func ParseFrame(data string) Frame {
	if len(data) == 0 || data[0] == 0x00 {
		return Frame{}
	}

	lines := strings.Split(data, "\n")
	if len(lines) == 0 {
		return Frame{}
	}

	frame := Frame{
		Command: lines[0],
		Headers: make(map[string]string),
	}

	bodyStart := -1
	for i := 1; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		if line == "" {
			bodyStart = i + 1
			break
		}
		if idx := strings.Index(line, ":"); idx > 0 {
			key := line[:idx]
			val := line[idx+1:]
			frame.Headers[key] = val
		}
	}

	if bodyStart > 0 && bodyStart < len(lines) {
		body := strings.Join(lines[bodyStart:], "\n")
		body = strings.TrimRight(body, "\x00")
		frame.Body = []byte(body)
	}

	return frame
}

func FormatFrame(command string, headers map[string]string, body string) string {
	var sb strings.Builder
	sb.WriteString(command)
	sb.WriteString("\n")

	for k, v := range headers {
		sb.WriteString(k)
		sb.WriteString(":")
		sb.WriteString(v)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	if body != "" {
		sb.WriteString(body)
	}

	sb.WriteString("\x00")

	return sb.String()
}

func SendMESSAGE(ws Client, subscription, destination, body string) error {
	frame := FormatFrame("MESSAGE", map[string]string{
		"subscription": subscription,
		"destination":  destination,
		"content-type":  "application/json",
	}, body)
	return ws.Send([]byte(frame))
}

func SendCONNECTED(ws Client) error {
	frame := FormatFrame("CONNECTED", map[string]string{
		"version": "1.2",
	}, "")
	return ws.Send([]byte(frame))
}

func SendERROR(ws Client, message string) error {
	frame := FormatFrame("ERROR", map[string]string{
		"message": message,
	}, "")
	return ws.Send([]byte(frame))
}

var _ = time.Now
var _ = json.Marshal
