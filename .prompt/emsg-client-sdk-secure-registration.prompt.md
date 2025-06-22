Below is your `.prompt.md` document for updating the `emsg-client-sdk` to support **domain-secured user registration**, specifically for:

* **Passing invite codes** (for `invite-only` domains)
* Ensuring the SDK can work smoothly with domain-based policy validation handled by `emsg-daemon`

---

### 📄 `emsg-client-sdk-secure-registration.prompt.md`

````markdown
# 🛡️ AI Agent Prompt: Update Secure User Registration Flow in emsg-client-sdk

## 🎯 Objective

Update the EMSG Client SDK to support domain-enforced user registration policies. The SDK must pass optional **invite codes** and support flexible policy enforcement by the daemon (such as allowlist or admin-approval).

---

## 📌 Background

The `emsg-daemon` enforces per-domain policies like:
- `open`: Anyone can register
- `invite-only`: Requires invite code
- `allowlist`: Username must be pre-approved
- `admin-approval`: Manual admin approval required

The SDK must support this securely while remaining easy for developers to use.

---

## ✅ Required Changes

### 1. Enhance `RegisterUser` Method

**File:** `client/client.go`

Update the `RegisterUser` function to accept an optional invite code:

```go
func (c *Client) RegisterUser(address string, inviteCode ...string) error
````

**Logic:**

* Generate nonce, timestamp, and signature
* If `inviteCode` is provided, include it in the payload
* Submit to `/register`

### 2. JSON Payload Format

```json
{
  "address": "user#domain.com",
  "public_key": "base64...",
  "timestamp": 1723812738,
  "nonce": "random-string",
  "signature": "base64-sig",
  "invite_code": "OPTIONAL-CODE"
}
```

> `invite_code` must be included **only** if provided.

---

## 🔄 Optional Helper Method

Create a helper like:

```go
func (c *Client) RegisterUserWithInvite(address, inviteCode string) error {
    return c.RegisterUser(address, inviteCode)
}
```

---

## 🧪 CLI Example Update

**File:** `examples/register_user.go`

* Add `-invite` CLI flag
* Pass `inviteCode` if provided
* Example usage:

```bash
go run examples/register_user.go \
  -address=sandip#magadhaempire.com \
  -key=admin-key.txt \
  -invite=TRUSTED2025
```

---

## 🧬 Testing Suggestions

* Try:

  * No invite code (should work only in open domains)
  * Invalid code (should be rejected)
  * Valid code (should succeed in invite-only domains)
* Validate:

  * Signature is accepted
  * Nonce is unique
  * Proper error handling

---

## 📁 Files to Modify

* `client/client.go` — Add `invite_code` support
* `examples/register_user.go` — Add CLI flag
* `README.md` — Document the invite code usage
* `test/registration_test.go` — Optional

---

## 📚 Related Projects

* [emsg-daemon](https://github.com/emsg-protocol/emsg-daemon)
* [emsg-client-sdk](https://github.com/emsg-protocol/emsg-client-sdk)
* [emsg-client-ui](https://github.com/emsg-protocol/emsg-client-ui)

---

## ✅ Outcome

With this update, the client SDK will:

* Be compatible with secure, domain-governed registration rules
* Allow registering only through legitimate domain workflows
* Surface clear errors when domain policy blocks a request

```

---

Save this as:

```

emsg-client-sdk/.prompt/emsg-client-sdk-secure-registration.prompt.md

```
