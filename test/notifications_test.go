package test

import (
	"sync"
	"testing"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/message"
	"github.com/emsg-protocol/emsg-client-sdk/notifications"
)

func TestNotificationManager(t *testing.T) {
	nm := notifications.NewNotificationManager(5)
	defer nm.Shutdown()

	// Test handler registration
	var receivedNotifications []*notifications.Notification
	var mutex sync.Mutex

	handler := func(notification *notifications.Notification) error {
		mutex.Lock()
		defer mutex.Unlock()
		receivedNotifications = append(receivedNotifications, notification)
		return nil
	}

	nm.RegisterHandler(notifications.EventMessageReceived, handler)

	// Test notification
	testNotification := &notifications.Notification{
		Event:     notifications.EventMessageReceived,
		Timestamp: time.Now().Unix(),
		Metadata:  map[string]any{"test": "value"},
	}

	err := nm.Notify(testNotification)
	if err != nil {
		t.Fatalf("Failed to notify: %v", err)
	}

	// Verify notification was received
	mutex.Lock()
	if len(receivedNotifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(receivedNotifications))
	}
	mutex.Unlock()

	// Test handler count
	count := nm.GetHandlerCount(notifications.EventMessageReceived)
	if count != 1 {
		t.Errorf("Expected 1 handler, got %d", count)
	}

	// Test unregister
	nm.UnregisterHandlers(notifications.EventMessageReceived)
	count = nm.GetHandlerCount(notifications.EventMessageReceived)
	if count != 0 {
		t.Errorf("Expected 0 handlers after unregister, got %d", count)
	}
}

func TestAsyncNotificationHandler(t *testing.T) {
	nm := notifications.NewNotificationManager(5)
	defer nm.Shutdown()

	var receivedCount int
	var mutex sync.Mutex
	var wg sync.WaitGroup

	asyncHandler := func(notification *notifications.Notification) {
		defer wg.Done()
		mutex.Lock()
		receivedCount++
		mutex.Unlock()
	}

	nm.RegisterAsyncHandler(notifications.EventMessageReceived, asyncHandler)

	// Send multiple notifications
	numNotifications := 3
	wg.Add(numNotifications)

	for i := 0; i < numNotifications; i++ {
		testNotification := &notifications.Notification{
			Event:     notifications.EventMessageReceived,
			Timestamp: time.Now().Unix(),
		}
		nm.Notify(testNotification)
	}

	// Wait for all async handlers to complete
	wg.Wait()

	mutex.Lock()
	if receivedCount != numNotifications {
		t.Errorf("Expected %d async notifications, got %d", numNotifications, receivedCount)
	}
	mutex.Unlock()
}

func TestNotifyMessageReceived(t *testing.T) {
	nm := notifications.NewNotificationManager(5)
	defer nm.Shutdown()

	var receivedNotification *notifications.Notification
	var mutex sync.Mutex

	handler := func(notification *notifications.Notification) error {
		mutex.Lock()
		defer mutex.Unlock()
		receivedNotification = notification
		return nil
	}

	nm.RegisterHandler(notifications.EventMessageReceived, handler)

	// Create test message
	testMsg := &message.Message{
		From:      "alice#example.com",
		To:        []string{"bob#example.com"},
		Subject:   "Test",
		Body:      "Test message",
		Timestamp: time.Now().Unix(),
	}

	err := nm.NotifyMessageReceived(testMsg)
	if err != nil {
		t.Fatalf("Failed to notify message received: %v", err)
	}

	// Verify notification
	mutex.Lock()
	defer mutex.Unlock()

	if receivedNotification == nil {
		t.Fatal("No notification received")
	}

	if receivedNotification.Event != notifications.EventMessageReceived {
		t.Errorf("Expected event %s, got %s", notifications.EventMessageReceived, receivedNotification.Event)
	}

	if receivedNotification.Message != testMsg {
		t.Error("Message in notification doesn't match original")
	}

	if receivedNotification.Metadata == nil {
		t.Error("Notification metadata is nil")
	}
}

