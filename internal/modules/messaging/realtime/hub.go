package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mars-admin-go/internal/modules/messaging/application"
	"mars-admin-go/internal/modules/messaging/domain"
	"mars-admin-go/internal/platform/httpserver"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

const redisChannel = "mars:admin:realtime"

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (requestctx.Actor, error)
}

type Hub struct {
	logger    *slog.Logger
	auth      Authenticator
	messaging *application.Service
	redis     *redis.Client

	mu      sync.RWMutex
	clients map[uuid.UUID]map[*client]struct{}
	count   atomic.Int64
	metrics prometheus.GaugeFunc
}

type client struct {
	userID uuid.UUID
	conn   *websocket.Conn
	send   chan []byte
}

type envelope struct {
	Recipients []uuid.UUID  `json:"recipients"`
	Event      domain.Event `json:"event"`
}

func New(logger *slog.Logger, auth Authenticator, messaging *application.Service, redisClient *redis.Client, registry prometheus.Registerer) *Hub {
	hub := &Hub{
		logger: logger, auth: auth, messaging: messaging, redis: redisClient,
		clients: make(map[uuid.UUID]map[*client]struct{}),
	}
	hub.metrics = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "mars_admin",
		Subsystem: "websocket",
		Name:      "connections",
		Help:      "Current WebSocket connections.",
	}, func() float64 { return float64(hub.count.Load()) })
	registry.MustRegister(hub.metrics)
	return hub
}

func (h *Hub) Start(ctx context.Context) error {
	if h.redis == nil {
		return nil
	}
	go h.subscribe(ctx)
	return nil
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		token = strings.TrimSpace(r.Header.Get("Authorization"))
		if parts := strings.Fields(token); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			token = parts[1]
		}
	}
	if token == "" {
		if cookie, err := r.Cookie("mars_session"); err == nil {
			token = cookie.Value
		}
	}
	if token == "" {
		httpserver.WriteProblem(w, r, httpserver.NewProblem(http.StatusUnauthorized, "authentication_required", "Authentication is required."))
		return
	}
	actor, err := h.auth.Authenticate(r.Context(), token)
	if err != nil {
		httpserver.WriteProblem(w, r, httpserver.NewProblem(http.StatusUnauthorized, "invalid_session", "The session is invalid or expired."))
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns:  []string{"localhost:*", "127.0.0.1:*"},
		CompressionMode: websocket.CompressionContextTakeover,
	})
	if err != nil {
		h.logger.WarnContext(r.Context(), "WebSocket accept failed", "error", err)
		return
	}
	conn.SetReadLimit(64 << 10)
	c := &client{userID: actor.UserID, conn: conn, send: make(chan []byte, 32)}
	h.register(c)
	defer h.unregister(c)

	writeCtx, cancelWriter := context.WithCancel(context.Background())
	defer cancelWriter()
	go h.writer(writeCtx, c)
	h.enqueue(c, domain.Event{Type: "connected", Content: "connected", Timestamp: time.Now().UTC().UnixMilli()})

	for {
		readCtx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
		messageType, payload, err := conn.Read(readCtx)
		cancel()
		if err != nil {
			return
		}
		if messageType != websocket.MessageText {
			continue
		}
		h.handleMessage(r.Context(), actor, c, payload)
	}
}

func (h *Hub) Publish(ctx context.Context, recipients []uuid.UUID, event domain.Event) error {
	if len(recipients) == 0 {
		return nil
	}
	payload, err := json.Marshal(envelope{Recipients: recipients, Event: event})
	if err != nil {
		return err
	}
	if h.redis == nil {
		h.dispatch(payload)
		return nil
	}
	if err := h.redis.Publish(ctx, redisChannel, payload).Err(); err != nil {
		h.logger.WarnContext(ctx, "Redis WebSocket publish failed", "error", err)
		h.dispatch(payload)
		return err
	}
	return nil
}

func (h *Hub) Online(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

func (h *Hub) OnlineCount() int64 {
	return h.count.Load()
}

func (h *Hub) handleMessage(ctx context.Context, actor requestctx.Actor, c *client, payload []byte) {
	var input struct {
		Type            string         `json:"type"`
		ReceiverID      string         `json:"receiverId"`
		GroupID         string         `json:"groupId"`
		ClientMessageID string         `json:"clientMessageId"`
		Content         string         `json:"content"`
		MessageType     string         `json:"messageType"`
		Metadata        map[string]any `json:"metadata"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		h.enqueue(c, domain.Event{Type: "error", Content: "Invalid WebSocket payload."})
		return
	}
	switch input.Type {
	case "ping":
		h.enqueue(c, domain.Event{Type: "pong", Content: "pong"})
	case "chat":
		receiverID, err := uuid.Parse(input.ReceiverID)
		if err != nil {
			h.enqueue(c, domain.Event{Type: "error", Content: "receiverId must be a UUID."})
			return
		}
		if _, err := h.messaging.SendDirect(ctx, domain.SendInput{
			SenderID: actor.UserID, ReceiverID: receiverID, ClientMessageID: input.ClientMessageID,
			MessageType: input.MessageType, Content: input.Content, Metadata: input.Metadata,
		}); err != nil {
			h.enqueueError(c, err)
		}
	case "groupChat":
		groupID, err := uuid.Parse(input.GroupID)
		if err != nil {
			h.enqueue(c, domain.Event{Type: "error", Content: "groupId must be a UUID."})
			return
		}
		if _, err := h.messaging.SendGroup(ctx, groupID, actor.UserID, input.ClientMessageID, input.MessageType, input.Content, input.Metadata); err != nil {
			h.enqueueError(c, err)
		}
	default:
		h.enqueue(c, domain.Event{Type: "error", Content: "Unsupported WebSocket message type."})
	}
}

func (h *Hub) enqueueError(c *client, err error) {
	message := "Message could not be sent."
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		message = "Request timed out."
	} else if err != nil {
		message = err.Error()
	}
	h.enqueue(c, domain.Event{Type: "error", Content: message})
}

func (h *Hub) enqueue(c *client, event domain.Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	select {
	case c.send <- payload:
	default:
		_ = c.conn.Close(websocket.StatusPolicyViolation, "outbound queue full")
	}
}

func (h *Hub) writer(ctx context.Context, c *client) {
	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := c.conn.Write(writeCtx, websocket.MessageText, payload)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
	h.count.Add(1)
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients := h.clients[c.userID]; clients != nil {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.clients, c.userID)
		}
	}
	close(c.send)
	_ = c.conn.CloseNow()
	h.count.Add(-1)
}

func (h *Hub) subscribe(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		pubsub := h.redis.Subscribe(ctx, redisChannel)
		channel := pubsub.Channel()
		for message := range channel {
			h.dispatch([]byte(message.Payload))
		}
		_ = pubsub.Close()
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (h *Hub) dispatch(payload []byte) {
	var message envelope
	if err := json.Unmarshal(payload, &message); err != nil {
		h.logger.Warn("invalid realtime envelope", "error", err)
		return
	}
	data, err := json.Marshal(message.Event)
	if err != nil {
		return
	}
	h.mu.RLock()
	targets := make([]*client, 0)
	for _, recipient := range message.Recipients {
		for c := range h.clients[recipient] {
			targets = append(targets, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range targets {
		select {
		case c.send <- data:
		default:
			_ = c.conn.Close(websocket.StatusPolicyViolation, "outbound queue full")
		}
	}
}
