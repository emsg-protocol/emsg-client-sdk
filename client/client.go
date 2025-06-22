package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/attachments"
	"github.com/emsg-protocol/emsg-client-sdk/auth"
	"github.com/emsg-protocol/emsg-client-sdk/delivery"
	"github.com/emsg-protocol/emsg-client-sdk/dns"
	"github.com/emsg-protocol/emsg-client-sdk/encryption"
	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
	"github.com/emsg-protocol/emsg-client-sdk/message"
	"github.com/emsg-protocol/emsg-client-sdk/notifications"
	"github.com/emsg-protocol/emsg-client-sdk/utils"
	"github.com/emsg-protocol/emsg-client-sdk/websocket"
)

// RetryStrategy defines retry behavior for rate limiting
type RetryStrategy struct {
	MaxRetries     int           // Maximum number of retries
	InitialDelay   time.Duration // Initial delay before first retry
	MaxDelay       time.Duration // Maximum delay between retries
	BackoffFactor  float64       // Exponential backoff factor
	RetryOn429     bool          // Retry on HTTP 429 (rate limit)
	RetryOnTimeout bool          // Retry on timeout errors
}

// DefaultRetryStrategy returns a default retry strategy
func DefaultRetryStrategy() *RetryStrategy {
	return &RetryStrategy{
		MaxRetries:     3,
		InitialDelay:   1 * time.Second,
		MaxDelay:       30 * time.Second,
		BackoffFactor:  2.0,
		RetryOn429:     true,
		RetryOnTimeout: true,
	}
}

// Client represents the EMSG client SDK
type Client struct {
	keyPair             *keymgmt.KeyPair
	resolver            *dns.CachedResolver
	httpClient          *http.Client
	userAgent           string
	retryStrategy       *RetryStrategy
	beforeSend          func(*message.Message) error
	afterSend           func(*message.Message, *http.Response) error
	encryptionManager   *encryption.EncryptionManager
	notificationManager *notifications.NotificationManager
	messagePoller       *notifications.MessagePoller
	webSocketClient     *websocket.WebSocketClient
	deliveryTracker     *delivery.DeliveryTracker
	attachmentManager   *attachments.AttachmentManager
}

// Config holds configuration for the EMSG client
type Config struct {
	KeyPair                *keymgmt.KeyPair
	Timeout                time.Duration
	UserAgent              string
	DNSConfig              *dns.ResolverConfig
	DNSTTL                 time.Duration
	RetryStrategy          *RetryStrategy
	BeforeSend             func(*message.Message) error
	AfterSend              func(*message.Message, *http.Response) error
	EncryptionConfig       *encryption.EncryptionConfig
	EnableNotifications    bool
	NotificationHandlers   map[notifications.NotificationEvent][]notifications.NotificationHandler
	AsyncHandlers          map[notifications.NotificationEvent][]notifications.AsyncNotificationHandler
	PollInterval           time.Duration
	EnableWebSocket        bool
	WebSocketConfig        *websocket.ReconnectStrategy
	EnableDeliveryTracking bool
	DeliveryRetryStrategy  *delivery.RetryStrategy
	AttachmentConfig       *attachments.AttachmentConfig
}

// DefaultConfig returns a default client configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:                30 * time.Second,
		UserAgent:              "emsg-client-sdk/1.0",
		DNSConfig:              dns.DefaultResolverConfig(),
		DNSTTL:                 5 * time.Minute,
		RetryStrategy:          DefaultRetryStrategy(),
		EncryptionConfig:       encryption.DefaultEncryptionConfig(),
		EnableNotifications:    false,
		NotificationHandlers:   make(map[notifications.NotificationEvent][]notifications.NotificationHandler),
		AsyncHandlers:          make(map[notifications.NotificationEvent][]notifications.AsyncNotificationHandler),
		PollInterval:           30 * time.Second,
		EnableWebSocket:        false,
		WebSocketConfig:        websocket.DefaultReconnectStrategy(),
		EnableDeliveryTracking: false,
		DeliveryRetryStrategy:  delivery.DefaultRetryStrategy(),
		AttachmentConfig:       attachments.DefaultAttachmentConfig(),
	}
}

