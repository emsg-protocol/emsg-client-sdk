package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/client"
	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
	"github.com/emsg-protocol/emsg-client-sdk/message"
)

// TestWithDockerEMSGDaemon tests against a real EMSG daemon running in Docker
// This test is skipped unless INTEGRATION_TEST=docker environment variable is set
func TestWithDockerEMSGDaemon(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "docker" {
		t.Skip("Skipping Docker integration test. Set INTEGRATION_TEST=docker to run.")
	}

	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create client
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	config.Timeout = 30 * time.Second

	emsgClient := client.New(config)

	// Test user registration
	testAddress := "testuser#localhost"
	t.Logf("Testing user registration for: %s", testAddress)

	err = emsgClient.RegisterUser(testAddress)
	if err != nil {
		t.Logf("User registration failed (expected if server not running): %v", err)
	} else {
		t.Logf("User registration successful")
	}

	// Test message sending
	msg, err := emsgClient.ComposeMessage().
		From(testAddress).
		To("recipient#localhost").
		Subject("Docker Integration Test").
		Body("This is a test message from Docker integration test").
		Build()

	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}

	err = emsgClient.SendMessage(msg)
	if err != nil {
		t.Logf("Message sending failed (expected if server not running): %v", err)
	} else {
		t.Logf("Message sent successfully")
	}
}

// TestWithRealEMSGServer tests against a real EMSG server
// This test is skipped unless INTEGRATION_TEST=real environment variable is set
func TestWithRealEMSGServer(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "real" {
		t.Skip("Skipping real server integration test. Set INTEGRATION_TEST=real to run.")
	}

	// Use sandipwalke.com for testing (as used in the original tests)
	testDomain := "sandipwalke.com"

	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create client with longer timeout for real network requests
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	config.Timeout = 60 * time.Second

	emsgClient := client.New(config)

	// Test DNS resolution first
	t.Logf("Testing DNS resolution for domain: %s", testDomain)
	serverInfo, err := emsgClient.ResolveDomain(testDomain)
	if err != nil {
		t.Fatalf("DNS resolution failed: %v", err)
	}
	t.Logf("DNS resolution successful: %s", serverInfo.URL)

	// Test user registration
	testAddress := fmt.Sprintf("testuser_%d#%s", time.Now().Unix(), testDomain)
	t.Logf("Testing user registration for: %s", testAddress)

	err = emsgClient.RegisterUser(testAddress)
	if err != nil {
		t.Logf("User registration result: %v", err)
	} else {
		t.Logf("User registration successful")
	}

	// Test message sending
	recipientAddress := fmt.Sprintf("recipient_%d#%s", time.Now().Unix(), testDomain)
	msg, err := emsgClient.ComposeMessage().
		From(testAddress).
		To(recipientAddress).
		Subject("Real Server Integration Test").
		Body("This is a test message from real server integration test").
		Build()

	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}

	t.Logf("Testing message sending from %s to %s", testAddress, recipientAddress)
	err = emsgClient.SendMessage(msg)
	if err != nil {
		t.Logf("Message sending result: %v", err)
	} else {
		t.Logf("Message sent successfully")
	}

	// Test system message
	systemMsg, err := emsgClient.ComposeSystemMessage().
		Type("system:test").
		Actor(testAddress).
		GroupID("test-group").
		Metadata("test_run", time.Now().Unix()).
		Build(testAddress, []string{recipientAddress})

	if err != nil {
		t.Fatalf("Failed to build system message: %v", err)
	}

	t.Logf("Testing system message sending")
	err = emsgClient.SendMessage(systemMsg)
	if err != nil {
		t.Logf("System message sending result: %v", err)
	} else {
		t.Logf("System message sent successfully")
	}
}

