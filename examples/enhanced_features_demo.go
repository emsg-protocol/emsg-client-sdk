package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/client"
	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
	"github.com/emsg-protocol/emsg-client-sdk/message"
)

func main() {
	fmt.Println("🚀 EMSG Client SDK - Enhanced Features Demo")
	fmt.Println("==========================================")

	// Generate key pair for demo
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	fmt.Printf("✅ Generated key pair. Public key: %s\n", keyPair.PublicKeyBase64())

	// Demo 1: System Messages
	fmt.Println("\n📋 Demo 1: System Messages")
	fmt.Println("---------------------------")

	demoSystemMessages(keyPair)

	// Demo 2: Retry Logic and Hooks
	fmt.Println("\n🔄 Demo 2: Retry Logic and Hooks")
	fmt.Println("--------------------------------")

	demoRetryAndHooks(keyPair)

	// Demo 3: Enhanced Client Configuration
	fmt.Println("\n⚙️  Demo 3: Enhanced Client Configuration")
	fmt.Println("----------------------------------------")

	demoEnhancedConfig(keyPair)

	fmt.Println("\n🎉 Demo completed successfully!")
}

func demoSystemMessages(keyPair *keymgmt.KeyPair) {
	// Create client
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	emsgClient := client.New(config)

	// Demo system message types
	systemTypes := []struct {
		name    string
		msgFunc func() (*message.Message, error)
	}{
		{
			name: "User Joined",
			msgFunc: func() (*message.Message, error) {
				return message.NewUserJoinedMessage(
					"system#example.com",
					[]string{"group#example.com"},
					"alice#example.com",
					"team-alpha",
				)
			},
		},
		{
			name: "User Left",
			msgFunc: func() (*message.Message, error) {
				return message.NewUserLeftMessage(
					"system#example.com",
					[]string{"group#example.com"},
					"bob#example.com",
					"team-alpha",
				)
			},
		},
		{
			name: "User Removed",
			msgFunc: func() (*message.Message, error) {
				return message.NewUserRemovedMessage(
					"system#example.com",
					[]string{"group#example.com"},
					"admin#example.com",
					"charlie#example.com",
					"team-alpha",
				)
			},
		},
		{
			name: "Admin Changed",
			msgFunc: func() (*message.Message, error) {
				return message.NewAdminChangedMessage(
					"system#example.com",
					[]string{"group#example.com"},
					"owner#example.com",
					"alice#example.com",
					"team-alpha",
				)
			},
		},
		{
			name: "Group Created",
			msgFunc: func() (*message.Message, error) {
				return message.NewGroupCreatedMessage(
					"system#example.com",
					[]string{"all#example.com"},
					"admin#example.com",
					"team-beta",
				)
			},
		},
	}

	for _, st := range systemTypes {
		fmt.Printf("  📝 Creating %s message...\n", st.name)

		msg, err := st.msgFunc()
		if err != nil {
			fmt.Printf("    ❌ Failed: %v\n", err)
			continue
		}

		// Validate the message
		if err := msg.Validate(); err != nil {
			fmt.Printf("    ❌ Validation failed: %v\n", err)
			continue
		}

		// Sign the message
		if err := msg.Sign(keyPair); err != nil {
			fmt.Printf("    ❌ Signing failed: %v\n", err)
			continue
		}

		// Parse system message data
		systemData, err := msg.GetSystemMessage()
		if err != nil {
			fmt.Printf("    ❌ Failed to parse system data: %v\n", err)
			continue
		}

		fmt.Printf("    ✅ Success! Type: %s, Actor: %s\n", systemData.Type, systemData.Actor)
	}

	// Demo custom system message
	fmt.Println("  📝 Creating custom system message...")

	customMsg, err := emsgClient.ComposeSystemMessage().
		Type("system:custom_event").
		Actor("user#example.com").
		Target("resource#example.com").
		GroupID("project-gamma").
		Metadata("action", "file_uploaded").
		Metadata("filename", "document.pdf").
		Metadata("size", 1024*1024).
		Metadata("timestamp", time.Now().Unix()).
		Build("system#example.com", []string{"team#example.com"})

	if err != nil {
		fmt.Printf("    ❌ Failed: %v\n", err)
		return
	}

	if err := customMsg.Validate(); err != nil {
		fmt.Printf("    ❌ Validation failed: %v\n", err)
		return
	}

	fmt.Printf("    ✅ Custom system message created successfully!\n")
}

