
````markdown
# 🔧 AI Agent Prompt: Add Admin ResetUser Function to emsg-client-sdk

## 🎯 Objective

Enhance the EMSG Client SDK to allow **domain administrators** to programmatically reset a user's public key by calling the `/admin/reset_user` endpoint on the daemon.

This enables secure identity recovery for domains where user accounts are controlled centrally.

---

## 🔐 Background

Only authorized **admin public keys** (defined in the domain policy on the daemon) are allowed to reset user identities. The SDK should generate a secure, signed request from the admin’s private key.

---

## ✅ Tasks

### 1. Add a Method

**File:** `client/client.go`

```go
func (c *Client) ResetUserIdentity(adminAddr, targetAddr, newPubKey string) error
````

This method should:

* Generate a secure `nonce` and `timestamp`
* Build the message input string:

  ```
  input := targetAddr + newPubKey + timestamp + nonce
  ```
* Sign the input using the admin’s private key via `keymgmt`
* Submit the POST request to `/admin/reset_user`

### 2. Define Payload Format

```json
{
  "operation": "reset_user",
  "address": "user#domain.com",
  "new_public_key": "base64...",
  "timestamp": 1234567890,
  "nonce": "unique_nonce",
  "admin_signature": "base64-signature"
}
```

### 3. Handle Server Response

* ✅ 200 OK → Success
* ❌ 403 Forbidden → Invalid signature or unauthorized admin
* ❌ 409 Conflict → Target user doesn’t exist
* ❌ 429 Too Many Requests → Rate-limited

---

## 🧪 CLI Demo Tool

**File:** `examples/reset_user.go`

Create a CLI utility that:

* Accepts flags:

  * `-admin=admin#domain.com`
  * `-target=user#domain.com`
  * `-key=new-user-public-key.txt`
* Loads admin private key (from keymgmt)
* Loads or generates new user public key
* Calls `ResetUserIdentity()`
* Prints success/failure

---

## 📁 Files to Modify

* `client/client.go` — Add main logic
* `examples/reset_user.go` — CLI utility
* `README.md` — Add section under `Admin Tools` or `Reset`
* `test/client_admin_test.go` — Optional: test cases for reset

---

## 🧰 Dependencies

* `keymgmt` for signature and key file loading
* `utils.GenerateNonce()`, `time.Now().Unix()` for metadata
* Go's standard HTTP client (already used in SDK)

---

## 🔗 Related Repos

* [emsg-daemon](https://github.com/emsg-protocol/emsg-daemon)
* [emsg-client-sdk](https://github.com/emsg-protocol/emsg-client-sdk)

---

## 💡 Example Usage

```go
err := emsgClient.ResetUserIdentity(
    "admin#magadhaempire.com",
    "sandip#magadhaempire.com",
    newPublicKeyBase64,
)
```

This will trigger the reset flow for the target user with the new key.

---

```

You can now save this content as `emsg-client-sdk-reset-user.prompt.md` and place it in:

```

emsg-client-sdk/.prompt/emsg-client-sdk-reset-user.prompt.md

```

Would you like me to generate the CLI example code (`reset_user.go`) as well?
```
