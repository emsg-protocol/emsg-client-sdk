# EMSG Client SDK - Deployment Guide

## 🏗️ Architecture Overview

The EMSG Client SDK is a **client-side library** that does not require server deployment. It runs locally in your applications and communicates with remote EMSG servers.

```
┌─────────────────┐    HTTP/HTTPS    ┌──────────────────┐
│  Your App +     │ ───────────────> │   EMSG Server    │
│  EMSG Client    │                  │ (emsg.domain.com)│
│     SDK         │ <─────────────── │                  │
└─────────────────┘                  └──────────────────┘
   (Runs Locally)                      (Runs on Server)
```

## 🚀 Deployment Options

### Option 1: Go Module Integration (Recommended)

**Best for:** Production applications, libraries, services

```bash
# Add to your Go project
go get github.com/emsg-protocol/emsg-client-sdk
```

**Usage in your application:**
```go
package main

import (
    "log"
    "github.com/emsg-protocol/emsg-client-sdk/client"
    "github.com/emsg-protocol/emsg-client-sdk/keymgmt"
)

func main() {
    // Load or generate keys
    keyPair, err := keymgmt.GenerateKeyPair()
    if err != nil {
        log.Fatal(err)
    }
    
    // Create client
    emsgClient := client.NewWithKeyPair(keyPair)
    
    // Send messages
    msg, _ := emsgClient.ComposeMessage().
        From("user#yourdomain.com").
        To("recipient#targetdomain.com").
        Body("Hello from my app!").
        Build()
    
    err = emsgClient.SendMessage(msg)
    if err != nil {
        log.Printf("Failed to send: %v", err)
    }
}
```

### Option 2: Standalone CLI Tools

**Best for:** Scripts, automation, testing, one-off operations

```bash
# Clone the repository
git clone https://github.com/emsg-protocol/emsg-client-sdk
cd emsg-client-sdk

# Use directly
go run examples/send_message.go -generate-key -key=my-key.txt
go run examples/send_message.go -key=my-key.txt -from=user#domain.com -to=recipient#domain.com -body="Hello!"
```

### Option 3: Compiled Executables

**Best for:** Distribution, CI/CD, cross-platform deployment

```bash
# Build executables
go build -o emsg-send examples/send_message.go
go build -o emsg-register examples/register_user.go
go build -o emsg-get examples/get_messages.go

# Distribute and use
./emsg-send -key=my-key.txt -from=user@domain.com -to=recipient@domain.com -body="Hello!"
```

**Cross-platform builds:**
```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o emsg-send-linux examples/send_message.go

# Windows
GOOS=windows GOARCH=amd64 go build -o emsg-send.exe examples/send_message.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o emsg-send-mac examples/send_message.go
```

### Option 4: Docker Container

**Best for:** Containerized environments, microservices

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o emsg-client examples/send_message.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/emsg-client .
CMD ["./emsg-client"]
```

```bash
# Build and run
docker build -t emsg-client .
docker run -v $(pwd)/keys:/keys emsg-client -key=/keys/my-key.txt -from=user#domain.com -to=recipient#domain.com -body="Hello from Docker!"
```

## 🌐 Real-World Deployment Scenarios

### 1. Web Application Backend

```go
// main.go - Web server with EMSG integration
package main

import (
    "net/http"
    "github.com/emsg-protocol/emsg-client-sdk/client"
    "github.com/emsg-protocol/emsg-client-sdk/keymgmt"
)

var emsgClient *client.Client

func init() {
    keyPair, _ := keymgmt.LoadPrivateKeyFromFile("server-key.txt")
    emsgClient = client.NewWithKeyPair(keyPair)
}

func sendNotificationHandler(w http.ResponseWriter, r *http.Request) {
    // Extract user and message from request
    userEmail := r.FormValue("user")
    messageText := r.FormValue("message")
    
    // Send EMSG message
    msg, _ := emsgClient.ComposeMessage().
        From("notifications#myapp.com").
        To(userEmail).
        Subject("App Notification").
        Body(messageText).
        Build()
    
    err := emsgClient.SendMessage(msg)
    if err != nil {
        http.Error(w, "Failed to send notification", 500)
        return
    }
    
    w.WriteHeader(200)
    w.Write([]byte("Notification sent"))
}

func main() {
    http.HandleFunc("/send-notification", sendNotificationHandler)
    http.ListenAndServe(":8080", nil)
}
```

**Deployment:**
```bash
# Build and deploy
go build -o notification-server main.go
./notification-server
```

### 2. Microservice Integration

```go
// notification-service/main.go
package main

import (
    "context"
    "log"
    "github.com/emsg-protocol/emsg-client-sdk/client"
)

type NotificationService struct {
    emsgClient *client.Client
}

