# EMSG Client SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/emsg-protocol/emsg-client-sdk.svg)](https://pkg.go.dev/github.com/emsg-protocol/emsg-client-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/emsg-protocol/emsg-client-sdk)](https://goreportcard.com/report/github.com/emsg-protocol/emsg-client-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A cross-platform, developer-friendly client SDK for the EMSG (Electronic Message) protocol in Go, enabling secure communication, message routing, and cryptographic authentication.

## Features

- 🔐 **Ed25519 Cryptography**: Generate and manage Ed25519 key pairs for secure signing
- 🌐 **DNS-based Routing**: Resolve EMSG addresses via DNS TXT records
- 📨 **Message Management**: Compose, sign, and send EMSG messages
- 🔒 **Authentication**: Generate signed authentication headers for API requests
- ⚙️ **System Messages**: Built-in support for system events (joined, left, removed, admin_changed, group_created)
- 🔄 **Retry Logic**: Configurable retry strategies with exponential backoff for rate limiting
- 🪝 **Developer Hooks**: Before/after send callbacks for custom logging and processing
- 🏗️ **Clean API**: Idiomatic Go package structure with comprehensive documentation
- 🧪 **Integration Testing**: Mock and real server testing capabilities
- ✅ **Well Tested**: 50+ unit tests with comprehensive coverage

## Installation

```bash
go get github.com/emsg-protocol/emsg-client-sdk
```

## Quick Start

### 1. Generate a Key Pair

```go
package main

import (
    "fmt"
    "log"

    "github.com/emsg-protocol/emsg-client-sdk/keymgmt"
)

func main() {
    // Generate a new Ed25519 key pair
    keyPair, err := keymgmt.GenerateKeyPair()
    if err != nil {
        log.Fatal(err)
    }

    // Save private key to file
    err = keyPair.SavePrivateKeyToFile("my-key.txt")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Public key: %s\n", keyPair.PublicKeyBase64())
}
```

### 2. Send a Message

```go
package main

import (
    "log"

    "github.com/emsg-protocol/emsg-client-sdk/client"
    "github.com/emsg-protocol/emsg-client-sdk/keymgmt"
)

func main() {
    // Load your private key
    keyPair, err := keymgmt.LoadPrivateKeyFromFile("my-key.txt")
    if err != nil {
        log.Fatal(err)
    }

    // Create EMSG client
    emsgClient := client.NewWithKeyPair(keyPair)

    // Compose message
    msg, err := emsgClient.ComposeMessage().
        From("alice#example.com").
        To("bob#test.org").
        Subject("Hello").
        Body("Hello, Bob! This is a secure EMSG message.").
        Build()
    if err != nil {
        log.Fatal(err)
    }

    // Send message
    err = emsgClient.SendMessage(msg)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Message sent successfully!")
}
```

### 3. Register a User

```go
package main

import (
    "log"

    "github.com/emsg-protocol/emsg-client-sdk/client"
    "github.com/emsg-protocol/emsg-client-sdk/keymgmt"
)

func main() {
    // Load your private key
    keyPair, err := keymgmt.LoadPrivateKeyFromFile("my-key.txt")
    if err != nil {
        log.Fatal(err)
    }

    // Create EMSG client
    emsgClient := client.NewWithKeyPair(keyPair)

    // Register user
    err = emsgClient.RegisterUser("alice#example.com")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("User registered successfully!")
}
```

## API Reference

### Key Management (`keymgmt`)

The `keymgmt` package provides Ed25519 key pair generation and management.

```go
// Generate a new key pair
keyPair, err := keymgmt.GenerateKeyPair()

// Save private key to file
err = keyPair.SavePrivateKeyToFile("key.txt")

// Load private key from file
keyPair, err = keymgmt.LoadPrivateKeyFromFile("key.txt")

// Load private key from hex string
keyPair, err = keymgmt.LoadPrivateKeyFromHex("deadbeef...")

// Get public key as base64
pubKeyB64 := keyPair.PublicKeyBase64()

// Sign a message
signature := keyPair.Sign([]byte("message"))

// Verify a signature
valid := keyPair.Verify([]byte("message"), signature)
```

### Authentication (`auth`)

The `auth` package handles authentication header generation and verification.

```go
// Generate authentication header
authHeader, err := auth.GenerateAuthHeader(keyPair, "GET", "/api/v1/messages")

// Convert to HTTP header value
headerValue := authHeader.ToHeaderValue()
// Result: "EMSG pubkey=...,signature=...,timestamp=...,nonce=..."

// Parse authentication header
parsedHeader, err := auth.ParseAuthHeader(headerValue)

// Verify authentication header
err = auth.VerifyAuthHeader(parsedHeader, "GET", "/api/v1/messages")
```

### Address Parsing (`utils`)

The `utils` package provides EMSG address parsing and validation.

