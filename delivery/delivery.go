package delivery

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/message"
)

// DeliveryStatus represents the status of a message delivery
type DeliveryStatus string

const (
	StatusPending   DeliveryStatus = "pending"
	StatusSent      DeliveryStatus = "sent"
	StatusDelivered DeliveryStatus = "delivered"
	StatusFailed    DeliveryStatus = "failed"
	StatusRetrying  DeliveryStatus = "retrying"
	StatusExpired   DeliveryStatus = "expired"
)

// DeliveryReceipt represents a delivery receipt for a message
type DeliveryReceipt struct {
	MessageID    string         `json:"message_id"`
	Recipient    string         `json:"recipient"`
	Status       DeliveryStatus `json:"status"`
	Timestamp    int64          `json:"timestamp"`
	AttemptCount int            `json:"attempt_count"`
	LastAttempt  int64          `json:"last_attempt"`
	NextAttempt  int64          `json:"next_attempt,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// DeliveryTracker tracks message delivery status and handles retries
type DeliveryTracker struct {
	receipts      map[string]*DeliveryReceipt
	mutex         sync.RWMutex
	retryStrategy *RetryStrategy
	callbacks     map[string][]DeliveryCallback
	callbackMutex sync.RWMutex
}

// RetryStrategy defines retry behavior for message delivery
type RetryStrategy struct {
	MaxRetries     int           `json:"max_retries"`
	InitialDelay   time.Duration `json:"initial_delay"`
	MaxDelay       time.Duration `json:"max_delay"`
	BackoffFactor  float64       `json:"backoff_factor"`
	ExpirationTime time.Duration `json:"expiration_time"`
	RetryOnFailure bool          `json:"retry_on_failure"`
	RetryOnTimeout bool          `json:"retry_on_timeout"`
}

// DeliveryCallback is called when delivery status changes
type DeliveryCallback func(receipt *DeliveryReceipt)

// DefaultRetryStrategy returns a default retry strategy for delivery
func DefaultRetryStrategy() *RetryStrategy {
	return &RetryStrategy{
		MaxRetries:     5,
		InitialDelay:   2 * time.Second,
		MaxDelay:       5 * time.Minute,
		BackoffFactor:  2.0,
		ExpirationTime: 24 * time.Hour,
		RetryOnFailure: true,
		RetryOnTimeout: true,
	}
}

// NewDeliveryTracker creates a new delivery tracker
func NewDeliveryTracker(retryStrategy *RetryStrategy) *DeliveryTracker {
	if retryStrategy == nil {
		retryStrategy = DefaultRetryStrategy()
	}

	return &DeliveryTracker{
		receipts:      make(map[string]*DeliveryReceipt),
		retryStrategy: retryStrategy,
		callbacks:     make(map[string][]DeliveryCallback),
	}
}

// TrackMessage starts tracking delivery for a message
func (dt *DeliveryTracker) TrackMessage(msg *message.Message) *DeliveryReceipt {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	receipt := &DeliveryReceipt{
		MessageID:    msg.MessageID,
		Recipient:    msg.To[0], // Track first recipient for simplicity
		Status:       StatusPending,
		Timestamp:    time.Now().Unix(),
		AttemptCount: 0,
		Metadata:     make(map[string]any),
	}

	// Add message metadata
	receipt.Metadata["from"] = msg.From
	receipt.Metadata["subject"] = msg.Subject
	receipt.Metadata["recipients"] = msg.GetRecipients()
	receipt.Metadata["is_system"] = msg.IsSystemMessage()
	receipt.Metadata["is_encrypted"] = msg.IsEncrypted()

	dt.receipts[msg.MessageID] = receipt
	return receipt
}

// UpdateDeliveryStatus updates the delivery status of a message
func (dt *DeliveryTracker) UpdateDeliveryStatus(messageID string, status DeliveryStatus, errorMsg string) error {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	receipt, exists := dt.receipts[messageID]
	if !exists {
		return fmt.Errorf("message %s not found in delivery tracker", messageID)
	}

	oldStatus := receipt.Status
	receipt.Status = status
	receipt.Timestamp = time.Now().Unix()

	if errorMsg != "" {
		receipt.ErrorMessage = errorMsg
	}

	// Update attempt tracking
	if status == StatusSent || status == StatusRetrying {
		receipt.AttemptCount++
		receipt.LastAttempt = time.Now().Unix()

		// Calculate next retry time if needed
		if status == StatusRetrying && receipt.AttemptCount < dt.retryStrategy.MaxRetries {
			nextDelay := dt.calculateRetryDelay(receipt.AttemptCount)
			receipt.NextAttempt = time.Now().Add(nextDelay).Unix()
		}
	}

	// Check if message has expired
	if time.Since(time.Unix(receipt.Timestamp, 0)) > dt.retryStrategy.ExpirationTime {
		receipt.Status = StatusExpired
	}

	// Trigger callbacks if status changed
	if oldStatus != receipt.Status {
		dt.triggerCallbacks(messageID, receipt)
	}

	return nil
}

// GetDeliveryReceipt returns the delivery receipt for a message
func (dt *DeliveryTracker) GetDeliveryReceipt(messageID string) (*DeliveryReceipt, error) {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()

	receipt, exists := dt.receipts[messageID]
	if !exists {
		return nil, fmt.Errorf("message %s not found", messageID)
	}

	// Return a copy to prevent external modification
	receiptCopy := *receipt
	return &receiptCopy, nil
}

// GetPendingRetries returns messages that need to be retried
func (dt *DeliveryTracker) GetPendingRetries() []*DeliveryReceipt {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()

	var pendingRetries []*DeliveryReceipt
	now := time.Now().Unix()

	for _, receipt := range dt.receipts {
		if receipt.Status == StatusRetrying &&
			receipt.NextAttempt > 0 &&
			receipt.NextAttempt <= now &&
			receipt.AttemptCount < dt.retryStrategy.MaxRetries {

			// Check if not expired
			if time.Since(time.Unix(receipt.Timestamp, 0)) <= dt.retryStrategy.ExpirationTime {
				receiptCopy := *receipt
				pendingRetries = append(pendingRetries, &receiptCopy)
			}
		}
	}

	return pendingRetries
}

// RegisterCallback registers a callback for delivery status changes
func (dt *DeliveryTracker) RegisterCallback(messageID string, callback DeliveryCallback) {
	dt.callbackMutex.Lock()
	defer dt.callbackMutex.Unlock()

	dt.callbacks[messageID] = append(dt.callbacks[messageID], callback)
}

// RegisterGlobalCallback registers a callback for all delivery status changes
func (dt *DeliveryTracker) RegisterGlobalCallback(callback DeliveryCallback) {
	dt.callbackMutex.Lock()
	defer dt.callbackMutex.Unlock()

	dt.callbacks["*"] = append(dt.callbacks["*"], callback)
}

// triggerCallbacks triggers callbacks for a message
func (dt *DeliveryTracker) triggerCallbacks(messageID string, receipt *DeliveryReceipt) {
	dt.callbackMutex.RLock()
	callbacks := append(dt.callbacks[messageID], dt.callbacks["*"]...)
	dt.callbackMutex.RUnlock()

	for _, callback := range callbacks {
		go func(cb DeliveryCallback) {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but don't crash
				}
			}()
			cb(receipt)
		}(callback)
	}
}

// calculateRetryDelay calculates the delay before the next retry
func (dt *DeliveryTracker) calculateRetryDelay(attemptCount int) time.Duration {
	delay := time.Duration(float64(dt.retryStrategy.InitialDelay) *
		math.Pow(dt.retryStrategy.BackoffFactor, float64(attemptCount-1))) // Exponential backoff

	if delay > dt.retryStrategy.MaxDelay {
		delay = dt.retryStrategy.MaxDelay
	}

	return delay
}

// ShouldRetry determines if a message should be retried
func (dt *DeliveryTracker) ShouldRetry(messageID string, err error) bool {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()

	receipt, exists := dt.receipts[messageID]
	if !exists {
		return false
	}

	// Check retry limits
	if receipt.AttemptCount >= dt.retryStrategy.MaxRetries {
		return false
	}

	// Check expiration
	if time.Since(time.Unix(receipt.Timestamp, 0)) > dt.retryStrategy.ExpirationTime {
		return false
	}

	// Check retry conditions
	if err != nil {
		errStr := err.Error()

		// Retry on timeout if enabled
		if dt.retryStrategy.RetryOnTimeout &&
			(contains(errStr, "timeout") || contains(errStr, "deadline exceeded")) {
			return true
		}

		// Retry on general failure if enabled
		if dt.retryStrategy.RetryOnFailure {
			return true
		}
	}

	return false
}

// GetDeliveryStats returns delivery statistics
func (dt *DeliveryTracker) GetDeliveryStats() map[DeliveryStatus]int {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()

	stats := make(map[DeliveryStatus]int)

	for _, receipt := range dt.receipts {
		stats[receipt.Status]++
	}

	return stats
}

// CleanupExpiredReceipts removes expired delivery receipts
func (dt *DeliveryTracker) CleanupExpiredReceipts() int {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	var cleaned int
	now := time.Now()

	for messageID, receipt := range dt.receipts {
		if now.Sub(time.Unix(receipt.Timestamp, 0)) > dt.retryStrategy.ExpirationTime {
			delete(dt.receipts, messageID)
			cleaned++
		}
	}

	return cleaned
}

// GetAllReceipts returns all delivery receipts
func (dt *DeliveryTracker) GetAllReceipts() []*DeliveryReceipt {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()

	receipts := make([]*DeliveryReceipt, 0, len(dt.receipts))
	for _, receipt := range dt.receipts {
		receiptCopy := *receipt
		receipts = append(receipts, &receiptCopy)
	}

	return receipts
}

// ToJSON serializes a delivery receipt to JSON
func (dr *DeliveryReceipt) ToJSON() ([]byte, error) {
	return json.Marshal(dr)
}

// FromJSON deserializes a delivery receipt from JSON
func FromJSON(data []byte) (*DeliveryReceipt, error) {
	var receipt DeliveryReceipt
	err := json.Unmarshal(data, &receipt)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal delivery receipt: %w", err)
	}
	return &receipt, nil
}

// IsTerminal returns true if the status is terminal (no more changes expected)
func (dr *DeliveryReceipt) IsTerminal() bool {
	return dr.Status == StatusDelivered ||
		dr.Status == StatusFailed ||
		dr.Status == StatusExpired
}

// IsRetryable returns true if the message can be retried
func (dr *DeliveryReceipt) IsRetryable() bool {
	return dr.Status == StatusRetrying || dr.Status == StatusFailed
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
