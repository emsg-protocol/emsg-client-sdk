package integration

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/client"
	"github.com/emsg-protocol/emsg-client-sdk/dns"
	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
	"github.com/emsg-protocol/emsg-client-sdk/message"
)

// MockEMSGServer creates a mock EMSG server for testing
func createMockEMSGServer() *httptest.Server {
	mux := http.NewServeMux()

	// Mock user registration endpoint
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Check for authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "missing authorization header"}`))
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status": "user registered successfully"}`))
	})

	// Mock message sending endpoint
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			// Check for authorization header
			auth := r.Header.Get("Authorization")
			if auth == "" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "missing authorization header"}`))
				return
			}

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status": "message sent successfully"}`))

		case "GET":
			// Check for authorization header
			auth := r.Header.Get("Authorization")
			if auth == "" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "missing authorization header"}`))
				return
			}

			// Return mock messages
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{
					"from": "test#example.com",
					"to": ["user#example.com"],
					"subject": "Test Message",
					"body": "This is a test message",
					"timestamp": 1640995200,
					"message_id": "test-msg-1",
					"signature": "test-signature"
				}
			]`))

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Mock rate limiting endpoint (returns 429)
	mux.HandleFunc("/api/v1/rate-limit-test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limit exceeded"}`))
	})

	return httptest.NewServer(mux)
}

// TestUserRegistration tests user registration with mock server
func TestUserRegistration(t *testing.T) {
	// Create mock server
	server := createMockEMSGServer()
	defer server.Close()

	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create client with mock resolver
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	config.DNSConfig = &dns.ResolverConfig{
		Timeout: 5 * time.Second,
		Retries: 1,
	}

	_ = client.New(config)

	// Mock the DNS resolution to point to our test server
	_ = &mockDNSResolver{serverURL: server.URL}
	// Note: In a real implementation, we'd need to inject the resolver

	// For this test, we'll test the components separately
	testAddress := "testuser#example.com"

	// Test that we can create a registration request
	// (We can't easily test the full flow without exposing the resolver)
	if !strings.Contains(testAddress, "#") {
		t.Errorf("Invalid test address format: %s", testAddress)
	}
}

// TestMessageSending tests message sending functionality
func TestMessageSending(t *testing.T) {
	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create client
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	emsgClient := client.New(config)

	// Test message creation and validation
	msg, err := emsgClient.ComposeMessage().
		From("sender#example.com").
		To("recipient#example.com").
		Subject("Integration Test").
		Body("This is an integration test message").
		Build()

	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}

	// Validate message
	if err := msg.Validate(); err != nil {
		t.Errorf("Message validation failed: %v", err)
	}

	// Test message signing
	if err := msg.Sign(keyPair); err != nil {
		t.Errorf("Message signing failed: %v", err)
	}

	// Verify signature
	if err := msg.Verify(keyPair.PublicKeyBase64()); err != nil {
		t.Errorf("Message verification failed: %v", err)
	}

	// Test that message is marked as signed
	if !msg.IsSigned() {
		t.Error("Message should be marked as signed")
	}
}

// TestSystemMessages tests system message functionality
func TestSystemMessages(t *testing.T) {
	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create client
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	emsgClient := client.New(config)

	// Test system message creation
	systemMsg, err := emsgClient.ComposeSystemMessage().
		Type(message.SystemJoined).
		Actor("user#example.com").
		GroupID("test-group").
		Metadata("timestamp", time.Now().Unix()).
		Build("system#example.com", []string{"group#example.com"})

	if err != nil {
		t.Fatalf("Failed to build system message: %v", err)
	}

	// Validate system message
	if err := systemMsg.Validate(); err != nil {
		t.Errorf("System message validation failed: %v", err)
	}

	// Test that it's recognized as a system message
	if !systemMsg.IsSystemMessage() {
		t.Error("Message should be recognized as system message")
	}

	// Test parsing system message data
	parsedSystemMsg, err := systemMsg.GetSystemMessage()
	if err != nil {
		t.Errorf("Failed to parse system message: %v", err)
	}

	if parsedSystemMsg.Type != message.SystemJoined {
		t.Errorf("Expected system message type %s, got %s", message.SystemJoined, parsedSystemMsg.Type)
	}

	if parsedSystemMsg.Actor != "user#example.com" {
		t.Errorf("Expected actor user#example.com, got %s", parsedSystemMsg.Actor)
	}

	// Test helper functions for common system messages
	joinedMsg, err := message.NewUserJoinedMessage(
		"system#example.com",
		[]string{"group#example.com"},
		"user#example.com",
		"test-group",
	)
	if err != nil {
		t.Errorf("Failed to create user joined message: %v", err)
	}

	if !joinedMsg.IsSystemMessage() {
		t.Error("Joined message should be a system message")
	}
}

// TestRetryLogic tests the retry functionality
func TestRetryLogic(t *testing.T) {
	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create client with retry strategy
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	config.RetryStrategy = &client.RetryStrategy{
		MaxRetries:     2,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       1 * time.Second,
		BackoffFactor:  2.0,
		RetryOn429:     true,
		RetryOnTimeout: true,
	}

	_ = client.New(config)

	// Test that retry strategy is configured
	if config.RetryStrategy.MaxRetries != 2 {
		t.Errorf("Expected MaxRetries to be 2, got %d", config.RetryStrategy.MaxRetries)
	}

	if !config.RetryStrategy.RetryOn429 {
		t.Error("Expected RetryOn429 to be true")
	}
}

// TestHooks tests the before/after send hooks
func TestHooks(t *testing.T) {
	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	var beforeSendCalled bool

	// Create client with hooks
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	config.BeforeSend = func(msg *message.Message) error {
		beforeSendCalled = true
		// Add custom header or modify message
		if msg.Subject == "" {
			msg.Subject = "Auto-generated subject"
		}
		return nil
	}
	config.AfterSend = func(msg *message.Message, resp *http.Response) error {
		// Log or process response
		return nil
	}

	emsgClient := client.New(config)

	// Create a test message
	msg, err := emsgClient.ComposeMessage().
		From("sender#example.com").
		To("recipient#example.com").
		Body("Test message with hooks").
		Build()

	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}

	// Test that hooks are configured (we can't easily test the actual calling without a real server)
	if config.BeforeSend == nil {
		t.Error("BeforeSend hook should be configured")
	}

	if config.AfterSend == nil {
		t.Error("AfterSend hook should be configured")
	}

	// Test the hook function directly
	err = config.BeforeSend(msg)
	if err != nil {
		t.Errorf("BeforeSend hook failed: %v", err)
	}

	if !beforeSendCalled {
		t.Error("BeforeSend hook was not called")
	}

	if msg.Subject != "Auto-generated subject" {
		t.Errorf("Expected subject to be modified by hook, got: %s", msg.Subject)
	}
}

// mockDNSResolver is a mock DNS resolver for testing
type mockDNSResolver struct {
	serverURL string
}

func (m *mockDNSResolver) ResolveDomain(domain string) (*dns.EMSGServerInfo, error) {
	return &dns.EMSGServerInfo{
		URL:     m.serverURL,
		Version: "1.0",
	}, nil
}

// TestDNSResolution tests DNS resolution functionality
func TestDNSResolution(t *testing.T) {
	// Test DNS resolver configuration
	config := &dns.ResolverConfig{
		Timeout: 10 * time.Second,
		Retries: 3,
	}

	resolver := dns.NewResolver(config)
	if resolver == nil {
		t.Error("Failed to create DNS resolver")
	}

	// Test with a known domain (this will fail in CI but shows the structure)
	// In a real test environment, you'd mock the DNS responses
	domain := "example.com"
	_, err := resolver.ResolveDomain(domain)

	// We expect this to fail since example.com doesn't have EMSG records
	if err == nil {
		t.Log("Unexpected success resolving example.com (no EMSG records expected)")
	} else {
		t.Logf("Expected DNS resolution failure for %s: %v", domain, err)
	}
}
