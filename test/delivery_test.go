package test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/delivery"
	"github.com/emsg-protocol/emsg-client-sdk/message"
)

func TestDeliveryTracker(t *testing.T) {
	tracker := delivery.NewDeliveryTracker(nil) // Use default strategy

	// Create test message
	testMsg := &message.Message{
		MessageID: "test-msg-123",
		From:      "alice#example.com",
		To:        []string{"bob#example.com"},
		Subject:   "Test",
		Body:      "Test message",
		Timestamp: time.Now().Unix(),
	}

	// Track the message
	receipt := tracker.TrackMessage(testMsg)
	if receipt == nil {
		t.Fatal("Failed to track message")
	}

	if receipt.MessageID != testMsg.MessageID {
		t.Errorf("Expected message ID %s, got %s", testMsg.MessageID, receipt.MessageID)
	}

	if receipt.Status != delivery.StatusPending {
		t.Errorf("Expected status %s, got %s", delivery.StatusPending, receipt.Status)
	}

	if receipt.Recipient != "bob#example.com" {
		t.Errorf("Expected recipient bob#example.com, got %s", receipt.Recipient)
	}

	// Update status to sent
	err := tracker.UpdateDeliveryStatus(testMsg.MessageID, delivery.StatusSent, "")
	if err != nil {
		t.Fatalf("Failed to update delivery status: %v", err)
	}

	// Retrieve updated receipt
	updatedReceipt, err := tracker.GetDeliveryReceipt(testMsg.MessageID)
	if err != nil {
		t.Fatalf("Failed to get delivery receipt: %v", err)
	}

	if updatedReceipt.Status != delivery.StatusSent {
		t.Errorf("Expected status %s, got %s", delivery.StatusSent, updatedReceipt.Status)
	}

	if updatedReceipt.AttemptCount != 1 {
		t.Errorf("Expected attempt count 1, got %d", updatedReceipt.AttemptCount)
	}
}

func TestDeliveryStatus(t *testing.T) {
	statuses := []delivery.DeliveryStatus{
		delivery.StatusPending,
		delivery.StatusSent,
		delivery.StatusDelivered,
		delivery.StatusFailed,
		delivery.StatusRetrying,
		delivery.StatusExpired,
	}

	expectedStatuses := []string{
		"pending",
		"sent",
		"delivered",
		"failed",
		"retrying",
		"expired",
	}

	for i, status := range statuses {
		if string(status) != expectedStatuses[i] {
			t.Errorf("Expected status %s, got %s", expectedStatuses[i], string(status))
		}
	}
}

func TestDeliveryRetryStrategy(t *testing.T) {
	strategy := delivery.DefaultRetryStrategy()

	if strategy.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", strategy.MaxRetries)
	}

	if strategy.InitialDelay != 2*time.Second {
		t.Errorf("Expected InitialDelay 2s, got %v", strategy.InitialDelay)
	}

	if strategy.MaxDelay != 5*time.Minute {
		t.Errorf("Expected MaxDelay 5m, got %v", strategy.MaxDelay)
	}

	if strategy.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor 2.0, got %f", strategy.BackoffFactor)
	}

	if strategy.ExpirationTime != 24*time.Hour {
		t.Errorf("Expected ExpirationTime 24h, got %v", strategy.ExpirationTime)
	}

	if !strategy.RetryOnFailure {
		t.Error("Expected RetryOnFailure to be true")
	}

	if !strategy.RetryOnTimeout {
		t.Error("Expected RetryOnTimeout to be true")
	}
}

func TestDeliveryCallbacks(t *testing.T) {
	tracker := delivery.NewDeliveryTracker(nil)

	var callbackReceived bool
	var receivedReceipt *delivery.DeliveryReceipt
	var mutex sync.Mutex

	callback := func(receipt *delivery.DeliveryReceipt) {
		mutex.Lock()
		defer mutex.Unlock()
		callbackReceived = true
		receivedReceipt = receipt
	}

	// Create test message
	testMsg := &message.Message{
		MessageID: "test-msg-456",
		From:      "alice#example.com",
		To:        []string{"bob#example.com"},
		Subject:   "Test",
		Body:      "Test message",
		Timestamp: time.Now().Unix(),
	}

	// Track message and register callback
	tracker.TrackMessage(testMsg)
	tracker.RegisterCallback(testMsg.MessageID, callback)

	// Update status (should trigger callback)
	err := tracker.UpdateDeliveryStatus(testMsg.MessageID, delivery.StatusDelivered, "")
	if err != nil {
		t.Fatalf("Failed to update delivery status: %v", err)
	}

	// Wait a bit for async callback
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	defer mutex.Unlock()

	if !callbackReceived {
		t.Error("Callback was not triggered")
	}

	if receivedReceipt == nil {
		t.Error("Received receipt is nil")
	} else if receivedReceipt.Status != delivery.StatusDelivered {
		t.Errorf("Expected status %s, got %s", delivery.StatusDelivered, receivedReceipt.Status)
	}
}

func TestGlobalDeliveryCallback(t *testing.T) {
	tracker := delivery.NewDeliveryTracker(nil)

	var callbackCount int
	var mutex sync.Mutex

	globalCallback := func(receipt *delivery.DeliveryReceipt) {
		mutex.Lock()
		defer mutex.Unlock()
		callbackCount++
	}

	tracker.RegisterGlobalCallback(globalCallback)

	// Create and track multiple messages
	for i := 0; i < 3; i++ {
		testMsg := &message.Message{
			MessageID: fmt.Sprintf("test-msg-%d", i),
			From:      "alice#example.com",
			To:        []string{"bob#example.com"},
			Subject:   "Test",
			Body:      "Test message",
			Timestamp: time.Now().Unix(),
		}

		tracker.TrackMessage(testMsg)
		tracker.UpdateDeliveryStatus(testMsg.MessageID, delivery.StatusSent, "")
	}

	// Wait for async callbacks
	time.Sleep(200 * time.Millisecond)

	mutex.Lock()
	defer mutex.Unlock()

	if callbackCount != 3 {
		t.Errorf("Expected 3 global callbacks, got %d", callbackCount)
	}
}

