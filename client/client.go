package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/auth"
	"github.com/emsg-protocol/emsg-client-sdk/dns"
	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
	"github.com/emsg-protocol/emsg-client-sdk/message"
	"github.com/emsg-protocol/emsg-client-sdk/utils"
)

// Client represents the EMSG client SDK
type Client struct {
	keyPair  *keymgmt.KeyPair
	resolver *dns.CachedResolver
	httpClient *http.Client
	userAgent string
}

// Config holds configuration for the EMSG client
type Config struct {
	KeyPair    *keymgmt.KeyPair
	Timeout    time.Duration
	UserAgent  string
	DNSConfig  *dns.ResolverConfig
	DNSTTL     time.Duration
}

// DefaultConfig returns a default client configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:   30 * time.Second,
		UserAgent: "emsg-client-sdk/1.0",
		DNSConfig: dns.DefaultResolverConfig(),
		DNSTTL:    5 * time.Minute,
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

	return &Client{
		keyPair:    config.KeyPair,
		resolver:   resolver,
		httpClient: httpClient,
		userAgent:  config.UserAgent,
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
	for domain := range domains {
		if err := c.sendMessageToDomain(msg, domain); err != nil {
			return fmt.Errorf("failed to send message to domain %s: %w", domain, err)
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
	// Resolve the domain to get server information
	serverInfo, err := c.resolver.ResolveDomain(domain)
	if err != nil {
		return fmt.Errorf("failed to resolve domain %s: %w", domain, err)
	}

	// Prepare the message payload
	payload, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Send HTTP request
	endpoint := fmt.Sprintf("%s/api/v1/messages", serverInfo.URL)
	return c.sendHTTPRequest("POST", endpoint, payload)
}

// sendHTTPRequest sends an authenticated HTTP request
func (c *Client) sendHTTPRequest(method, url string, payload []byte) error {
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
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
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
	registrationData := map[string]interface{}{
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
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := ioutil.ReadAll(resp.Body)
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
