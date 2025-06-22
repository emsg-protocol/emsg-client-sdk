package notifications

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/message"
)

// NotificationEvent represents different types of notification events
type NotificationEvent string

const (
	EventMessageReceived NotificationEvent = "message_received"
	EventMessageSent     NotificationEvent = "message_sent"
	EventUserJoined      NotificationEvent = "user_joined"
	EventUserLeft        NotificationEvent = "user_left"
	EventTyping          NotificationEvent = "typing"
	EventDeliveryReceipt NotificationEvent = "delivery_receipt"
)

// Notification represents a notification with metadata
type Notification struct {
	Event     NotificationEvent `json:"event"`
	Message   *message.Message  `json:"message,omitempty"`
	Timestamp int64             `json:"timestamp"`
	Metadata  map[string]any    `json:"metadata,omitempty"`
}

// NotificationHandler is a function that handles notifications
type NotificationHandler func(notification *Notification) error

// AsyncNotificationHandler is a function that handles notifications asynchronously
type AsyncNotificationHandler func(notification *Notification)

// NotificationManager manages notification hooks and delivery
type NotificationManager struct {
	handlers      map[NotificationEvent][]NotificationHandler
	asyncHandlers map[NotificationEvent][]AsyncNotificationHandler
	mutex         sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	workerPool    chan struct{} // Limits concurrent async handlers
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(maxConcurrentHandlers int) *NotificationManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &NotificationManager{
		handlers:      make(map[NotificationEvent][]NotificationHandler),
		asyncHandlers: make(map[NotificationEvent][]AsyncNotificationHandler),
		ctx:           ctx,
		cancel:        cancel,
		workerPool:    make(chan struct{}, maxConcurrentHandlers),
	}
}

// RegisterHandler registers a synchronous notification handler
func (nm *NotificationManager) RegisterHandler(event NotificationEvent, handler NotificationHandler) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()
	
	nm.handlers[event] = append(nm.handlers[event], handler)
}

// RegisterAsyncHandler registers an asynchronous notification handler
func (nm *NotificationManager) RegisterAsyncHandler(event NotificationEvent, handler AsyncNotificationHandler) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()
	
	nm.asyncHandlers[event] = append(nm.asyncHandlers[event], handler)
}

// UnregisterHandlers removes all handlers for a specific event
func (nm *NotificationManager) UnregisterHandlers(event NotificationEvent) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()
	
	delete(nm.handlers, event)
	delete(nm.asyncHandlers, event)
}

// Notify sends a notification to all registered handlers
func (nm *NotificationManager) Notify(notification *Notification) error {
	nm.mutex.RLock()
	syncHandlers := nm.handlers[notification.Event]
	asyncHandlers := nm.asyncHandlers[notification.Event]
	nm.mutex.RUnlock()

	// Execute synchronous handlers first
	for _, handler := range syncHandlers {
		if err := handler(notification); err != nil {
			log.Printf("Synchronous notification handler error: %v", err)
			return fmt.Errorf("notification handler failed: %w", err)
		}
	}

	// Execute asynchronous handlers
	for _, handler := range asyncHandlers {
		go nm.executeAsyncHandler(handler, notification)
	}

	return nil
}

// executeAsyncHandler executes an async handler with worker pool limiting
func (nm *NotificationManager) executeAsyncHandler(handler AsyncNotificationHandler, notification *Notification) {
	select {
	case nm.workerPool <- struct{}{}: // Acquire worker slot
		defer func() { <-nm.workerPool }() // Release worker slot
		
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Async notification handler panicked: %v", r)
			}
		}()
		
		handler(notification)
		
	case <-nm.ctx.Done():
		log.Printf("Notification manager shutting down, skipping async handler")
		return
	}
}

// NotifyMessageReceived is a convenience method for message received notifications
func (nm *NotificationManager) NotifyMessageReceived(msg *message.Message) error {
	notification := &Notification{
		Event:     EventMessageReceived,
		Message:   msg,
		Timestamp: time.Now().Unix(),
		Metadata:  make(map[string]any),
	}
	
	// Add message metadata
	if msg.IsSystemMessage() {
		notification.Metadata["is_system"] = true
		if systemMsg, err := msg.GetSystemMessage(); err == nil {
			notification.Metadata["system_type"] = systemMsg.Type
			notification.Metadata["actor"] = systemMsg.Actor
		}
	}
	
	if msg.IsEncrypted() {
		notification.Metadata["is_encrypted"] = true
	}
	
	return nm.Notify(notification)
}

