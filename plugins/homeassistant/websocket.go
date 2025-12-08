// ABOUTME: Home Assistant WebSocket API implementation
// ABOUTME: Handles real-time bidirectional communication for state updates and service calls

package homeassistant

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 54 * time.Second
	maxMessageSize = 512 * 1024 // 512 KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Check origin header for security
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Allow requests with no origin (like direct WebSocket clients)
		}
		// Allow localhost and common development origins
		allowedOrigins := []string{"localhost", "127.0.0.1", "::1"}
		for _, allowed := range allowedOrigins {
			if strings.Contains(origin, allowed) {
				return true
			}
		}
		return false
	},
}

// WSMessage represents a Home Assistant WebSocket message
type WSMessage struct {
	ID          int                    `json:"id,omitempty"`
	Type        string                 `json:"type"`
	AccessToken string                 `json:"access_token,omitempty"`
	HAVersion   string                 `json:"ha_version,omitempty"`
	Result      interface{}            `json:"result,omitempty"`
	Success     bool                   `json:"success,omitempty"`
	Error       *WSError               `json:"error,omitempty"`
	Event       map[string]interface{} `json:"event,omitempty"`
}

// WSError represents a WebSocket error response
type WSError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WSClient represents a connected WebSocket client
type WSClient struct {
	conn          *websocket.Conn
	send          chan []byte
	instance      *Instance
	authenticated bool
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	closeOnce     sync.Once // ensures send channel is closed only once
	closeConn     sync.Once // ensures conn.Close() is called only once
}

// handleWebSocket upgrades HTTP connection to WebSocket and manages the client lifecycle
func (p *HomeAssistantPlugin) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := &WSClient{
		conn:   conn,
		send:   make(chan []byte, 256),
		ctx:    ctx,
		cancel: cancel,
	}

	// Start goroutines for reading and writing
	go client.writePump()
	go p.readPump(client)

	// Send auth_required message
	authRequired := WSMessage{
		Type:      "auth_required",
		HAVersion: "2024.1.0",
	}
	client.sendMessage(authRequired)
}

// readPump handles incoming WebSocket messages
func (p *HomeAssistantPlugin) readPump(client *WSClient) {
	defer func() {
		client.cancel() // Signal writePump to stop
		client.closeOnce.Do(func() {
			close(client.send)
		})
		client.closeConn.Do(func() {
			client.conn.Close()
		})
	}()

	client.conn.SetReadLimit(maxMessageSize)
	if err := client.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
		return
	}

	client.conn.SetPongHandler(func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to unmarshal WebSocket message: %v", err)
			// Send error response
			client.sendMessage(WSMessage{
				Type:    "result",
				ID:      msg.ID,
				Success: false,
				Error: &WSError{
					Code:    "invalid_format",
					Message: "Failed to parse message",
				},
			})
			continue
		}

		p.handleWSMessage(client, msg)
	}
}

// writePump handles outgoing WebSocket messages
func (client *WSClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.closeConn.Do(func() {
			client.conn.Close()
		})
	}()

	for {
		select {
		case message, ok := <-client.send:
			if err := client.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Failed to set write deadline: %v", err)
				return
			}
			if !ok {
				// Channel closed, send close message
				if err := client.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("Failed to write close message: %v", err)
				}
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Failed to write message: %v", err)
				return
			}

		case <-ticker.C:
			if err := client.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Failed to set ping write deadline: %v", err)
				return
			}
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to write ping: %v", err)
				return
			}

		case <-client.ctx.Done():
			// Context cancelled, clean shutdown
			return
		}
	}
}

// sendMessage serializes and sends a message to the client
func (client *WSClient) sendMessage(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal WebSocket message: %v", err)
		return
	}

	select {
	case client.send <- data:
	case <-client.ctx.Done():
		// Connection is closing
		return
	default:
		// Buffer full, drop message (checking context would introduce TOCTOU race)
		log.Printf("Client send buffer full, dropping message")
	}
}

