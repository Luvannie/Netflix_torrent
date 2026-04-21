package websocket

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type WebSocketConn struct {
	conn   net.Conn
	reader *bufio.Reader
	Frames chan Frame
}

func HijackAndUpgrade(w http.ResponseWriter, r *http.Request) *WebSocketConn {
	h, ok := w.(http.Hijacker)
	if !ok {
		return nil
	}
	conn, _, err := h.Hijack()
	if err != nil {
		return nil
	}

	conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))

	return &WebSocketConn{
		conn:   conn,
		reader: bufio.NewReader(conn),
		Frames: make(chan Frame, 10),
	}
}

func (ws *WebSocketConn) ReadMessage() (int, []byte, error) {
	data, err := ws.reader.ReadBytes(0x00)
	if err != nil {
		return 0, nil, err
	}
	data = data[:len(data)-1]
	return 1, data, nil
}

func (ws *WebSocketConn) WriteMessage(msgType int, data []byte) error {
	if msgType == 1 {
		frame := append([]byte{0x81}, data...)
		frame = append(frame, 0x00)
		_, err := ws.conn.Write(frame)
		return err
	}
	return nil
}

func (ws *WebSocketConn) Close() error {
	return ws.conn.Close()
}

func (ws *WebSocketConn) WriteFrame(data string) {
	if ws.conn != nil {
		ws.conn.Write([]byte(data))
	} else {
		ws.Frames <- ParseFrame(data)
	}
}

func (ws *WebSocketConn) ReadFrame() (Frame, error) {
	if ws.conn != nil {
		data, _ := ws.reader.ReadBytes(0x00)
		data = data[:len(data)-1]
		return ParseFrame(string(data)), nil
	}
	return <-ws.Frames, nil
}

type STOMPHandler struct {
	hub      *Hub
	events   EventBus
	sessions map[string]*Subscription
	mu       sync.RWMutex
}

type EventBus interface {
	Publish(destination string, payload any) error
	Subscribe(destination string) (<-chan EventMessage, func())
}

type EventMessage struct {
	Destination string
	Body        []byte
}

type Subscription struct {
	ID          string
	Destination string
	ch          <-chan EventMessage
	cancel      func()
}

func NewSTOMPHandler(hub *Hub, events EventBus) *STOMPHandler {
	return &STOMPHandler{
		hub:      hub,
		events:   events,
		sessions: make(map[string]*Subscription),
	}
}

func (h *STOMPHandler) Handle(ws *WebSocketConn, frame Frame) error {
	switch frame.Command {
	case "CONNECT":
		return nil
	case "SUBSCRIBE":
		return h.handleSUBSCRIBE(ws, frame)
	case "UNSUBSCRIBE":
		return h.handleUNSUBSCRIBE(ws, frame)
	case "DISCONNECT":
		return h.handleDISCONNECT(ws, frame)
	default:
		ws.WriteFrame("ERROR\nmessage:Unknown command\n\n\x00")
	}
	return nil
}

func (h *STOMPHandler) handleSUBSCRIBE(ws *WebSocketConn, frame Frame) error {
	id := frame.Headers["id"]
	dest := frame.Headers["destination"]

	if id == "" || dest == "" {
		ws.WriteFrame("ERROR\nmessage:Missing id or destination\n\n\x00")
		return nil
	}

	ch, cancel := h.events.Subscribe(dest)

	h.mu.Lock()
	h.sessions[id] = &Subscription{
		ID:          id,
		Destination: dest,
		ch:          ch,
		cancel:      cancel,
	}
	h.mu.Unlock()

	go func() {
		for msg := range ch {
			body := string(msg.Body)
			ws.WriteFrame("MESSAGE\nsubscription:" + id + "\ndestination:" + dest + "\ncontent-type:application/json\n\n" + body + "\x00")
		}
	}()

	return nil
}

func (h *STOMPHandler) handleUNSUBSCRIBE(ws *WebSocketConn, frame Frame) error {
	id := frame.Headers["id"]

	h.mu.Lock()
	if sub, ok := h.sessions[id]; ok {
		sub.cancel()
		delete(h.sessions, id)
	}
	h.mu.Unlock()

	return nil
}

func (h *STOMPHandler) handleDISCONNECT(ws *WebSocketConn, frame Frame) error {
	h.mu.Lock()
	for _, sub := range h.sessions {
		sub.cancel()
	}
	h.sessions = make(map[string]*Subscription)
	h.mu.Unlock()
	return nil
}

func PublishDownloadProgress(events EventBus, taskID int64, progress int) error {
	body, _ := json.Marshal(map[string]interface{}{
		"type":     "DownloadProgressEvent",
		"taskId":   taskID,
		"progress": progress,
	})
	return events.Publish("/topic/downloads/"+formatInt(taskID), body)
}

func PublishSearchCompleted(events EventBus, jobID int64) error {
	body, _ := json.Marshal(map[string]interface{}{
		"type":  "SearchCompletedEvent",
		"jobId": jobID,
	})
	return events.Publish("/topic/search/jobs/"+formatInt(jobID), body)
}

func formatInt(n int64) string {
	return strconv.FormatInt(n, 10)
}

var _ = time.Now