// NotifyMessageSent is a convenience method for message sent notifications
func (nm *NotificationManager) NotifyMessageSent(msg *message.Message) error {
	notification := &Notification{
		Event:     EventMessageSent,
		Message:   msg,
		Timestamp: time.Now().Unix(),
		Metadata:  map[string]any{
			"recipients": msg.GetRecipients(),
		},
	}
	
	return nm.Notify(notification)
}

// NotifyUserJoined is a convenience method for user joined notifications
func (nm *NotificationManager) NotifyUserJoined(userAddress, groupID string) error {
	notification := &Notification{
		Event:     EventUserJoined,
		Timestamp: time.Now().Unix(),
		Metadata: map[string]any{
			"user":     userAddress,
			"group_id": groupID,
		},
	}
	
	return nm.Notify(notification)
}

// NotifyUserLeft is a convenience method for user left notifications
func (nm *NotificationManager) NotifyUserLeft(userAddress, groupID string) error {
	notification := &Notification{
		Event:     EventUserLeft,
		Timestamp: time.Now().Unix(),
		Metadata: map[string]any{
			"user":     userAddress,
			"group_id": groupID,
		},
	}
	
	return nm.Notify(notification)
}

// NotifyTyping is a convenience method for typing notifications
func (nm *NotificationManager) NotifyTyping(userAddress, groupID string, isTyping bool) error {
	notification := &Notification{
		Event:     EventTyping,
		Timestamp: time.Now().Unix(),
		Metadata: map[string]any{
			"user":       userAddress,
			"group_id":   groupID,
			"is_typing":  isTyping,
		},
	}
	
	return nm.Notify(notification)
}

// NotifyDeliveryReceipt is a convenience method for delivery receipt notifications
func (nm *NotificationManager) NotifyDeliveryReceipt(messageID, recipientAddress string, delivered bool) error {
	notification := &Notification{
		Event:     EventDeliveryReceipt,
		Timestamp: time.Now().Unix(),
		Metadata: map[string]any{
			"message_id": messageID,
			"recipient":  recipientAddress,
			"delivered":  delivered,
		},
	}
	
	return nm.Notify(notification)
}

// Shutdown gracefully shuts down the notification manager
func (nm *NotificationManager) Shutdown() {
	nm.cancel()
}

// GetHandlerCount returns the number of handlers for an event
func (nm *NotificationManager) GetHandlerCount(event NotificationEvent) int {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()
	
	return len(nm.handlers[event]) + len(nm.asyncHandlers[event])
}

// MessagePoller polls for new messages and triggers notifications
type MessagePoller struct {
	client              MessageClient
	notificationManager *NotificationManager
	pollInterval        time.Duration
	lastPollTime        time.Time
	ctx                 context.Context
	cancel              context.CancelFunc
	running             bool
	mutex               sync.Mutex
}

// MessageClient interface for polling messages
type MessageClient interface {
	GetMessages(address string) ([]*message.Message, error)
}

// NewMessagePoller creates a new message poller
func NewMessagePoller(client MessageClient, notificationManager *NotificationManager, pollInterval time.Duration) *MessagePoller {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MessagePoller{
		client:              client,
		notificationManager: notificationManager,
		pollInterval:        pollInterval,
		lastPollTime:        time.Now(),
		ctx:                 ctx,
		cancel:              cancel,
	}
}

// Start starts the message polling
func (mp *MessagePoller) Start(userAddress string) error {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	
	if mp.running {
		return fmt.Errorf("message poller is already running")
	}
	
	mp.running = true
	go mp.pollLoop(userAddress)
	
	return nil
}

// Stop stops the message polling
func (mp *MessagePoller) Stop() {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	
	if mp.running {
		mp.cancel()
		mp.running = false
	}
}

// IsRunning returns true if the poller is running
func (mp *MessagePoller) IsRunning() bool {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	return mp.running
}

// pollLoop is the main polling loop
func (mp *MessagePoller) pollLoop(userAddress string) {
	ticker := time.NewTicker(mp.pollInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mp.pollMessages(userAddress)
		case <-mp.ctx.Done():
			return
		}
	}
}

// pollMessages polls for new messages and triggers notifications
func (mp *MessagePoller) pollMessages(userAddress string) {
	messages, err := mp.client.GetMessages(userAddress)
	if err != nil {
		log.Printf("Failed to poll messages: %v", err)
		return
	}
	
	// Filter messages received since last poll
	newMessages := make([]*message.Message, 0)
	for _, msg := range messages {
		if msg.Timestamp > mp.lastPollTime.Unix() {
			newMessages = append(newMessages, msg)
		}
	}
	
	// Update last poll time
	mp.lastPollTime = time.Now()
	
	// Notify about new messages
	for _, msg := range newMessages {
		if err := mp.notificationManager.NotifyMessageReceived(msg); err != nil {
			log.Printf("Failed to notify message received: %v", err)
		}
	}
}