func (ns *NotificationService) SendUserNotification(ctx context.Context, userID, message string) error {
    userAddress := fmt.Sprintf("%s#myapp.com", userID)
    
    msg, err := ns.emsgClient.ComposeMessage().
        From("system#myapp.com").
        To(userAddress).
        Body(message).
        Build()
    
    if err != nil {
        return err
    }
    
    return ns.emsgClient.SendMessage(msg)
}
```

**Kubernetes Deployment:**
```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notification-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: notification-service
  template:
    metadata:
      labels:
        app: notification-service
    spec:
      containers:
      - name: notification-service
        image: myapp/notification-service:latest
        env:
        - name: EMSG_KEY_PATH
          value: "/keys/service-key.txt"
        volumeMounts:
        - name: emsg-keys
          mountPath: /keys
          readOnly: true
      volumes:
      - name: emsg-keys
        secret:
          secretName: emsg-service-keys
```

### 3. Desktop Application

```go
// desktop-app/main.go
package main

import (
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/widget"
    "github.com/emsg-protocol/emsg-client-sdk/client"
)

func main() {
    myApp := app.New()
    myWindow := myApp.NewWindow("EMSG Desktop Client")
    
    // Load user's key
    keyPair, _ := keymgmt.LoadPrivateKeyFromFile("user-key.txt")
    emsgClient := client.NewWithKeyPair(keyPair)
    
    // Create UI
    toEntry := widget.NewEntry()
    messageEntry := widget.NewMultiLineEntry()
    
    sendButton := widget.NewButton("Send Message", func() {
        msg, _ := emsgClient.ComposeMessage().
            From("user#myapp.com").
            To(toEntry.Text).
            Body(messageEntry.Text).
            Build()
        
        emsgClient.SendMessage(msg)
    })
    
    // Layout and show
    content := container.NewVBox(toEntry, messageEntry, sendButton)
    myWindow.SetContent(content)
    myWindow.ShowAndRun()
}
```

### 4. CI/CD Pipeline Integration

```yaml
# .github/workflows/notify.yml
name: Build and Notify
on: [push]

jobs:
  build-and-notify:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    
    - name: Setup Go
      uses: actions/setup-go@v2
      with:
        go-version: 1.21
    
    - name: Build
      run: go build ./...
    
    - name: Send Build Notification
      run: |
        go run examples/send_message.go \
          -key="${{ secrets.EMSG_KEY }}" \
          -from="ci#mycompany.com" \
          -to="team#mycompany.com" \
          -subject="Build Complete" \
          -body="Build ${{ github.sha }} completed successfully"
```

## 📋 Prerequisites

### System Requirements
- **Go 1.19+** (for building from source)
- **Network access** to target EMSG servers
- **DNS resolution** capability for EMSG server discovery

### EMSG Server Requirements
- Target domains must have **EMSG DNS records** configured
- EMSG daemon must be **running and accessible**
- Proper **API endpoints** must be available

### Key Management
- **Ed25519 key pairs** for authentication
- **Secure key storage** (files, environment variables, or key management systems)
- **Key rotation** strategy for production environments

## 🔧 Configuration

### Environment Variables
```bash
# Optional environment variables
export EMSG_KEY_PATH="/path/to/private-key.txt"
export EMSG_DEFAULT_FROM="myapp#mydomain.com"
export EMSG_TIMEOUT="30s"
export EMSG_DNS_TTL="5m"
```

### Configuration File
```go
// config.go
package main

import (
    "time"
    "github.com/emsg-protocol/emsg-client-sdk/client"
    "github.com/emsg-protocol/emsg-client-sdk/dns"
)

func createEMSGClient() *client.Client {
    config := client.DefaultConfig()
    config.Timeout = 30 * time.Second
    config.UserAgent = "MyApp/1.0"
    config.DNSConfig = &dns.ResolverConfig{
        Timeout: 10 * time.Second,
        Retries: 3,
    }
    config.DNSTTL = 5 * time.Minute

    return client.New(config)
}
```

## 🔐 Security Considerations

### Key Management Best Practices

#### Development Environment
```bash
# Generate development keys
go run examples/send_message.go -generate-key -key=dev-key.txt

# Store in project (add to .gitignore)
echo "*.key" >> .gitignore
echo "*-key.txt" >> .gitignore
```

#### Production Environment
```bash
# Use environment variables
export EMSG_PRIVATE_KEY="$(cat /secure/path/to/production-key.txt)"

# Or use key management services
# AWS Secrets Manager, HashiCorp Vault, etc.
```

#### Docker Secrets
```yaml
# docker-compose.yml
version: '3.8'
services:
  app:
    image: myapp:latest
    secrets:
      - emsg_private_key
    environment:
      - EMSG_KEY_PATH=/run/secrets/emsg_private_key

secrets:
  emsg_private_key:
    file: ./secrets/emsg-key.txt
```

### Network Security
```go
// Use TLS for production
config := client.DefaultConfig()
config.Timeout = 30 * time.Second