// handleWSMessage routes WebSocket messages to appropriate handlers
func (p *HomeAssistantPlugin) handleWSMessage(client *WSClient, msg WSMessage) {
	client.mu.RLock()
	authenticated := client.authenticated
	client.mu.RUnlock()

	// Handle authentication
	if !authenticated {
		if msg.Type == "auth" {
			p.handleWSAuth(client, msg)
		} else {
			client.sendMessage(WSMessage{
				Type: "result",
				ID:   msg.ID,
				Success: false,
				Error: &WSError{
					Code:    "unauthorized",
					Message: "Authentication required",
				},
			})
		}
		return
	}

	// Route authenticated messages
	switch msg.Type {
	case "get_states":
		p.handleWSGetStates(client, msg)
	case "ping":
		p.handleWSPing(client, msg)
	default:
		client.sendMessage(WSMessage{
			Type:    "result",
			ID:      msg.ID,
			Success: false,
			Error: &WSError{
				Code:    "unknown_command",
				Message: "Unknown command: " + msg.Type,
			},
		})
	}
}

// handleWSAuth handles WebSocket authentication
func (p *HomeAssistantPlugin) handleWSAuth(client *WSClient, msg WSMessage) {
	if msg.AccessToken == "" {
		// Send auth_invalid message
		authInvalid := WSMessage{
			Type: "auth_invalid",
			Error: &WSError{
				Code:    "invalid_auth",
				Message: "Access token required",
			},
		}
		client.sendMessage(authInvalid)
		// Close connection after auth failure
		client.cancel()
		return
	}

	// Validate token and get instance
	instance, err := p.store.GetInstanceByToken(msg.AccessToken)
	if err != nil {
		log.Printf("Failed to get instance by token: %v", err)
		authInvalid := WSMessage{
			Type: "auth_invalid",
			Error: &WSError{
				Code:    "invalid_auth",
				Message: "Invalid access token",
			},
		}
		client.sendMessage(authInvalid)
		// Close connection after auth failure
		client.cancel()
		return
	}

	client.mu.Lock()
	client.authenticated = true
	client.instance = instance
	client.mu.Unlock()

	client.sendMessage(WSMessage{
		Type:      "auth_ok",
		HAVersion: "2024.1.0",
	})

	log.Printf("WebSocket client authenticated for instance: %s", instance.Name)
}

// handleWSGetStates returns all entity states
func (p *HomeAssistantPlugin) handleWSGetStates(client *WSClient, msg WSMessage) {
	client.mu.RLock()
	instance := client.instance
	client.mu.RUnlock()

	if instance == nil {
		client.sendMessage(WSMessage{
			Type:    "result",
			ID:      msg.ID,
			Success: false,
			Error: &WSError{
				Code:    "unauthorized",
				Message: "Not authenticated",
			},
		})
		return
	}

	states, err := p.store.ListAllStates(1000, 0)
	if err != nil {
		client.sendMessage(WSMessage{
			Type:    "result",
			ID:      msg.ID,
			Success: false,
			Error: &WSError{
				Code:    "internal_error",
				Message: err.Error(),
			},
		})
		return
	}

	// Filter to this instance and convert to HA format
	result := make([]map[string]interface{}, 0)
	for _, state := range states {
		if state.InstanceID != instance.ID {
			continue
		}

		var attributes map[string]interface{}
		if state.Attributes != "" {
			if err := json.Unmarshal([]byte(state.Attributes), &attributes); err != nil {
				log.Printf("Failed to unmarshal attributes for entity %s: %v", state.EntityID, err)
				attributes = make(map[string]interface{})
			}
		}

		result = append(result, map[string]interface{}{
			"entity_id":    state.EntityID,
			"state":        state.State,
			"attributes":   attributes,
			"last_changed": state.LastChanged.Format(time.RFC3339),
			"last_updated": state.LastUpdated.Format(time.RFC3339),
			"context": map[string]interface{}{
				"id":        state.ID,
				"parent_id": nil,
				"user_id":   nil,
			},
		})
	}

	client.sendMessage(WSMessage{
		Type:    "result",
		ID:      msg.ID,
		Success: true,
		Result:  result,
	})
}

// handleWSPing responds to ping messages
func (p *HomeAssistantPlugin) handleWSPing(client *WSClient, msg WSMessage) {
	client.sendMessage(WSMessage{
		Type: "pong",
		ID:   msg.ID,
	})
}