// New creates a new EMSG client with the given configuration
func New(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	resolver := dns.NewCachedResolver(config.DNSConfig, config.DNSTTL)

	retryStrategy := config.RetryStrategy
	if retryStrategy == nil {
		retryStrategy = DefaultRetryStrategy()
	}

	client := &Client{
		keyPair:       config.KeyPair,
		resolver:      resolver,
		httpClient:    httpClient,
		userAgent:     config.UserAgent,
		retryStrategy: retryStrategy,
		beforeSend:    config.BeforeSend,
		afterSend:     config.AfterSend,
	}

	// Initialize encryption manager if encryption is enabled
	if config.EncryptionConfig != nil && config.EncryptionConfig.Enabled && config.EncryptionConfig.KeyPair != nil {
		client.encryptionManager = encryption.NewEncryptionManager(
			config.EncryptionConfig.KeyPair,
			config.EncryptionConfig.KeyStore,
		)
	}

	// Initialize notification manager if notifications are enabled
	if config.EnableNotifications {
		client.notificationManager = notifications.NewNotificationManager(10) // Max 10 concurrent handlers

		// Register handlers from config
		for event, handlers := range config.NotificationHandlers {
			for _, handler := range handlers {
				client.notificationManager.RegisterHandler(event, handler)
			}
		}

		for event, handlers := range config.AsyncHandlers {
			for _, handler := range handlers {
				client.notificationManager.RegisterAsyncHandler(event, handler)
			}
		}

		// Initialize message poller
		client.messagePoller = notifications.NewMessagePoller(client, client.notificationManager, config.PollInterval)
	}

	// Initialize delivery tracker if enabled
	if config.EnableDeliveryTracking {
		client.deliveryTracker = delivery.NewDeliveryTracker(config.DeliveryRetryStrategy)
	}

	// Initialize attachment manager
	if config.AttachmentConfig != nil {
		attachmentManager, err := attachments.NewAttachmentManager(config.AttachmentConfig)
		if err != nil {
			log.Printf("Warning: failed to initialize attachment manager: %v", err)
		} else {
			client.attachmentManager = attachmentManager
		}
	}

	return client
}

// NewWithKeyPair creates a new EMSG client with a key pair
func NewWithKeyPair(keyPair *keymgmt.KeyPair) *Client {
	config := DefaultConfig()
	config.KeyPair = keyPair
	return New(config)
}

// SetKeyPair sets the key pair for the client
func (c *Client) SetKeyPair(keyPair *keymgmt.KeyPair) {
	c.keyPair = keyPair
}

// GetKeyPair returns the current key pair
func (c *Client) GetKeyPair() *keymgmt.KeyPair {
	return c.keyPair
}

// ComposeMessage creates a new message builder
func (c *Client) ComposeMessage() *message.MessageBuilder {
	builder := message.NewMessageBuilder()
	if c.encryptionManager != nil {
		builder.WithEncryption(c.encryptionManager)
	}
	if c.attachmentManager != nil {
		builder.WithAttachmentManager(c.attachmentManager)
	}
	return builder
}

// SendMessage sends an EMSG message
func (c *Client) SendMessage(msg *message.Message) error {
	if c.keyPair == nil {
		return fmt.Errorf("no key pair configured")
	}

	// Start delivery tracking if enabled
	var receipt *delivery.DeliveryReceipt
	if c.deliveryTracker != nil {
		receipt = c.deliveryTracker.TrackMessage(msg)
	}

	// Call BeforeSend hook if configured
	if c.beforeSend != nil {
		if err := c.beforeSend(msg); err != nil {
			if receipt != nil {
				c.deliveryTracker.UpdateDeliveryStatus(msg.MessageID, delivery.StatusFailed, err.Error())
			}
			return fmt.Errorf("before send hook failed: %w", err)
		}
	}

	// Validate the message
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	// Sign the message
	if err := msg.Sign(c.keyPair); err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	// Get all unique domains from recipients
	domains := c.getDomainsFromMessage(msg)

	// Send to each domain
	var lastResp *http.Response
	var sendErr error
	for domain := range domains {
		resp, err := c.sendMessageToDomainWithResponse(msg, domain)
		if err != nil {
			sendErr = fmt.Errorf("failed to send message to domain %s: %w", domain, err)
			if receipt != nil {
				c.deliveryTracker.UpdateDeliveryStatus(msg.MessageID, delivery.StatusFailed, sendErr.Error())
			}
			return sendErr
		}
		lastResp = resp
	}

	// Update delivery status to sent
	if receipt != nil {
		c.deliveryTracker.UpdateDeliveryStatus(msg.MessageID, delivery.StatusSent, "")
	}

	// Call AfterSend hook if configured
	if c.afterSend != nil && lastResp != nil {
		if err := c.afterSend(msg, lastResp); err != nil {
			log.Printf("Warning: after send hook failed: %v", err)
		}
	}

	// Trigger message sent notification
	if c.notificationManager != nil {
		if err := c.notificationManager.NotifyMessageSent(msg); err != nil {
			log.Printf("Warning: failed to notify message sent: %v", err)
		}
	}

	return nil
}