// The SDK automatically uses HTTPS when servers specify it
// Ensure your EMSG servers use HTTPS in production
```

## 🧪 Testing Your Deployment

### Unit Testing
```go
// main_test.go
package main

import (
    "testing"
    "github.com/emsg-protocol/emsg-client-sdk/keymgmt"
    "github.com/emsg-protocol/emsg-client-sdk/client"
)

func TestEMSGIntegration(t *testing.T) {
    // Generate test key
    keyPair, err := keymgmt.GenerateKeyPair()
    if err != nil {
        t.Fatal(err)
    }

    // Create client
    emsgClient := client.NewWithKeyPair(keyPair)

    // Test message creation
    msg, err := emsgClient.ComposeMessage().
        From("test#example.com").
        To("recipient#example.com").
        Body("Test message").
        Build()

    if err != nil {
        t.Fatal(err)
    }

    if msg.From != "test#example.com" {
        t.Errorf("Expected from test#example.com, got %s", msg.From)
    }
}
```

### Integration Testing
```bash
# Test with real server (sandipwalke.com)
go run examples/send_message.go \
  -generate-key \
  -key=test-key.txt

go run examples/send_message.go \
  -key=test-key.txt \
  -from="testuser#sandipwalke.com" \
  -to="recipient#sandipwalke.com" \
  -subject="Integration Test" \
  -body="Testing EMSG SDK deployment"
```

### Health Check Endpoint
```go
// Add to your web service
func healthHandler(w http.ResponseWriter, r *http.Request) {
    // Test EMSG connectivity
    resolver := dns.NewResolver(dns.DefaultResolverConfig())
    _, err := resolver.ResolveDomain("yourdomain.com")

    if err != nil {
        w.WriteHeader(503)
        w.Write([]byte("EMSG service unavailable"))
        return
    }

    w.WriteHeader(200)
    w.Write([]byte("OK"))
}
```

## 🚨 Troubleshooting

### Common Issues

#### DNS Resolution Failures
```bash
# Test DNS manually
nslookup -type=TXT _emsg.yourdomain.com

# Expected output:
# _emsg.yourdomain.com text = "https://emsg.yourdomain.com:8765"
```

#### Connection Timeouts
```go
// Increase timeout for slow networks
config := client.DefaultConfig()
config.Timeout = 60 * time.Second
config.DNSConfig.Timeout = 20 * time.Second
```

#### Authentication Failures
```bash
# Verify key format
go run -c "
import 'github.com/emsg-protocol/emsg-client-sdk/keymgmt'
keyPair, err := keymgmt.LoadPrivateKeyFromFile('your-key.txt')
if err != nil { panic(err) }
fmt.Println('Key loaded successfully')
fmt.Println('Public key:', keyPair.PublicKeyBase64())
"
```

#### Server Endpoint Issues
```bash
# Test server connectivity
curl -v https://emsg.yourdomain.com:8765/api/v1/health

# Check for 404 errors - endpoints might be different
curl -v https://emsg.yourdomain.com:8765/health
curl -v https://emsg.yourdomain.com:8765/users
curl -v https://emsg.yourdomain.com:8765/messages
```

## 📊 Monitoring and Logging

### Application Logging
```go
import "log"

// Add logging to your EMSG operations
func sendMessage(client *client.Client, msg *message.Message) error {
    log.Printf("Sending message from %s to %v", msg.From, msg.To)

    err := client.SendMessage(msg)
    if err != nil {
        log.Printf("Failed to send message: %v", err)
        return err
    }

    log.Printf("Message sent successfully: %s", msg.MessageID)
    return nil
}
```

### Metrics Collection
```go
import "github.com/prometheus/client_golang/prometheus"

var (
    messagesSent = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "emsg_messages_sent_total",
            Help: "Total number of EMSG messages sent",
        },
        []string{"from_domain", "to_domain", "status"},
    )
)

func sendMessageWithMetrics(client *client.Client, msg *message.Message) error {
    err := client.SendMessage(msg)

    status := "success"
    if err != nil {
        status = "error"
    }

    messagesSent.WithLabelValues(
        extractDomain(msg.From),
        extractDomain(msg.To[0]),
        status,
    ).Inc()

    return err
}
```

## 🎯 Summary

The EMSG Client SDK is designed for **client-side deployment** and does not require server infrastructure. Choose the deployment option that best fits your use case:

- **🏢 Enterprise Applications**: Use as Go module with proper key management
- **🔧 DevOps/Automation**: Use CLI tools in scripts and CI/CD
- **📱 Desktop/Mobile**: Embed in applications with local key storage
- **🐳 Containerized**: Deploy in Docker/Kubernetes with secret management

**Key Points:**
- ✅ No server deployment required for the SDK
- ✅ Runs locally in your applications
- ✅ Communicates with remote EMSG servers
- ✅ Supports multiple deployment patterns
- ✅ Production-ready with proper security practices
```