func TestDeliveryStats(t *testing.T) {
	tracker := delivery.NewDeliveryTracker(nil)

	// Create test messages with different statuses
	messages := []struct {
		id     string
		status delivery.DeliveryStatus
	}{
		{"msg-1", delivery.StatusSent},
		{"msg-2", delivery.StatusDelivered},
		{"msg-3", delivery.StatusFailed},
		{"msg-4", delivery.StatusSent},
		{"msg-5", delivery.StatusDelivered},
	}

	for _, msg := range messages {
		testMsg := &message.Message{
			MessageID: msg.id,
			From:      "alice#example.com",
			To:        []string{"bob#example.com"},
			Subject:   "Test",
			Body:      "Test message",
			Timestamp: time.Now().Unix(),
		}

		tracker.TrackMessage(testMsg)
		tracker.UpdateDeliveryStatus(msg.id, msg.status, "")
	}

	stats := tracker.GetDeliveryStats()

	if stats[delivery.StatusSent] != 2 {
		t.Errorf("Expected 2 sent messages, got %d", stats[delivery.StatusSent])
	}

	if stats[delivery.StatusDelivered] != 2 {
		t.Errorf("Expected 2 delivered messages, got %d", stats[delivery.StatusDelivered])
	}

	if stats[delivery.StatusFailed] != 1 {
		t.Errorf("Expected 1 failed message, got %d", stats[delivery.StatusFailed])
	}
}

func TestPendingRetries(t *testing.T) {
	strategy := &delivery.RetryStrategy{
		MaxRetries:     3,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       1 * time.Second,
		BackoffFactor:  2.0,
		ExpirationTime: 1 * time.Hour,
		RetryOnFailure: true,
		RetryOnTimeout: true,
	}

	tracker := delivery.NewDeliveryTracker(strategy)

	// Create test message
	testMsg := &message.Message{
		MessageID: "retry-msg-123",
		From:      "alice#example.com",
		To:        []string{"bob#example.com"},
		Subject:   "Test",
		Body:      "Test message",
		Timestamp: time.Now().Unix(),
	}

	tracker.TrackMessage(testMsg)
	tracker.UpdateDeliveryStatus(testMsg.MessageID, delivery.StatusRetrying, "Network error")

	// Initially, no retries should be pending (next attempt is in the future)
	pendingRetries := tracker.GetPendingRetries()
	if len(pendingRetries) != 0 {
		t.Errorf("Expected 0 pending retries initially, got %d", len(pendingRetries))
	}

	// Wait for retry time to pass
	time.Sleep(150 * time.Millisecond)

	pendingRetries = tracker.GetPendingRetries()
	if len(pendingRetries) != 1 {
		t.Errorf("Expected 1 pending retry after delay, got %d", len(pendingRetries))
	}

	if pendingRetries[0].MessageID != testMsg.MessageID {
		t.Errorf("Expected message ID %s, got %s", testMsg.MessageID, pendingRetries[0].MessageID)
	}
}

func TestDeliveryReceiptSerialization(t *testing.T) {
	receipt := &delivery.DeliveryReceipt{
		MessageID:    "test-123",
		Recipient:    "bob#example.com",
		Status:       delivery.StatusDelivered,
		Timestamp:    time.Now().Unix(),
		AttemptCount: 2,
		LastAttempt:  time.Now().Unix(),
		Metadata:     map[string]any{"test": "value"},
	}

	// Test JSON serialization
	jsonData, err := receipt.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize receipt to JSON: %v", err)
	}

	// Test JSON deserialization
	deserializedReceipt, err := delivery.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize receipt from JSON: %v", err)
	}

	if deserializedReceipt.MessageID != receipt.MessageID {
		t.Errorf("Expected message ID %s, got %s", receipt.MessageID, deserializedReceipt.MessageID)
	}

	if deserializedReceipt.Status != receipt.Status {
		t.Errorf("Expected status %s, got %s", receipt.Status, deserializedReceipt.Status)
	}

	if deserializedReceipt.AttemptCount != receipt.AttemptCount {
		t.Errorf("Expected attempt count %d, got %d", receipt.AttemptCount, deserializedReceipt.AttemptCount)
	}
}

func TestDeliveryReceiptMethods(t *testing.T) {
	// Test terminal statuses
	terminalReceipt := &delivery.DeliveryReceipt{
		Status: delivery.StatusDelivered,
	}

	if !terminalReceipt.IsTerminal() {
		t.Error("Delivered status should be terminal")
	}

	// Test non-terminal status
	pendingReceipt := &delivery.DeliveryReceipt{
		Status: delivery.StatusPending,
	}

	if pendingReceipt.IsTerminal() {
		t.Error("Pending status should not be terminal")
	}

	// Test retryable status
	retryingReceipt := &delivery.DeliveryReceipt{
		Status: delivery.StatusRetrying,
	}

	if !retryingReceipt.IsRetryable() {
		t.Error("Retrying status should be retryable")
	}

	// Test non-retryable status
	deliveredReceipt := &delivery.DeliveryReceipt{
		Status: delivery.StatusDelivered,
	}

	if deliveredReceipt.IsRetryable() {
		t.Error("Delivered status should not be retryable")
	}
}
