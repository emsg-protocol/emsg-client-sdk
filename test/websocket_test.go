package test

import (
	"testing"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
	"github.com/emsg-protocol/emsg-client-sdk/message"
	"github.com/emsg-protocol/emsg-client-sdk/notifications"
	"github.com/emsg-protocol/emsg-client-sdk/websocket"
)

func TestWebSocketClientCreation(t *testing.T) {
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	notificationManager := notifications.NewNotificationManager(5)
	defer notificationManager.Shutdown()

	wsClient := websocket.NewWebSocketClient("ws://localhost:8080", keyPair, notificationManager)
	if wsClient == nil {
		t.Fatal("Failed to create WebSocket client")
	}

	if wsClient.IsConnected() {
		t.Error("WebSocket should not be connected initially")
	}
}

func TestReconnectStrategy(t *testing.T) {
	strategy := websocket.DefaultReconnectStrategy()
	
	if strategy.MaxRetries != 10 {
		t.Errorf("Expected MaxRetries 10, got %d", strategy.MaxRetries)
	}

	if strategy.InitialDelay != 1*time.Second {
		t.Errorf("Expected InitialDelay 1s, got %v", strategy.InitialDelay)
	}

	if strategy.MaxDelay != 30*time.Second {
		t.Errorf("Expected MaxDelay 30s, got %v", strategy.MaxDelay)
	}

	if strategy.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor 2.0, got %f", strategy.BackoffFactor)
	}

	if !strategy.EnableReconnect {
		t.Error("Expected EnableReconnect to be true")
	}
}

func TestWebSocketEventHandlers(t *testing.T) {
	keyPair, _ := keymgmt.GenerateKeyPair()
	notificationManager := notifications.NewNotificationManager(5)
	defer notificationManager.Shutdown()

	wsClient := websocket.NewWebSocketClient("ws://localhost:8080", keyPair, notificationManager)

	// Test event handler registration
	var eventReceived bool
	handler := func(data interface{}) {
		eventReceived = true
	}

	wsClient.RegisterEventHandler(websocket.EventConnected, handler)

	// We can't easily test the actual event triggering without a real WebSocket server
	// But we can verify the handler was registered without errors
	if eventReceived {
		t.Error("Event should not have been triggered yet")
	}
}

func TestWebSocketMessage(t *testing.T) {
	// Test WebSocket message structure
	testMsg := &message.Message{
		From:      "alice#example.com",
		To:        []string{"bob#example.com"},
		Subject:   "Test",
		Body:      "Test message",
		Timestamp: time.Now().Unix(),
	}

	wsMsg := &websocket.WebSocketMessage{
		Type:      "message",
		Message:   testMsg,
		Timestamp: time.Now().Unix(),
	}

	if wsMsg.Type != "message" {
		t.Errorf("Expected type 'message', got %s", wsMsg.Type)
	}

	if wsMsg.Message != testMsg {
		t.Error("Message not set correctly")
	}
}

func TestSetReconnectStrategy(t *testing.T) {
	keyPair, _ := keymgmt.GenerateKeyPair()
	notificationManager := notifications.NewNotificationManager(5)
	defer notificationManager.Shutdown()

	wsClient := websocket.NewWebSocketClient("ws://localhost:8080", keyPair, notificationManager)

	customStrategy := &websocket.ReconnectStrategy{
		MaxRetries:      5,
		InitialDelay:    500 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		BackoffFactor:   1.5,
		EnableReconnect: false,
	}

	wsClient.SetReconnectStrategy(customStrategy)

	// We can't directly access the strategy to verify it was set,
	// but we can verify the method doesn't panic
}

func TestWebSocketEvents(t *testing.T) {
	// Test WebSocket event constants
	events := []websocket.WebSocketEvent{
		websocket.EventConnected,
		websocket.EventDisconnected,
		websocket.EventMessage,
		websocket.EventError,
		websocket.EventReconnecting,
	}

	expectedEvents := []string{
		"connected",
		"disconnected",
		"message",
		"error",
		"reconnecting",
	}

	for i, event := range events {
		if string(event) != expectedEvents[i] {
			t.Errorf("Expected event %s, got %s", expectedEvents[i], string(event))
		}
	}
}

