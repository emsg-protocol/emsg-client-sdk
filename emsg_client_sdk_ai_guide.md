# EMSG Client SDK - AI Agent Instruction Document

## 🔄 Project Title:

**emsg-client-sdk**

## 💪 Objective:

To build a cross-platform, developer-friendly client SDK for the EMSG (Electronic Message) protocol in **Go**, enabling secure communication, message routing, and cryptographic authentication.

The SDK will empower developers to:

- Build apps that talk to EMSG Daemon via REST API
- Generate and manage Ed25519 key pairs
- Sign and send messages
- Resolve EMSG addresses via DNS
- Authenticate requests

## 🌐 Project Repository:

[https://github.com/emsg-protocol/emsg-client-sdk](https://github.com/emsg-protocol/emsg-client-sdk)

---

## 🔢 Language:

**Go (Golang)**

---

## 🔍 Goals for the AI Agent:

1. **Design and implement** a clean, idiomatic Go package structure.
2. **Develop reusable modules** for:
   - Cryptographic key generation and signing
   - DNS-based routing
   - Message construction
   - Authentication header generation
   - EMSG address parsing/validation
3. **Write clear documentation and examples** for usage.
4. **Ensure test coverage** for all modules.
5. **Provide compatibility** with the EMSG Daemon defined protocol.

---

## 📁 Project Structure

```bash
emsg-client-sdk/
├── go.mod / go.sum             # Module definition
├── client/                    # Core high-level API layer
│   └── client.go              # Entry point for users
├── keymgmt/                   # Key generation and storage (Ed25519)
│   └── key.go
├── auth/                      # Auth header creation, nonce/timestamp/signing
│   └── auth.go
├── message/                   # Message creation and validation
│   └── message.go
├── dns/                       # EMSG DNS resolution
│   └── resolver.go
├── utils/                     # Common helper functions
│   └── helpers.go
├── examples/                  # Sample usage apps
│   └── send_message.go
├── test/                      # Test files for each module
└── README.md                  # Documentation
```

---

## 🔧 Steps to Perform

### 1. Key Management

- Generate Ed25519 key pair
- Save/load private keys securely (file or in-memory)

### 2. Authentication

- Generate signed auth payloads for protected endpoints:
  ```
  METHOD:PATH:TIMESTAMP:NONCE
  ```
- Construct base64-encoded Authorization headers

### 3. EMSG Address Parser

- Parse address like `user#domain.com`
- Validate format and extract domain

### 4. DNS Resolver

- Perform TXT lookup for `_emsg.domain.com`
- Parse JSON or raw URL formats
- Return resolved server URL and pubkey if available

### 5. Message System

- Construct EMSG message structure (from/to/cc/body/group\_id/signature)
- Sign message with sender key
- Validate message format before send

### 6. Client Interface (High-Level)

Expose user-friendly methods:

```go
sdk := client.New()
msg := sdk.ComposeMessage(...)
err := sdk.SendMessage(msg)
```

### 7. Example Apps

- CLI example: Send message
- CLI example: Register user

### 8. Tests

- Unit tests for each module
- Integration test using mock daemon server

---

## 📖 References

- [EMSG Protocol README](https://github.com/emsg-protocol/emsg-daemon/blob/main/README.md)
- [EMSG Address Format](https://github.com/emsg-protocol/emsg-daemon#address-format)
- Ed25519 docs from Go crypto lib

---

## ✅ Deliverables

- Fully documented Go SDK
- Working CLI example for registration & messaging
- Unit + integration tests
- README with usage and installation

---

## 🚀 Final Notes

This SDK is the official client library of the EMSG Protocol. It must adhere strictly to the protocol spec, promote secure defaults, and support clean extensibility for future frontend or mobile integrations.