// getDomainsFromMessage extracts unique domains from message recipients
func (c *Client) getDomainsFromMessage(msg *message.Message) map[string]bool {
	domains := make(map[string]bool)

	allRecipients := msg.GetRecipients()
	for _, recipient := range allRecipients {
		if domain, err := utils.ExtractDomainFromEMSGAddress(recipient); err == nil {
			domains[domain] = true
		}
	}

	return domains
}

// sendMessageToDomain sends a message to a specific domain
func (c *Client) sendMessageToDomain(msg *message.Message, domain string) error {
	_, err := c.sendMessageToDomainWithResponse(msg, domain)
	return err
}

// sendMessageToDomainWithResponse sends a message to a specific domain and returns the response
func (c *Client) sendMessageToDomainWithResponse(msg *message.Message, domain string) (*http.Response, error) {
	// Resolve the domain to get server information
	serverInfo, err := c.resolver.ResolveDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve domain %s: %w", domain, err)
	}

	// Prepare the message payload
	payload, err := msg.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	// Send HTTP request
	endpoint := fmt.Sprintf("%s/api/v1/messages", serverInfo.URL)
	return c.sendHTTPRequestWithResponse("POST", endpoint, payload)
}

// sendHTTPRequest sends an authenticated HTTP request with retry logic
func (c *Client) sendHTTPRequest(method, url string, payload []byte) error {
	var lastErr error

	for attempt := 0; attempt <= c.retryStrategy.MaxRetries; attempt++ {
		// Create HTTP request
		req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", c.userAgent)

		// Generate authentication header
		authHeader, err := auth.GenerateAuthHeader(c.keyPair, method, req.URL.Path)
		if err != nil {
			return fmt.Errorf("failed to generate auth header: %w", err)
		}

		req.Header.Set("Authorization", authHeader.ToHeaderValue())

		// Send request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			if c.shouldRetry(err, 0, attempt) {
				c.waitBeforeRetry(attempt)
				continue
			}
			return lastErr
		}
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))

			if c.shouldRetry(nil, resp.StatusCode, attempt) {
				if attempt < c.retryStrategy.MaxRetries {
					// Log retry attempt
					if resp.StatusCode == 429 {
						fmt.Printf("Rate limited (429), retrying in %v (attempt %d/%d)\n",
							c.calculateDelay(attempt), attempt+1, c.retryStrategy.MaxRetries+1)
					}
					c.waitBeforeRetry(attempt)
					continue
				}
			}
			return lastErr
		}

		return nil
	}

	return lastErr
}

// shouldRetry determines if a request should be retried
func (c *Client) shouldRetry(err error, statusCode, attempt int) bool {
	if attempt >= c.retryStrategy.MaxRetries {
		return false
	}

	// Retry on 429 (rate limit) if enabled
	if statusCode == 429 && c.retryStrategy.RetryOn429 {
		return true
	}

	// Retry on timeout errors if enabled
	if err != nil && c.retryStrategy.RetryOnTimeout {
		errStr := err.Error()
		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
			return true
		}
	}

	return false
}

// calculateDelay calculates the delay before the next retry
func (c *Client) calculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(c.retryStrategy.InitialDelay) * math.Pow(c.retryStrategy.BackoffFactor, float64(attempt)))
	if delay > c.retryStrategy.MaxDelay {
		delay = c.retryStrategy.MaxDelay
	}
	return delay
}

// waitBeforeRetry waits before retrying a request
func (c *Client) waitBeforeRetry(attempt int) {
	delay := c.calculateDelay(attempt)
	time.Sleep(delay)
}