// Mock WebSocket message for testing
type MockWebSocketMessage struct {
	Type      string                 `json:"type"`
	Event     string                 `json:"event,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

func TestWebSocketMessageTypes(t *testing.T) {
	// Test different message types
	messageTypes := []string{"message", "event", "ping", "pong"}

	for _, msgType := range messageTypes {
		mockMsg := MockWebSocketMessage{
			Type:      msgType,
			Timestamp: time.Now().Unix(),
		}

		if mockMsg.Type != msgType {
			t.Errorf("Expected type %s, got %s", msgType, mockMsg.Type)
		}
	}
}

func TestWebSocketEventData(t *testing.T) {
	// Test event data structure
	eventData := map[string]interface{}{
		"user":     "alice#example.com",
		"group_id": "test-group",
		"action":   "joined",
	}

	mockMsg := MockWebSocketMessage{
		Type:      "event",
		Event:     "user_joined",
		Data:      eventData,
		Timestamp: time.Now().Unix(),
	}

	if mockMsg.Event != "user_joined" {
		t.Errorf("Expected event 'user_joined', got %s", mockMsg.Event)
	}

	if user, ok := mockMsg.Data["user"].(string); !ok || user != "alice#example.com" {
		t.Errorf("Expected user 'alice#example.com', got %v", mockMsg.Data["user"])
	}

	if groupID, ok := mockMsg.Data["group_id"].(string); !ok || groupID != "test-group" {
		t.Errorf("Expected group_id 'test-group', got %v", mockMsg.Data["group_id"])
	}
}

func TestWebSocketConfiguration(t *testing.T) {
	keyPair, _ := keymgmt.GenerateKeyPair()
	notificationManager := notifications.NewNotificationManager(5)
	defer notificationManager.Shutdown()

	wsClient := websocket.NewWebSocketClient("wss://secure.example.com", keyPair, notificationManager)

	// Test that client was created with secure URL
	if wsClient == nil {
		t.Fatal("Failed to create WebSocket client with secure URL")
	}

	// Test with HTTP URL (should be converted to WS)
	wsClient2 := websocket.NewWebSocketClient("http://example.com", keyPair, notificationManager)
	if wsClient2 == nil {
		t.Fatal("Failed to create WebSocket client with HTTP URL")
	}

	// Test with HTTPS URL (should be converted to WSS)
	wsClient3 := websocket.NewWebSocketClient("https://example.com", keyPair, notificationManager)
	if wsClient3 == nil {
		t.Fatal("Failed to create WebSocket client with HTTPS URL")
	}
}

func TestWebSocketClientState(t *testing.T) {
	keyPair, _ := keymgmt.GenerateKeyPair()
	notificationManager := notifications.NewNotificationManager(5)
	defer notificationManager.Shutdown()

	wsClient := websocket.NewWebSocketClient("ws://localhost:8080", keyPair, notificationManager)

	// Test initial state
	if wsClient.IsConnected() {
		t.Error("WebSocket should not be connected initially")
	}

	// Test that we can call methods without panicking
	err := wsClient.Disconnect()
	if err == nil {
		t.Error("Disconnect should fail when not connected")
	}
}

func TestWebSocketMessageSending(t *testing.T) {
	keyPair, _ := keymgmt.GenerateKeyPair()
	notificationManager := notifications.NewNotificationManager(5)
	defer notificationManager.Shutdown()

	wsClient := websocket.NewWebSocketClient("ws://localhost:8080", keyPair, notificationManager)

	testMsg := &message.Message{
		From:      "alice#example.com",
		To:        []string{"bob#example.com"},
		Subject:   "Test",
		Body:      "Test message",
		Timestamp: time.Now().Unix(),
	}

	// Should fail when not connected
	err := wsClient.SendMessage(testMsg)
	if err == nil {
		t.Error("SendMessage should fail when not connected")
	}
}