func TestNotifySystemMessage(t *testing.T) {
	nm := notifications.NewNotificationManager(5)
	defer nm.Shutdown()

	var receivedNotification *notifications.Notification
	var mutex sync.Mutex

	handler := func(notification *notifications.Notification) error {
		mutex.Lock()
		defer mutex.Unlock()
		receivedNotification = notification
		return nil
	}

	nm.RegisterHandler(notifications.EventMessageReceived, handler)

	// Create system message
	systemMsg, err := message.NewUserJoinedMessage(
		"system#example.com",
		[]string{"group#example.com"},
		"alice#example.com",
		"test-group",
	)
	if err != nil {
		t.Fatalf("Failed to create system message: %v", err)
	}

	err = nm.NotifyMessageReceived(systemMsg)
	if err != nil {
		t.Fatalf("Failed to notify system message: %v", err)
	}

	// Verify notification metadata for system message
	mutex.Lock()
	defer mutex.Unlock()

	if receivedNotification == nil {
		t.Fatal("No notification received")
	}

	isSystem, exists := receivedNotification.Metadata["is_system"]
	if !exists || !isSystem.(bool) {
		t.Error("System message metadata not set correctly")
	}

	systemType, exists := receivedNotification.Metadata["system_type"]
	if !exists || systemType != message.SystemJoined {
		t.Errorf("Expected system type %s, got %v", message.SystemJoined, systemType)
	}

	actor, exists := receivedNotification.Metadata["actor"]
	if !exists || actor != "alice#example.com" {
		t.Errorf("Expected actor alice#example.com, got %v", actor)
	}
}

func TestNotifyUserJoined(t *testing.T) {
	nm := notifications.NewNotificationManager(5)
	defer nm.Shutdown()

	var receivedNotification *notifications.Notification
	var mutex sync.Mutex

	handler := func(notification *notifications.Notification) error {
		mutex.Lock()
		defer mutex.Unlock()
		receivedNotification = notification
		return nil
	}

	nm.RegisterHandler(notifications.EventUserJoined, handler)

	err := nm.NotifyUserJoined("alice#example.com", "test-group")
	if err != nil {
		t.Fatalf("Failed to notify user joined: %v", err)
	}

	// Verify notification
	mutex.Lock()
	defer mutex.Unlock()

	if receivedNotification == nil {
		t.Fatal("No notification received")
	}

	if receivedNotification.Event != notifications.EventUserJoined {
		t.Errorf("Expected event %s, got %s", notifications.EventUserJoined, receivedNotification.Event)
	}

	user, exists := receivedNotification.Metadata["user"]
	if !exists || user != "alice#example.com" {
		t.Errorf("Expected user alice#example.com, got %v", user)
	}

	groupID, exists := receivedNotification.Metadata["group_id"]
	if !exists || groupID != "test-group" {
		t.Errorf("Expected group_id test-group, got %v", groupID)
	}
}

func TestNotifyTyping(t *testing.T) {
	nm := notifications.NewNotificationManager(5)
	defer nm.Shutdown()

	var receivedNotification *notifications.Notification
	var mutex sync.Mutex

	handler := func(notification *notifications.Notification) error {
		mutex.Lock()
		defer mutex.Unlock()
		receivedNotification = notification
		return nil
	}

	nm.RegisterHandler(notifications.EventTyping, handler)

	err := nm.NotifyTyping("alice#example.com", "test-group", true)
	if err != nil {
		t.Fatalf("Failed to notify typing: %v", err)
	}

	// Verify notification
	mutex.Lock()
	defer mutex.Unlock()

	if receivedNotification == nil {
		t.Fatal("No notification received")
	}

	if receivedNotification.Event != notifications.EventTyping {
		t.Errorf("Expected event %s, got %s", notifications.EventTyping, receivedNotification.Event)
	}

	isTyping, exists := receivedNotification.Metadata["is_typing"]
	if !exists || !isTyping.(bool) {
		t.Error("Expected is_typing to be true")
	}
}

func TestNotifyDeliveryReceipt(t *testing.T) {
	nm := notifications.NewNotificationManager(5)
	defer nm.Shutdown()

	var receivedNotification *notifications.Notification
	var mutex sync.Mutex

	handler := func(notification *notifications.Notification) error {
		mutex.Lock()
		defer mutex.Unlock()
		receivedNotification = notification
		return nil
	}

	nm.RegisterHandler(notifications.EventDeliveryReceipt, handler)

	err := nm.NotifyDeliveryReceipt("msg-123", "bob#example.com", true)
	if err != nil {
		t.Fatalf("Failed to notify delivery receipt: %v", err)
	}

	// Verify notification
	mutex.Lock()
	defer mutex.Unlock()

	if receivedNotification == nil {
		t.Fatal("No notification received")
	}

	if receivedNotification.Event != notifications.EventDeliveryReceipt {
		t.Errorf("Expected event %s, got %s", notifications.EventDeliveryReceipt, receivedNotification.Event)
	}

	messageID, exists := receivedNotification.Metadata["message_id"]
	if !exists || messageID != "msg-123" {
		t.Errorf("Expected message_id msg-123, got %v", messageID)
	}

	delivered, exists := receivedNotification.Metadata["delivered"]
	if !exists || !delivered.(bool) {
		t.Error("Expected delivered to be true")
	}
}