```go
// Parse EMSG address
addr, err := utils.ParseEMSGAddress("alice#example.com")
fmt.Printf("User: %s, Domain: %s\n", addr.User, addr.Domain)

// Get DNS name for EMSG lookup
dnsName := addr.GetEMSGDNSName() // "_emsg.example.com"

// Validate address
valid := utils.IsValidEMSGAddress("alice#example.com")

// Normalize address (trim whitespace, lowercase domain)
normalized := utils.NormalizeEMSGAddress("  Alice#EXAMPLE.COM  ")
// Result: "Alice#example.com"

// Extract parts
domain, err := utils.ExtractDomainFromEMSGAddress("alice#example.com")
user, err := utils.ExtractUserFromEMSGAddress("alice#example.com")
```

### DNS Resolution (`dns`)

The `dns` package handles EMSG server discovery via DNS.

```go
// Create resolver
resolver := dns.NewResolver(dns.DefaultResolverConfig())

// Resolve domain to server info
serverInfo, err := resolver.ResolveDomain("example.com")
fmt.Printf("Server URL: %s\n", serverInfo.URL)
fmt.Printf("Public Key: %s\n", serverInfo.PublicKey)

// Create cached resolver (recommended)
cachedResolver := dns.NewCachedResolver(dns.DefaultResolverConfig(), 5*time.Minute)
serverInfo, err = cachedResolver.ResolveDomain("example.com")
```

### Message Handling (`message`)

The `message` package provides message composition, signing, and validation.

```go
// Create message builder
builder := message.NewMessageBuilder()

// Build message
msg, err := builder.
    From("alice#example.com").
    To("bob#test.org", "charlie#example.net").
    CC("dave#example.org").
    Subject("Meeting Tomorrow").
    Body("Don't forget about our meeting tomorrow at 2 PM.").
    GroupID("team-alpha").
    Build()

// Sign message
err = msg.Sign(keyPair)

// Verify message signature
err = msg.Verify(publicKeyBase64)

// Serialize to JSON
jsonData, err := msg.ToJSON()

// Deserialize from JSON
msg, err = message.FromJSON(jsonData)

// Validate message structure
err = msg.Validate()

// Get all recipients (To + CC)
recipients := msg.GetRecipients()

// Check if message is signed
signed := msg.IsSigned()

// Clone message
msgCopy := msg.Clone()
```

### High-Level Client (`client`)

The `client` package provides a high-level interface for EMSG operations.

```go
// Create client with configuration
config := client.DefaultConfig()
config.KeyPair = keyPair
config.Timeout = 30 * time.Second
emsgClient := client.New(config)

// Or create with just a key pair
emsgClient := client.NewWithKeyPair(keyPair)

// Compose and send message
msg, err := emsgClient.ComposeMessage().
    From("alice#example.com").
    To("bob#test.org").
    Body("Hello!").
    Build()

err = emsgClient.SendMessage(msg)

// Register user
err = emsgClient.RegisterUser("alice#example.com")

// Get messages
messages, err := emsgClient.GetMessages("alice#example.com")

// Resolve domain
serverInfo, err := emsgClient.ResolveDomain("example.com")
```

## Enhanced Features

### System Messages

The SDK provides built-in support for system message types commonly used in messaging applications:

```go
// Create system message for user joining
joinedMsg, err := message.NewUserJoinedMessage(
    "system#example.com",
    []string{"group#example.com"},
    "alice#example.com",  // actor (who joined)
    "team-alpha",         // group ID
)

// Create system message for user leaving
leftMsg, err := message.NewUserLeftMessage(
    "system#example.com",
    []string{"group#example.com"},
    "bob#example.com",    // actor (who left)
    "team-alpha",         // group ID
)

// Create system message for user being removed
removedMsg, err := message.NewUserRemovedMessage(
    "system#example.com",
    []string{"group#example.com"},
    "admin#example.com",  // actor (who removed)
    "charlie#example.com", // target (who was removed)
    "team-alpha",         // group ID
)

// Create custom system message
customMsg, err := emsgClient.ComposeSystemMessage().
    Type("system:custom_event").
    Actor("user#example.com").
    Target("resource#example.com").
    GroupID("project-gamma").
    Metadata("action", "file_uploaded").
    Metadata("filename", "document.pdf").
    Build("system#example.com", []string{"team#example.com"})

// Check if message is a system message
if msg.IsSystemMessage() {
    systemData, err := msg.GetSystemMessage()
    fmt.Printf("System event: %s by %s\n", systemData.Type, systemData.Actor)
}
```

### Retry Logic and Rate Limiting

Configure automatic retry behavior for handling rate limits and network issues:

```go
config := client.DefaultConfig()
config.KeyPair = keyPair

// Configure retry strategy
config.RetryStrategy = &client.RetryStrategy{
    MaxRetries:      5,                    // Maximum retry attempts
    InitialDelay:    1 * time.Second,      // Initial delay before first retry
    MaxDelay:        30 * time.Second,     // Maximum delay between retries
    BackoffFactor:   2.0,                  // Exponential backoff multiplier
    RetryOn429:      true,                 // Retry on HTTP 429 (rate limit)
    RetryOnTimeout:  true,                 // Retry on timeout errors
}

emsgClient := client.New(config)

// Messages will automatically retry on rate limits with exponential backoff
err := emsgClient.SendMessage(msg)
```

