package websocket

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/netflixtorrent/backend-go/internal/events"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	Subprotocols: []string{"v10.stomp", "v11.stomp", "v12.stomp"},
}

type WebSocketConn struct {
	conn *websocket.Conn
}

func (ws *WebSocketConn) RemoteAddr() net.Addr {
	return ws.conn.RemoteAddr()
}

func (ws *WebSocketConn) Send(msg []byte) error {
	return ws.conn.WriteMessage(websocket.TextMessage, msg)
}

func (ws *WebSocketConn) Close() error {
	return ws.conn.Close()
}

func (ws *WebSocketConn) ReadFrame() (Frame, error) {
	_, reader, err := ws.conn.NextReader()
	if err != nil {
		return Frame{}, err
	}
	data := make([]byte, 0, 4096)
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
			if len(data) > 0 && data[len(data)-1] == 0x00 {
				data = data[:len(data)-1]
				break
			}
		}
		if err != nil {
			break
		}
	}
	return ParseFrame(string(data)), nil
}

func (ws *WebSocketConn) WriteFrame(data string) {
	ws.conn.WriteMessage(websocket.TextMessage, []byte(data))
}

func ServeWS(hub *Hub, eventBus *events.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		wsConn := &WebSocketConn{conn: conn}
		hub.Register(wsConn)
		defer hub.Unregister(wsConn)

		stompHandler := NewSTOMPHandler(hub, eventBus)

		for {
			frame, err := wsConn.ReadFrame()
			if err != nil {
				break
			}

			if err := stompHandler.Handle(wsConn, frame); err != nil {
				break
			}
		}
	}
}

type STOMPHandler struct {
	hub      *Hub
	events   *events.Bus
	sessions map[string]*Subscription
	mu       sync.RWMutex
}

type Subscription struct {
	ID          string
	Destination string
	ch          <-chan events.Message
	cancel      func()
}

func NewSTOMPHandler(hub *Hub, bus *events.Bus) *STOMPHandler {
	return &STOMPHandler{
		hub:      hub,
		events:   bus,
		sessions: make(map[string]*Subscription),
	}
}

func (h *STOMPHandler) Handle(ws *WebSocketConn, frame Frame) error {
	switch frame.Command {
	case "CONNECT":
		ws.WriteFrame(FormatFrame("CONNECTED", map[string]string{
			"version":   "1.2",
			"user-name": "local-user",
		}, ""))
		return nil
	case "SUBSCRIBE":
		return h.handleSUBSCRIBE(ws, frame)
	case "UNSUBSCRIBE":
		return h.handleUNSUBSCRIBE(ws, frame)
	case "DISCONNECT":
		return h.handleDISCONNECT(ws, frame)
	default:
		ws.WriteFrame(FormatFrame("ERROR", map[string]string{
			"message": "Unknown command",
		}, ""))
	}
	return nil
}

func (h *STOMPHandler) handleSUBSCRIBE(ws *WebSocketConn, frame Frame) error {
	id := frame.Headers["id"]
	dest := frame.Headers["destination"]

	if id == "" || dest == "" {
		ws.WriteFrame(FormatFrame("ERROR", map[string]string{
			"message": "Missing id or destination",
		}, ""))
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
			ws.WriteFrame(FormatFrame("MESSAGE", map[string]string{
				"subscription": id,
				"destination":  dest,
				"content-type": "application/json",
			}, body))
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

func PublishDownloadProgress(eventsBus *events.Bus, taskID int64, progress int) error {
	body, _ := json.Marshal(map[string]interface{}{
		"type":     "DownloadProgressEvent",
		"taskId":   taskID,
		"progress": progress,
	})
	return eventsBus.Publish("/topic/downloads/"+formatInt(taskID), body)
}

func PublishSearchCompleted(eventsBus *events.Bus, jobID int64) error {
	body, _ := json.Marshal(map[string]interface{}{
		"type":  "SearchCompletedEvent",
		"jobId": jobID,
	})
	return eventsBus.Publish("/topic/search/jobs/"+formatInt(jobID), body)
}

func formatInt(n int64) string {
	return strconv.FormatInt(n, 10)
}

var _ = time.Now