// TestRetryWithRealServer tests retry logic with a real server
func TestRetryWithRealServer(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "retry" {
		t.Skip("Skipping retry integration test. Set INTEGRATION_TEST=retry to run.")
	}

	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create client with aggressive retry strategy
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	config.Timeout = 10 * time.Second
	config.RetryStrategy = &client.RetryStrategy{
		MaxRetries:     5,
		InitialDelay:   500 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		BackoffFactor:  2.0,
		RetryOn429:     true,
		RetryOnTimeout: true,
	}

	// Add logging hooks to observe retry behavior
	var attempts int
	config.BeforeSend = func(msg *message.Message) error {
		attempts++
		t.Logf("Attempt %d: Sending message from %s", attempts, msg.From)
		return nil
	}

	config.AfterSend = func(msg *message.Message, resp *http.Response) error {
		t.Logf("Message sent successfully with status: %d", resp.StatusCode)
		return nil
	}

	emsgClient := client.New(config)

	// Test with a domain that might have rate limiting
	testDomain := "sandipwalke.com"
	testAddress := fmt.Sprintf("retrytest_%d#%s", time.Now().Unix(), testDomain)

	msg, err := emsgClient.ComposeMessage().
		From(testAddress).
		To(fmt.Sprintf("target_%d#%s", time.Now().Unix(), testDomain)).
		Subject("Retry Test").
		Body("Testing retry logic").
		Build()

	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}

	start := time.Now()
	err = emsgClient.SendMessage(msg)
	duration := time.Since(start)

	t.Logf("Send operation took %v with %d attempts", duration, attempts)

	if err != nil {
		t.Logf("Send failed after retries: %v", err)
	} else {
		t.Logf("Send successful")
	}
}

// TestConcurrentRequests tests concurrent message sending
func TestConcurrentRequests(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "concurrent" {
		t.Skip("Skipping concurrent integration test. Set INTEGRATION_TEST=concurrent to run.")
	}

	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create client
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	config.Timeout = 30 * time.Second

	emsgClient := client.New(config)

	// Test concurrent message sending
	const numMessages = 5
	testDomain := "sandipwalke.com"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	results := make(chan error, numMessages)

	for i := 0; i < numMessages; i++ {
		go func(msgNum int) {
			testAddress := fmt.Sprintf("concurrent_%d_%d#%s", time.Now().Unix(), msgNum, testDomain)

			msg, err := emsgClient.ComposeMessage().
				From(testAddress).
				To(fmt.Sprintf("target_%d#%s", msgNum, testDomain)).
				Subject(fmt.Sprintf("Concurrent Test %d", msgNum)).
				Body(fmt.Sprintf("This is concurrent test message %d", msgNum)).
				Build()

			if err != nil {
				results <- fmt.Errorf("failed to build message %d: %w", msgNum, err)
				return
			}

			err = emsgClient.SendMessage(msg)
			results <- err
		}(i)
	}

	// Collect results
	var successCount, failureCount int
	for i := 0; i < numMessages; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Logf("Message %d failed: %v", i, err)
				failureCount++
			} else {
				t.Logf("Message %d sent successfully", i)
				successCount++
			}
		case <-ctx.Done():
			t.Fatalf("Test timed out waiting for results")
		}
	}

	t.Logf("Concurrent test results: %d successful, %d failed", successCount, failureCount)
}

// TestPerformance tests basic performance characteristics
func TestPerformance(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "performance" {
		t.Skip("Skipping performance integration test. Set INTEGRATION_TEST=performance to run.")
	}

	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create client
	config := client.DefaultConfig()
	config.KeyPair = keyPair

	emsgClient := client.New(config)

	// Test message creation performance
	const numMessages = 1000
	start := time.Now()

	for i := 0; i < numMessages; i++ {
		msg, err := emsgClient.ComposeMessage().
			From("perf#example.com").
			To("target#example.com").
			Subject(fmt.Sprintf("Performance Test %d", i)).
			Body(fmt.Sprintf("This is performance test message %d", i)).
			Build()

		if err != nil {
			t.Fatalf("Failed to build message %d: %v", i, err)
		}

		// Sign the message
		if err := msg.Sign(keyPair); err != nil {
			t.Fatalf("Failed to sign message %d: %v", i, err)
		}

		// Verify the signature
		if err := msg.Verify(keyPair.PublicKeyBase64()); err != nil {
			t.Fatalf("Failed to verify message %d: %v", i, err)
		}
	}

	duration := time.Since(start)
	messagesPerSecond := float64(numMessages) / duration.Seconds()

	t.Logf("Created, signed, and verified %d messages in %v", numMessages, duration)
	t.Logf("Performance: %.2f messages/second", messagesPerSecond)

	if messagesPerSecond < 100 {
		t.Logf("Warning: Performance is below 100 messages/second (%.2f)", messagesPerSecond)
	}
}