// sendHTTPRequestWithResponse sends an authenticated HTTP request with retry logic and returns the response
func (c *Client) sendHTTPRequestWithResponse(method, url string, payload []byte) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= c.retryStrategy.MaxRetries; attempt++ {
		// Create HTTP request
		req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", c.userAgent)

		// Generate authentication header
		authHeader, err := auth.GenerateAuthHeader(c.keyPair, method, req.URL.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to generate auth header: %w", err)
		}

		req.Header.Set("Authorization", authHeader.ToHeaderValue())

		// Send request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			if c.shouldRetry(err, 0, attempt) {
				c.waitBeforeRetry(attempt)
				continue
			}
			return nil, lastErr
		}

		lastResp = resp

		// Check response status
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))

			if c.shouldRetry(nil, resp.StatusCode, attempt) {
				if attempt < c.retryStrategy.MaxRetries {
					// Log retry attempt
					if resp.StatusCode == 429 {
						log.Printf("Rate limited (429), retrying in %v (attempt %d/%d)",
							c.calculateDelay(attempt), attempt+1, c.retryStrategy.MaxRetries+1)
					}
					c.waitBeforeRetry(attempt)
					continue
				}
			}
			return nil, lastErr
		}

		return resp, nil
	}

	return lastResp, lastErr
}

// RegisterUser registers a user with an EMSG server
func (c *Client) RegisterUser(address string) error {
	if c.keyPair == nil {
		return fmt.Errorf("no key pair configured")
	}

	// Parse the address to get the domain
	addr, err := utils.ParseEMSGAddress(address)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	// Resolve the domain
	serverInfo, err := c.resolver.ResolveDomain(addr.Domain)
	if err != nil {
		return fmt.Errorf("failed to resolve domain: %w", err)
	}

	// Prepare registration payload
	registrationData := map[string]any{
		"address":    address,
		"public_key": c.keyPair.PublicKeyBase64(),
	}

	payload, err := json.Marshal(registrationData)
	if err != nil {
		return fmt.Errorf("failed to serialize registration data: %w", err)
	}

	// Send registration request
	endpoint := fmt.Sprintf("%s/api/v1/users", serverInfo.URL)
	return c.sendHTTPRequest("POST", endpoint, payload)
}

