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

	"github.com/emsg-protocol/emsg-client-sdk/auth"
	"github.com/emsg-protocol/emsg-client-sdk/dns"
	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
	"github.com/emsg-protocol/emsg-client-sdk/message"
	"github.com/emsg-protocol/emsg-client-sdk/utils"
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
	keyPair       *keymgmt.KeyPair
	resolver      *dns.CachedResolver
	httpClient    *http.Client
	userAgent     string
	retryStrategy *RetryStrategy
	beforeSend    func(*message.Message) error
	afterSend     func(*message.Message, *http.Response) error
}

// Config holds configuration for the EMSG client
type Config struct {
	KeyPair       *keymgmt.KeyPair
	Timeout       time.Duration
	UserAgent     string
	DNSConfig     *dns.ResolverConfig
	DNSTTL        time.Duration
	RetryStrategy *RetryStrategy
	BeforeSend    func(*message.Message) error
	AfterSend     func(*message.Message, *http.Response) error
}

// DefaultConfig returns a default client configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:       30 * time.Second,
		UserAgent:     "emsg-client-sdk/1.0",
		DNSConfig:     dns.DefaultResolverConfig(),
		DNSTTL:        5 * time.Minute,
		RetryStrategy: DefaultRetryStrategy(),
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

	return &Client{
		keyPair:       config.KeyPair,
		resolver:      resolver,
		httpClient:    httpClient,
		userAgent:     config.UserAgent,
		retryStrategy: retryStrategy,
		beforeSend:    config.BeforeSend,
		afterSend:     config.AfterSend,
	}
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
	return message.NewMessageBuilder()
}

// SendMessage sends an EMSG message
func (c *Client) SendMessage(msg *message.Message) error {
	if c.keyPair == nil {
		return fmt.Errorf("no key pair configured")
	}

	// Call BeforeSend hook if configured
	if c.beforeSend != nil {
		if err := c.beforeSend(msg); err != nil {
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
	for domain := range domains {
		resp, err := c.sendMessageToDomainWithResponse(msg, domain)
		if err != nil {
			return fmt.Errorf("failed to send message to domain %s: %w", domain, err)
		}
		lastResp = resp
	}

	// Call AfterSend hook if configured
	if c.afterSend != nil && lastResp != nil {
		if err := c.afterSend(msg, lastResp); err != nil {
			log.Printf("Warning: after send hook failed: %v", err)
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
