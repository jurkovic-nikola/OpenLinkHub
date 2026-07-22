package ws

import (
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/stats"
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	TopicDashboard = "dashboard"
	TopicTemps     = "temps"
	TopicBattery   = "battery"
)

type clientMsg struct {
	Type   string `json:"type"`
	Topic  string `json:"topic,omitempty"`
	Serial string `json:"serial,omitempty"`
}

type serverMsg struct {
	Type   string `json:"type"`
	Topic  string `json:"topic,omitempty"`
	Serial string `json:"serial,omitempty"`
	Data   any    `json:"data,omitempty"`
}

type client struct {
	conn    *websocket.Conn
	mu      sync.Mutex
	topics  map[string]struct{}
	sendCh  chan []byte
	closed  bool
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

var Default = NewHub()

func NewHub() *Hub {
	h := &Hub{clients: make(map[*client]struct{})}
	go h.broadcastLoop()
	return h
}

func (h *Hub) Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		CompressionMode:    websocket.CompressionDisabled,
	})
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("WebSocket accept failed")
		return
	}

	c := &client{
		conn:   conn,
		topics: make(map[string]struct{}),
		sendCh: make(chan []byte, 32),
	}

	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	ctx := r.Context()
	go c.writePump(ctx)
	c.readPump(ctx, h)

	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	c.close()
}

func (c *client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	close(c.sendCh)
	_ = c.conn.Close(websocket.StatusNormalClosure, "")
}

func (c *client) writePump(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.sendCh:
			if !ok {
				return
			}
			wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.conn.Write(wctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (c *client) readPump(ctx context.Context, h *Hub) {
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}
		var msg clientMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "subscribe":
			topic := msg.Topic
			if topic == "" && msg.Serial != "" {
				topic = "device:" + msg.Serial
			}
			if topic == "" {
				continue
			}
			c.mu.Lock()
			c.topics[topic] = struct{}{}
			c.mu.Unlock()
			h.pushTopicToClient(c, topic)
		case "unsubscribe":
			topic := msg.Topic
			if topic == "" && msg.Serial != "" {
				topic = "device:" + msg.Serial
			}
			c.mu.Lock()
			delete(c.topics, topic)
			c.mu.Unlock()
		case "ping":
			c.enqueue(serverMsg{Type: "pong"})
		}
	}
}

func (c *client) enqueue(msg serverMsg) {
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	select {
	case c.sendCh <- b:
	default:
		// Drop if client is slow
	}
}

func (c *client) subscribed(topic string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.topics[topic]
	return ok
}

func (h *Hub) broadcastLoop() {
	t := time.NewTicker(1500 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		h.broadcastAll()
	}
}

func (h *Hub) broadcastAll() {
	h.mu.RLock()
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		c.mu.Lock()
		topics := make([]string, 0, len(c.topics))
		for t := range c.topics {
			topics = append(topics, t)
		}
		c.mu.Unlock()
		for _, topic := range topics {
			h.pushTopicToClient(c, topic)
		}
	}
}

func (h *Hub) pushTopicToClient(c *client, topic string) {
	switch {
	case topic == TopicDashboard:
		c.enqueue(serverMsg{
			Type:  "devices",
			Topic: TopicDashboard,
			Data:  devices.GetDevices(),
		})
	case topic == TopicTemps:
		c.enqueue(serverMsg{
			Type:  "temps",
			Topic: TopicTemps,
			Data: map[string]any{
				"cpu":     dashboard.GetDashboard().TemperatureToString(temperatures.GetCpuTemperature()),
				"cpuRaw":  temperatures.GetCpuTemperature(),
				"gpu":     dashboard.GetDashboard().TemperatureToString(temperatures.GetGpuTemperature()),
				"gpuRaw":  temperatures.GetGpuTemperature(),
				"storage": temperatures.GetStorageTemperatures(),
				"gpus": func() map[int]any {
					out := make(map[int]any)
					for key, val := range systeminfo.GetInfo().GPU {
						out[key] = dashboard.GetDashboard().TemperatureToString(temperatures.GetGpuTemperatureIndex(val.Index))
					}
					return out
				}(),
			},
		})
	case topic == TopicBattery:
		c.enqueue(serverMsg{
			Type:  "battery",
			Topic: TopicBattery,
			Data:  stats.GetBatteryStats(),
		})
	case len(topic) > 7 && topic[:7] == "device:":
		serial := topic[7:]
		c.enqueue(serverMsg{
			Type:   "device",
			Topic:  topic,
			Serial: serial,
			Data:   devices.GetDevice(serial),
		})
	}
}