// GetMessages retrieves messages for the authenticated user
func (c *Client) GetMessages(address string) ([]*message.Message, error) {
	if c.keyPair == nil {
		return nil, fmt.Errorf("no key pair configured")
	}

	// Parse the address to get the domain
	addr, err := utils.ParseEMSGAddress(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	// Resolve the domain
	serverInfo, err := c.resolver.ResolveDomain(addr.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve domain: %w", err)
	}

	// Create HTTP request
	endpoint := fmt.Sprintf("%s/api/v1/messages", serverInfo.URL)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", c.userAgent)

	// Generate authentication header
	authHeader, err := auth.GenerateAuthHeader(c.keyPair, "GET", req.URL.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth header: %w", err)
	}

	req.Header.Set("Authorization", authHeader.ToHeaderValue())

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var messages []*message.Message
	if err := json.Unmarshal(body, &messages); err != nil {
		return nil, fmt.Errorf("failed to parse messages: %w", err)
	}

	return messages, nil
}

// ResolveDomain resolves an EMSG domain to server information
func (c *Client) ResolveDomain(domain string) (*dns.EMSGServerInfo, error) {
	return c.resolver.ResolveDomain(domain)
}

// ComposeSystemMessage creates a new system message builder
func (c *Client) ComposeSystemMessage() *message.SystemMessageBuilder {
	return message.NewSystemMessageBuilder()
}

// EnableEncryption enables encryption with the provided key pair and key store
func (c *Client) EnableEncryption(keyPair *encryption.EncryptionKeyPair, keyStore encryption.KeyStore) {
	c.encryptionManager = encryption.NewEncryptionManager(keyPair, keyStore)
}

// DisableEncryption disables encryption
func (c *Client) DisableEncryption() {
	c.encryptionManager = nil
}

// IsEncryptionEnabled returns true if encryption is enabled
func (c *Client) IsEncryptionEnabled() bool {
	return c.encryptionManager != nil
}

// RegisterPublicKey registers a public key for an address (for encryption)
func (c *Client) RegisterPublicKey(address, publicKeyBase64 string) error {
	if c.encryptionManager == nil {
		return fmt.Errorf("encryption not enabled")
	}
	return c.encryptionManager.RegisterPublicKey(address, publicKeyBase64)
}

// CanEncryptFor checks if we can encrypt for a recipient
func (c *Client) CanEncryptFor(address string) bool {
	if c.encryptionManager == nil {
		return false
	}
	return c.encryptionManager.CanEncryptFor(address)
}

// Notification methods

// RegisterNotificationHandler registers a synchronous notification handler
func (c *Client) RegisterNotificationHandler(event notifications.NotificationEvent, handler notifications.NotificationHandler) error {
	if c.notificationManager == nil {
		return fmt.Errorf("notifications not enabled")
	}
	c.notificationManager.RegisterHandler(event, handler)
	return nil
}

// RegisterAsyncNotificationHandler registers an asynchronous notification handler
func (c *Client) RegisterAsyncNotificationHandler(event notifications.NotificationEvent, handler notifications.AsyncNotificationHandler) error {
	if c.notificationManager == nil {
		return fmt.Errorf("notifications not enabled")
	}
	c.notificationManager.RegisterAsyncHandler(event, handler)
	return nil
}

// UnregisterNotificationHandlers removes all handlers for a specific event
func (c *Client) UnregisterNotificationHandlers(event notifications.NotificationEvent) error {
	if c.notificationManager == nil {
		return fmt.Errorf("notifications not enabled")
	}
	c.notificationManager.UnregisterHandlers(event)
	return nil
}

// StartMessagePolling starts polling for new messages
func (c *Client) StartMessagePolling(userAddress string) error {
	if c.messagePoller == nil {
		return fmt.Errorf("notifications not enabled")
	}
	return c.messagePoller.Start(userAddress)
}

// StopMessagePolling stops polling for new messages
func (c *Client) StopMessagePolling() {
	if c.messagePoller != nil {
		c.messagePoller.Stop()
	}
}

// IsMessagePollingRunning returns true if message polling is running
func (c *Client) IsMessagePollingRunning() bool {
	if c.messagePoller == nil {
		return false
	}
	return c.messagePoller.IsRunning()
}

// IsNotificationsEnabled returns true if notifications are enabled
func (c *Client) IsNotificationsEnabled() bool {
	return c.notificationManager != nil
}

// GetNotificationHandlerCount returns the number of handlers for an event
func (c *Client) GetNotificationHandlerCount(event notifications.NotificationEvent) int {
	if c.notificationManager == nil {
		return 0
	}
	return c.notificationManager.GetHandlerCount(event)
}

// WebSocket methods

// ConnectWebSocket establishes a WebSocket connection for real-time updates
func (c *Client) ConnectWebSocket(userAddress string) error {
	if c.webSocketClient != nil && c.webSocketClient.IsConnected() {
		return fmt.Errorf("WebSocket already connected")
	}

	// Get server URL from domain
	addr, err := utils.ParseEMSGAddress(userAddress)
	if err != nil {
		return fmt.Errorf("invalid user address: %w", err)
	}

	serverInfo, err := c.resolver.ResolveDomain(addr.Domain)
	if err != nil {
		return fmt.Errorf("failed to resolve domain: %w", err)
	}

	// Create WebSocket client
	c.webSocketClient = websocket.NewWebSocketClient(serverInfo.URL, c.keyPair, c.notificationManager)

	// Set reconnect strategy if configured
	if c.webSocketClient != nil {
		c.webSocketClient.SetReconnectStrategy(c.getWebSocketConfig())
	}

	return c.webSocketClient.Connect(userAddress)
}

// DisconnectWebSocket closes the WebSocket connection
func (c *Client) DisconnectWebSocket() error {
	if c.webSocketClient == nil {
		return fmt.Errorf("WebSocket not initialized")
	}
	return c.webSocketClient.Disconnect()
}

// IsWebSocketConnected returns true if WebSocket is connected
func (c *Client) IsWebSocketConnected() bool {
	return c.webSocketClient != nil && c.webSocketClient.IsConnected()
}

// SendWebSocketMessage sends a message via WebSocket if connected, otherwise falls back to HTTP
func (c *Client) SendWebSocketMessage(msg *message.Message) error {
	if c.IsWebSocketConnected() {
		return c.webSocketClient.SendMessage(msg)
	}
	// Fallback to HTTP
	return c.SendMessage(msg)
}

// RegisterWebSocketEventHandler registers a WebSocket event handler
func (c *Client) RegisterWebSocketEventHandler(event websocket.WebSocketEvent, handler func(data interface{})) error {
	if c.webSocketClient == nil {
		return fmt.Errorf("WebSocket not initialized")
	}
	c.webSocketClient.RegisterEventHandler(event, handler)
	return nil
}

// getWebSocketConfig returns the WebSocket configuration
func (c *Client) getWebSocketConfig() *websocket.ReconnectStrategy {
	// This would be set from the client config
	return websocket.DefaultReconnectStrategy()
}

// Delivery tracking methods

// GetDeliveryReceipt returns the delivery receipt for a message
func (c *Client) GetDeliveryReceipt(messageID string) (*delivery.DeliveryReceipt, error) {
	if c.deliveryTracker == nil {
		return nil, fmt.Errorf("delivery tracking not enabled")
	}
	return c.deliveryTracker.GetDeliveryReceipt(messageID)
}

// GetDeliveryStats returns delivery statistics
func (c *Client) GetDeliveryStats() map[delivery.DeliveryStatus]int {
	if c.deliveryTracker == nil {
		return make(map[delivery.DeliveryStatus]int)
	}
	return c.deliveryTracker.GetDeliveryStats()
}

// RegisterDeliveryCallback registers a callback for delivery status changes
func (c *Client) RegisterDeliveryCallback(messageID string, callback delivery.DeliveryCallback) error {
	if c.deliveryTracker == nil {
		return fmt.Errorf("delivery tracking not enabled")
	}
	c.deliveryTracker.RegisterCallback(messageID, callback)
	return nil
}

// RegisterGlobalDeliveryCallback registers a callback for all delivery status changes
func (c *Client) RegisterGlobalDeliveryCallback(callback delivery.DeliveryCallback) error {
	if c.deliveryTracker == nil {
		return fmt.Errorf("delivery tracking not enabled")
	}
	c.deliveryTracker.RegisterGlobalCallback(callback)
	return nil
}

// GetPendingRetries returns messages that need to be retried
func (c *Client) GetPendingRetries() []*delivery.DeliveryReceipt {
	if c.deliveryTracker == nil {
		return nil
	}
	return c.deliveryTracker.GetPendingRetries()
}

// IsDeliveryTrackingEnabled returns true if delivery tracking is enabled
func (c *Client) IsDeliveryTrackingEnabled() bool {
	return c.deliveryTracker != nil
}

// CleanupExpiredReceipts removes expired delivery receipts
func (c *Client) CleanupExpiredReceipts() int {
	if c.deliveryTracker == nil {
		return 0
	}
	return c.deliveryTracker.CleanupExpiredReceipts()
}

// Attachment methods

// CreateAttachmentFromFile creates an attachment from a file
func (c *Client) CreateAttachmentFromFile(filePath string) (*attachments.Attachment, error) {
	if c.attachmentManager == nil {
		return nil, fmt.Errorf("attachment manager not initialized")
	}
	return c.attachmentManager.CreateAttachmentFromFile(filePath)
}

// CreateAttachmentFromData creates an attachment from raw data
func (c *Client) CreateAttachmentFromData(name string, data []byte, mimeType string) (*attachments.Attachment, error) {
	if c.attachmentManager == nil {
		return nil, fmt.Errorf("attachment manager not initialized")
	}
	return c.attachmentManager.CreateAttachmentFromData(name, data, mimeType)
}

// SaveAttachment saves an attachment to storage
func (c *Client) SaveAttachment(attachment *attachments.Attachment) error {
	if c.attachmentManager == nil {
		return fmt.Errorf("attachment manager not initialized")
	}
	return c.attachmentManager.SaveAttachment(attachment)
}

// LoadAttachment loads an attachment from storage
func (c *Client) LoadAttachment(attachmentID string) (*attachments.Attachment, error) {
	if c.attachmentManager == nil {
		return nil, fmt.Errorf("attachment manager not initialized")
	}
	return c.attachmentManager.LoadAttachment(attachmentID)
}

// ValidateAttachment validates an attachment's integrity
func (c *Client) ValidateAttachment(attachment *attachments.Attachment) error {
	if c.attachmentManager == nil {
		return fmt.Errorf("attachment manager not initialized")
	}
	return c.attachmentManager.ValidateAttachment(attachment)
}

// GetAttachmentData returns the complete data of an attachment
func (c *Client) GetAttachmentData(attachment *attachments.Attachment) ([]byte, error) {
	if c.attachmentManager == nil {
		return nil, fmt.Errorf("attachment manager not initialized")
	}
	return c.attachmentManager.GetAttachmentData(attachment)
}

// IsAttachmentManagerEnabled returns true if attachment manager is enabled
func (c *Client) IsAttachmentManagerEnabled() bool {
	return c.attachmentManager != nil
}
