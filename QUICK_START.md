# EMSG Client SDK - Quick Start Guide

## 🚀 5-Minute Setup

### Step 1: Install the SDK
```bash
go get github.com/emsg-protocol/emsg-client-sdk
```

### Step 2: Generate Keys
```bash
# Clone the repo for examples
git clone https://github.com/emsg-protocol/emsg-client-sdk
cd emsg-client-sdk

# Generate your key pair
go run examples/send_message.go -generate-key -key=my-key.txt
```

### Step 3: Send Your First Message
```bash
# Send a test message
go run examples/send_message.go \
  -key=my-key.txt \
  -from="yourname#yourdomain.com" \
  -to="recipient#targetdomain.com" \
  -subject="Hello EMSG!" \
  -body="This is my first EMSG message!"
```

## 📝 Common Use Cases

### 1. Simple Message Sending
```go
package main

import (
    "log"
    "github.com/emsg-protocol/emsg-client-sdk/client"
    "github.com/emsg-protocol/emsg-client-sdk/keymgmt"
)

func main() {
    // Load your key
    keyPair, err := keymgmt.LoadPrivateKeyFromFile("my-key.txt")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create client
    emsgClient := client.NewWithKeyPair(keyPair)
    
    // Send message
    msg, _ := emsgClient.ComposeMessage().
        From("alice#example.com").
        To("bob#test.org").
        Subject("Hello!").
        Body("Hello Bob, how are you?").
        Build()
    
    err = emsgClient.SendMessage(msg)
    if err != nil {
        log.Printf("Failed: %v", err)
    } else {
        log.Println("Message sent!")
    }
}
```

### 2. Group Messaging
```go
func sendGroupMessage() {
    emsgClient := client.NewWithKeyPair(keyPair)
    
    msg, _ := emsgClient.ComposeMessage().
        From("admin#company.com").
        To("alice#company.com", "bob#company.com").
        CC("manager#company.com").
        Subject("Team Meeting").
        Body("Team meeting tomorrow at 2 PM").
        GroupID("team-alpha").
        Build()
    
    emsgClient.SendMessage(msg)
}
```

### 3. User Registration
```go
func registerUser() {
    emsgClient := client.NewWithKeyPair(keyPair)
    
    err := emsgClient.RegisterUser("newuser#mydomain.com")
    if err != nil {
        log.Printf("Registration failed: %v", err)
    } else {
        log.Println("User registered successfully!")
    }
}
```

### 4. Retrieve Messages
```go
func getMessages() {
    emsgClient := client.NewWithKeyPair(keyPair)
    
    messages, err := emsgClient.GetMessages("myuser#mydomain.com")
    if err != nil {
        log.Printf("Failed to get messages: %v", err)
        return
    }
    
    for _, msg := range messages {
        fmt.Printf("From: %s\n", msg.From)
        fmt.Printf("Subject: %s\n", msg.Subject)
        fmt.Printf("Body: %s\n", msg.Body)
        fmt.Println("---")
    }
}
```

## 🔧 CLI Usage Examples

### Generate Keys
```bash
# Generate new key pair
go run examples/send_message.go -generate-key -key=alice-key.txt

# Output shows your public key
# Public key: k0aEc5bYLQZ6sd49DJc/yVNdfH2jsIfz9UShZQa2Khk=
```

### Send Messages
```bash
# Basic message
go run examples/send_message.go \
  -key=alice-key.txt \
  -from="alice#example.com" \
  -to="bob#test.org" \
  -body="Hello Bob!"

# Message with subject and CC
go run examples/send_message.go \
  -key=alice-key.txt \
  -from="alice#example.com" \
  -to="bob#test.org,charlie@example.net" \
  -cc="manager#example.com" \
  -subject="Project Update" \
  -body="The project is on track for delivery next week."

# Group message
go run examples/send_message.go \
  -key=alice-key.txt \
  -from="alice#example.com" \
  -to="team#example.com" \
  -group="project-alpha" \
  -subject="Sprint Review" \
  -body="Sprint review meeting scheduled for Friday."
```

### Register Users
```bash
# Register a new user
go run examples/register_user.go \
  -key=alice-key.txt \
  -address="alice#example.com"

# Generate key and register in one step
go run examples/register_user.go \
  -generate-key \
  -key=newuser-key.txt \
  -address="newuser#example.com"
```

### Get Messages
```bash
# Retrieve messages for a user
go run examples/get_messages.go \
  -key=alice-key.txt \
  -address="alice#example.com"
```

## 🌐 Testing with Real Servers

### Test with sandipwalke.com
```bash
# Generate test key
go run examples/send_message.go -generate-key -key=test-key.txt

# Test user registration
go run examples/register_user.go \
  -key=test-key.txt \
  -address="testuser#sandipwalke.com"

# Send test message
go run examples/send_message.go \
  -key=test-key.txt \
  -from="testuser#sandipwalke.com" \
  -to="recipient#sandipwalke.com" \
  -subject="SDK Test" \
  -body="Testing the EMSG Client SDK!"
```

### Verify DNS Configuration
```bash
# Check if domain has EMSG DNS records
nslookup -type=TXT _emsg.sandipwalke.com

# Expected output:
# _emsg.sandipwalke.com text = "https://emsg.sandipwalke.com:8765"
```

## 🔍 Troubleshooting Quick Fixes

### Issue: "invalid address format"
```bash
# ❌ Wrong format (using @)
-from="user@domain.com"

# ✅ Correct format (using #)
-from="user#domain.com"
```

### Issue: "failed to resolve domain"
```bash
# Check DNS records
nslookup -type=TXT _emsg.yourdomain.com

# If no records found, contact domain administrator
```

### Issue: "HTTP request failed with status 404"
```bash
# Server is reachable but endpoints might be different
# This is expected if server uses different API paths
# The SDK is working correctly!
```

### Issue: "failed to load private key"
```bash
# Check file exists and has correct permissions
ls -la your-key.txt

# Regenerate if corrupted
go run examples/send_message.go -generate-key -key=new-key.txt
```

## 📚 Next Steps

### For Developers
1. **Read the full [README.md](README.md)** for complete API documentation
2. **Check [DEPLOYMENT.md](DEPLOYMENT.md)** for production deployment guides
3. **Explore the [examples/](examples/)** directory for more use cases
4. **Run the test suite** with `go test ./test/`

### For System Administrators
1. **Set up EMSG DNS records** for your domain
2. **Deploy EMSG daemon** on your servers
3. **Configure proper endpoints** and authentication
4. **Set up monitoring** for EMSG services

### For Security Teams
1. **Review key management** practices in your organization
2. **Implement proper key rotation** procedures
3. **Set up secure key storage** (HSM, key vaults, etc.)
4. **Audit EMSG communications** and access patterns

## 🎯 Key Takeaways

- ✅ **No server deployment needed** - SDK runs in your applications
- ✅ **Simple integration** - Just add as Go module dependency
- ✅ **Secure by default** - Ed25519 cryptography and signed messages
- ✅ **Production ready** - Comprehensive testing and error handling
- ✅ **Cross-platform** - Works on Linux, Windows, macOS
- ✅ **Well documented** - Complete API reference and examples

**Start building secure messaging into your applications today!** 🚀