### Developer Hooks

Add custom logic before and after message sending:

```go
config := client.DefaultConfig()
config.KeyPair = keyPair

// Hook called before sending each message
config.BeforeSend = func(msg *message.Message) error {
    log.Printf("Sending message from %s to %v", msg.From, msg.To)

    // Add custom headers, modify message, or perform validation
    if msg.Subject == "" {
        msg.Subject = "Auto-generated subject"
    }

    return nil // Return error to abort sending
}

// Hook called after successful message sending
config.AfterSend = func(msg *message.Message, resp *http.Response) error {
    log.Printf("Message sent successfully with status %d", resp.StatusCode)

    // Log metrics, update database, send notifications, etc.
    return nil // Errors are logged but don't affect the send operation
}

emsgClient := client.New(config)
```

## Command Line Examples

The SDK includes example CLI applications in the `examples/` directory.

### Send Message

```bash
# Generate a key pair
go run examples/send_message.go -generate-key -key=my-key.txt

# Send a message
go run examples/send_message.go \
    -key=my-key.txt \
    -from=alice#example.com \
    -to=bob#test.org \
    -subject="Hello" \
    -body="Hello, Bob!"
```

### Register User

```bash
# Register a user
go run examples/register_user.go \
    -key=my-key.txt \
    -address=alice#example.com
```

### Get Messages

```bash
# Retrieve messages
go run examples/get_messages.go \
    -key=my-key.txt \
    -address=alice#example.com
```

## EMSG Address Format

EMSG addresses use the format `user#domain.com`, similar to email addresses but with `#` instead of `@`.

Examples:
- `alice#example.com`
- `bob.smith#test.org`
- `user_123#sub.domain.co.uk`

## DNS Configuration

EMSG servers are discovered via DNS TXT records at `_emsg.domain.com`. The TXT record can contain:

### JSON Format
```
{"url": "https://emsg.example.com", "pubkey": "base64-encoded-public-key", "version": "1.0"}
```

### URL Format
```
https://emsg.example.com
```

### Key-Value Format
```
url=https://emsg.example.com pubkey=base64-encoded-public-key version=1.0
```

## Testing

The SDK includes comprehensive unit tests and integration tests:

### Unit Tests

```bash
# Run all unit tests
go test ./test/...

# Run tests with coverage
go test -cover ./test/...

# Run specific test
go test ./test/ -run TestGenerateKeyPair
```

### Integration Tests

```bash
# Run mock server integration tests
go test ./integration/ -v

# Run tests against real EMSG server (sandipwalke.com)
INTEGRATION_TEST=real go test ./integration/ -run TestWithRealEMSGServer -v

# Run retry logic tests
INTEGRATION_TEST=retry go test ./integration/ -run TestRetryWithRealServer -v

# Run performance tests
INTEGRATION_TEST=performance go test ./integration/ -run TestPerformance -v
```

### Enhanced Features Demo

Run the comprehensive demo showcasing all enhanced features:

```bash
go run examples/enhanced_features_demo.go
```

## Project Structure

```
emsg-client-sdk/
├── go.mod                      # Module definition
├── client/                     # High-level API layer with retry logic and hooks
│   └── client.go
├── keymgmt/                    # Key generation and storage
│   └── key.go
├── auth/                       # Authentication headers
│   └── auth.go
├── message/                    # Message creation, validation, and system messages
│   └── message.go
├── dns/                        # EMSG DNS resolution
│   └── resolver.go
├── utils/                      # Helper functions
│   └── helpers.go
├── examples/                   # CLI examples and demos
│   ├── send_message.go
│   ├── register_user.go
│   ├── get_messages.go
│   └── enhanced_features_demo.go
├── test/                       # Unit tests
│   ├── keymgmt_test.go
│   ├── auth_test.go
│   ├── utils_test.go
│   ├── message_test.go
│   ├── system_message_test.go
│   └── client_enhancements_test.go
├── integration/                # Integration tests
│   ├── integration_test.go
│   ├── docker_test.go
│   └── README.md
├── DEPLOYMENT.md               # Production deployment guide
├── QUICK_START.md              # 5-minute setup guide
├── PROJECT_SUMMARY.md          # Executive summary
└── README.md
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Security

This SDK implements Ed25519 digital signatures for message authentication and uses secure random number generation for nonces and key generation. All cryptographic operations use Go's standard `crypto` packages.

For security issues, please email security@emsg-protocol.org instead of using the issue tracker.

## Support

- 📖 [Documentation](https://pkg.go.dev/github.com/emsg-protocol/emsg-client-sdk)
- 🐛 [Issue Tracker](https://github.com/emsg-protocol/emsg-client-sdk/issues)
- 💬 [Discussions](https://github.com/emsg-protocol/emsg-client-sdk/discussions)

## Related Projects

- [EMSG Daemon](https://github.com/emsg-protocol/emsg-daemon) - The official EMSG server implementation
- [EMSG Protocol Specification](https://github.com/emsg-protocol/specification) - The EMSG protocol specification
```
