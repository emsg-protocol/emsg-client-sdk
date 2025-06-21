package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
)

// AuthPayload represents the authentication payload structure
type AuthPayload struct {
	Method    string
	Path      string
	Timestamp int64
	Nonce     string
}

// GenerateNonce creates a random nonce for authentication
func GenerateNonce() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// NewAuthPayload creates a new authentication payload
func NewAuthPayload(method, path string) (*AuthPayload, error) {
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, err
	}

	return &AuthPayload{
		Method:    strings.ToUpper(method),
		Path:      path,
		Timestamp: time.Now().Unix(),
		Nonce:     nonce,
	}, nil
}

// String returns the string representation of the auth payload for signing
// Format: METHOD:PATH:TIMESTAMP:NONCE
func (ap *AuthPayload) String() string {
	return fmt.Sprintf("%s:%s:%d:%s", ap.Method, ap.Path, ap.Timestamp, ap.Nonce)
}

// AuthHeader represents an authorization header
type AuthHeader struct {
	PublicKey string
	Signature string
	Timestamp int64
	Nonce     string
}

// GenerateAuthHeader creates a signed authorization header
func GenerateAuthHeader(keyPair *keymgmt.KeyPair, method, path string) (*AuthHeader, error) {
	payload, err := NewAuthPayload(method, path)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth payload: %w", err)
	}

	// Sign the payload
	payloadBytes := []byte(payload.String())
	signature := keyPair.Sign(payloadBytes)

	return &AuthHeader{
		PublicKey: keyPair.PublicKeyBase64(),
		Signature: base64.StdEncoding.EncodeToString(signature),
		Timestamp: payload.Timestamp,
		Nonce:     payload.Nonce,
	}, nil
}

// ToHeaderValue converts the auth header to a string suitable for HTTP Authorization header
func (ah *AuthHeader) ToHeaderValue() string {
	return fmt.Sprintf("EMSG pubkey=%s,signature=%s,timestamp=%d,nonce=%s",
		ah.PublicKey, ah.Signature, ah.Timestamp, ah.Nonce)
}

// ParseAuthHeader parses an authorization header value
func ParseAuthHeader(headerValue string) (*AuthHeader, error) {
	if !strings.HasPrefix(headerValue, "EMSG ") {
		return nil, fmt.Errorf("invalid auth header format: missing EMSG prefix")
	}

	// Remove "EMSG " prefix
	params := strings.TrimPrefix(headerValue, "EMSG ")

	// Parse key-value pairs
	pairs := strings.Split(params, ",")
	authHeader := &AuthHeader{}

	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "pubkey":
			authHeader.PublicKey = value
		case "signature":
			authHeader.Signature = value
		case "timestamp":
			timestamp, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid timestamp: %w", err)
			}
			authHeader.Timestamp = timestamp
		case "nonce":
			authHeader.Nonce = value
		}
	}

	// Validate required fields
	if authHeader.PublicKey == "" || authHeader.Signature == "" ||
		authHeader.Timestamp == 0 || authHeader.Nonce == "" {
		return nil, fmt.Errorf("missing required auth header fields")
	}

	return authHeader, nil
}

// VerifyAuthHeader verifies an authorization header against a method and path
func VerifyAuthHeader(authHeader *AuthHeader, method, path string) error {
	// Load public key
	publicKey, err := keymgmt.LoadPublicKeyFromBase64(authHeader.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to load public key: %w", err)
	}

	// Reconstruct the payload
	payload := &AuthPayload{
		Method:    strings.ToUpper(method),
		Path:      path,
		Timestamp: authHeader.Timestamp,
		Nonce:     authHeader.Nonce,
	}

	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(authHeader.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify signature
	payloadBytes := []byte(payload.String())
	if !ed25519.Verify(publicKey, payloadBytes, signature) {
		return fmt.Errorf("signature verification failed")
	}

	// Check timestamp (allow 5 minute window)
	now := time.Now().Unix()
	if abs(now-authHeader.Timestamp) > 300 {
		return fmt.Errorf("timestamp too old or too far in future")
	}

	return nil
}

// abs returns the absolute value of x
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