func demoRetryAndHooks(keyPair *keymgmt.KeyPair) {
	var beforeSendCount int
	var afterSendCount int

	// Create client with retry strategy and hooks
	config := client.DefaultConfig()
	config.KeyPair = keyPair

	// Configure aggressive retry strategy
	config.RetryStrategy = &client.RetryStrategy{
		MaxRetries:     3,
		InitialDelay:   500 * time.Millisecond,
		MaxDelay:       5 * time.Second,
		BackoffFactor:  2.0,
		RetryOn429:     true,
		RetryOnTimeout: true,
	}

	// Configure hooks
	config.BeforeSend = func(msg *message.Message) error {
		beforeSendCount++
		fmt.Printf("    🔍 BeforeSend hook called (count: %d)\n", beforeSendCount)

		// Add timestamp to subject if not present
		if msg.Subject == "" {
			msg.Subject = fmt.Sprintf("Auto-generated at %s", time.Now().Format("15:04:05"))
		}

		// Log message details
		fmt.Printf("      📧 Message: %s -> %v\n", msg.From, msg.To)

		return nil
	}

	config.AfterSend = func(msg *message.Message, resp *http.Response) error {
		afterSendCount++
		fmt.Printf("    📬 AfterSend hook called (count: %d)\n", afterSendCount)
		fmt.Printf("      📊 Response status: %d %s\n", resp.StatusCode, resp.Status)

		// Log successful sends
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			fmt.Printf("      ✅ Message sent successfully!\n")
		}

		return nil
	}

	emsgClient := client.New(config)

	fmt.Printf("  ⚙️  Retry strategy configured: %d max retries, %v initial delay\n",
		config.RetryStrategy.MaxRetries, config.RetryStrategy.InitialDelay)

	// Create test message
	msg, err := emsgClient.ComposeMessage().
		From("sender#example.com").
		To("recipient#example.com").
		Body("This message demonstrates retry logic and hooks").
		Build()

	if err != nil {
		fmt.Printf("    ❌ Failed to build message: %v\n", err)
		return
	}

	fmt.Printf("  📝 Created test message (subject will be auto-generated by hook)\n")
	fmt.Printf("  🔄 Attempting to send message (will fail but demonstrate retry logic)...\n")

	// This will fail since we don't have a real server, but it demonstrates the hooks
	err = emsgClient.SendMessage(msg)
	if err != nil {
		fmt.Printf("    ⚠️  Send failed as expected: %v\n", err)
		fmt.Printf("    📊 BeforeSend called %d times, AfterSend called %d times\n",
			beforeSendCount, afterSendCount)
	}
}

func demoEnhancedConfig(keyPair *keymgmt.KeyPair) {
	// Demo different configuration options
	configs := []struct {
		name   string
		config *client.Config
	}{
		{
			name:   "Default Configuration",
			config: client.DefaultConfig(),
		},
		{
			name: "High Performance Configuration",
			config: &client.Config{
				KeyPair:   keyPair,
				Timeout:   10 * time.Second,
				UserAgent: "HighPerf-EMSG-Client/1.0",
				RetryStrategy: &client.RetryStrategy{
					MaxRetries:     1,
					InitialDelay:   100 * time.Millisecond,
					MaxDelay:       1 * time.Second,
					BackoffFactor:  1.5,
					RetryOn429:     false,
					RetryOnTimeout: false,
				},
			},
		},
		{
			name: "Resilient Configuration",
			config: &client.Config{
				KeyPair:   keyPair,
				Timeout:   60 * time.Second,
				UserAgent: "Resilient-EMSG-Client/1.0",
				RetryStrategy: &client.RetryStrategy{
					MaxRetries:     5,
					InitialDelay:   2 * time.Second,
					MaxDelay:       30 * time.Second,
					BackoffFactor:  2.5,
					RetryOn429:     true,
					RetryOnTimeout: true,
				},
			},
		},
	}

	for _, cfg := range configs {
		fmt.Printf("  ⚙️  %s:\n", cfg.name)

		if cfg.config.KeyPair == nil {
			cfg.config.KeyPair = keyPair
		}

		emsgClient := client.New(cfg.config)

		fmt.Printf("      🕐 Timeout: %v\n", cfg.config.Timeout)
		fmt.Printf("      🏷️  User Agent: %s\n", cfg.config.UserAgent)

		if cfg.config.RetryStrategy != nil {
			fmt.Printf("      🔄 Max Retries: %d\n", cfg.config.RetryStrategy.MaxRetries)
			fmt.Printf("      ⏱️  Initial Delay: %v\n", cfg.config.RetryStrategy.InitialDelay)
			fmt.Printf("      📈 Backoff Factor: %.1f\n", cfg.config.RetryStrategy.BackoffFactor)
			fmt.Printf("      🚫 Retry on 429: %t\n", cfg.config.RetryStrategy.RetryOn429)
		}

		// Test message composition
		_, err := emsgClient.ComposeMessage().
			From("test#example.com").
			To("target#example.com").
			Subject(fmt.Sprintf("Test from %s", cfg.name)).
			Body("Configuration test message").
			Build()

		if err != nil {
			fmt.Printf("      ❌ Failed to create message: %v\n", err)
		} else {
			fmt.Printf("      ✅ Message created successfully\n")
		}

		// Test system message composition
		_, err = emsgClient.ComposeSystemMessage().
			Type(message.SystemJoined).
			Actor("user#example.com").
			GroupID("test-group").
			Build("system#example.com", []string{"group#example.com"})

		if err != nil {
			fmt.Printf("      ❌ Failed to create system message: %v\n", err)
		} else {
			fmt.Printf("      ✅ System message created successfully\n")
		}

		fmt.Println()
	}
}
