// ABOUTME: Home Assistant WebSocket API implementation
// ABOUTME: Handles real-time bidirectional communication for state updates and service calls

package homeassistant

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
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
	subscriptions map[int]string // message ID -> subscription type
}

// handleWebSocket upgrades HTTP connection to WebSocket and manages the client lifecycle
func (p *HomeAssistantPlugin) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &WSClient{
		conn:          conn,
		send:          make(chan []byte, 256),
		subscriptions: make(map[int]string),
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
		client.conn.Close()
	}()

	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
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
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		p.handleWSMessage(client, msg)
	}
}

// writePump handles outgoing WebSocket messages
func (client *WSClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendMessage serializes and sends a message to the client
func (client *WSClient) sendMessage(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	select {
	case client.send <- data:
	default:
		log.Printf("Client send buffer full, closing connection")
		close(client.send)
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
		// Send auth_invalid with error field
		client.send <- []byte(`{"type":"auth_invalid","message":"Invalid access token"}`)
		return
	}

	// Validate token and get instance
	instance, err := p.store.GetInstanceByToken(msg.AccessToken)
	if err != nil {
		log.Printf("Failed to get instance by token: %v", err)
		client.send <- []byte(`{"type":"auth_invalid","message":"Invalid access token"}`)
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
			json.Unmarshal([]byte(state.Attributes), &attributes)
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
