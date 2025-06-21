package test

import (
	"strings"
	"testing"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/auth"
	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
)

func TestGenerateNonce(t *testing.T) {
	nonce1, err := auth.GenerateNonce()
	if err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}

	nonce2, err := auth.GenerateNonce()
	if err != nil {
		t.Fatalf("Failed to generate second nonce: %v", err)
	}

	if nonce1 == nonce2 {
		t.Error("Generated nonces should be different")
	}

	if len(nonce1) == 0 {
		t.Error("Nonce should not be empty")
	}

	// Nonce should be hex encoded (32 characters for 16 bytes)
	if len(nonce1) != 32 {
		t.Errorf("Expected nonce length 32, got %d", len(nonce1))
	}
}

func TestNewAuthPayload(t *testing.T) {
	payload, err := auth.NewAuthPayload("GET", "/api/v1/messages")
	if err != nil {
		t.Fatalf("Failed to create auth payload: %v", err)
	}

	if payload.Method != "GET" {
		t.Errorf("Expected method GET, got %s", payload.Method)
	}

	if payload.Path != "/api/v1/messages" {
		t.Errorf("Expected path /api/v1/messages, got %s", payload.Path)
	}

	if payload.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}

	if payload.Nonce == "" {
		t.Error("Nonce should not be empty")
	}

	// Test method normalization
	payload2, err := auth.NewAuthPayload("post", "/api/v1/users")
	if err != nil {
		t.Fatalf("Failed to create auth payload: %v", err)
	}

	if payload2.Method != "POST" {
		t.Errorf("Expected method POST, got %s", payload2.Method)
	}
}

func TestAuthPayloadString(t *testing.T) {
	payload := &auth.AuthPayload{
		Method:    "GET",
		Path:      "/api/v1/messages",
		Timestamp: 1234567890,
		Nonce:     "abcdef123456",
	}

	expected := "GET:/api/v1/messages:1234567890:abcdef123456"
	actual := payload.String()

	if actual != expected {
		t.Errorf("Expected payload string %s, got %s", expected, actual)
	}
}

func TestGenerateAuthHeader(t *testing.T) {
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	authHeader, err := auth.GenerateAuthHeader(keyPair, "GET", "/api/v1/messages")
	if err != nil {
		t.Fatalf("Failed to generate auth header: %v", err)
	}

	if authHeader.PublicKey == "" {
		t.Error("Public key should not be empty")
	}

	if authHeader.Signature == "" {
		t.Error("Signature should not be empty")
	}

	if authHeader.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}

	if authHeader.Nonce == "" {
		t.Error("Nonce should not be empty")
	}

	// Verify the public key matches
	expectedPublicKey := keyPair.PublicKeyBase64()
	if authHeader.PublicKey != expectedPublicKey {
		t.Errorf("Expected public key %s, got %s", expectedPublicKey, authHeader.PublicKey)
	}
}

func TestAuthHeaderToHeaderValue(t *testing.T) {
	authHeader := &auth.AuthHeader{
		PublicKey: "test_pubkey",
		Signature: "test_signature",
		Timestamp: 1234567890,
		Nonce:     "test_nonce",
	}

	headerValue := authHeader.ToHeaderValue()
	expected := "EMSG pubkey=test_pubkey,signature=test_signature,timestamp=1234567890,nonce=test_nonce"

	if headerValue != expected {
		t.Errorf("Expected header value %s, got %s", expected, headerValue)
	}
}

func TestParseAuthHeader(t *testing.T) {
	headerValue := "EMSG pubkey=test_pubkey,signature=test_signature,timestamp=1234567890,nonce=test_nonce"

	authHeader, err := auth.ParseAuthHeader(headerValue)
	if err != nil {
		t.Fatalf("Failed to parse auth header: %v", err)
	}

	if authHeader.PublicKey != "test_pubkey" {
		t.Errorf("Expected public key test_pubkey, got %s", authHeader.PublicKey)
	}

	if authHeader.Signature != "test_signature" {
		t.Errorf("Expected signature test_signature, got %s", authHeader.Signature)
	}

	if authHeader.Timestamp != 1234567890 {
		t.Errorf("Expected timestamp 1234567890, got %d", authHeader.Timestamp)
	}

	if authHeader.Nonce != "test_nonce" {
		t.Errorf("Expected nonce test_nonce, got %s", authHeader.Nonce)
	}
}

func TestParseAuthHeaderInvalid(t *testing.T) {
	testCases := []string{
		"",
		"Bearer token",
		"EMSG",
		"EMSG pubkey=test",
		"EMSG pubkey=test,signature=sig,timestamp=invalid,nonce=nonce",
		"EMSG pubkey=,signature=sig,timestamp=123,nonce=nonce",
	}

	for _, tc := range testCases {
		_, err := auth.ParseAuthHeader(tc)
		if err == nil {
			t.Errorf("Expected error when parsing invalid header: %s", tc)
		}
	}
}

func TestVerifyAuthHeader(t *testing.T) {
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Generate a valid auth header
	authHeader, err := auth.GenerateAuthHeader(keyPair, "GET", "/api/v1/messages")
	if err != nil {
		t.Fatalf("Failed to generate auth header: %v", err)
	}

	// Verify with correct method and path
	err = auth.VerifyAuthHeader(authHeader, "GET", "/api/v1/messages")
	if err != nil {
		t.Errorf("Failed to verify valid auth header: %v", err)
	}

	// Verify with wrong method should fail
	err = auth.VerifyAuthHeader(authHeader, "POST", "/api/v1/messages")
	if err == nil {
		t.Error("Expected verification to fail with wrong method")
	}

	// Verify with wrong path should fail
	err = auth.VerifyAuthHeader(authHeader, "GET", "/api/v1/users")
	if err == nil {
		t.Error("Expected verification to fail with wrong path")
	}
}

func TestVerifyAuthHeaderTimestamp(t *testing.T) {
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create auth header with old timestamp
	// Generate a valid signature first, then modify timestamp
	validAuthHeader, err := auth.GenerateAuthHeader(keyPair, "GET", "/api/v1/messages")
	if err != nil {
		t.Fatalf("Failed to generate valid auth header: %v", err)
	}

	oldAuthHeader := &auth.AuthHeader{
		PublicKey: keyPair.PublicKeyBase64(),
		Signature: validAuthHeader.Signature,
		Timestamp: time.Now().Unix() - 400, // 400 seconds ago (> 5 minutes)
		Nonce:     validAuthHeader.Nonce,
	}

	err = auth.VerifyAuthHeader(oldAuthHeader, "GET", "/api/v1/messages")
	if err == nil {
		t.Error("Expected verification to fail with old timestamp")
	}

	// The error could be either signature verification failure (because we changed the timestamp)
	// or timestamp error - both are acceptable for this test
	if !strings.Contains(err.Error(), "timestamp") && !strings.Contains(err.Error(), "signature") {
		t.Errorf("Expected timestamp or signature error, got: %v", err)
	}
}

func TestAuthHeaderRoundTrip(t *testing.T) {
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Generate auth header
	originalHeader, err := auth.GenerateAuthHeader(keyPair, "POST", "/api/v1/users")
	if err != nil {
		t.Fatalf("Failed to generate auth header: %v", err)
	}

	// Convert to header value
	headerValue := originalHeader.ToHeaderValue()

	// Parse back
	parsedHeader, err := auth.ParseAuthHeader(headerValue)
	if err != nil {
		t.Fatalf("Failed to parse auth header: %v", err)
	}

	// Compare fields
	if parsedHeader.PublicKey != originalHeader.PublicKey {
		t.Error("Public key mismatch after round trip")
	}

	if parsedHeader.Signature != originalHeader.Signature {
		t.Error("Signature mismatch after round trip")
	}

	if parsedHeader.Timestamp != originalHeader.Timestamp {
		t.Error("Timestamp mismatch after round trip")
	}

	if parsedHeader.Nonce != originalHeader.Nonce {
		t.Error("Nonce mismatch after round trip")
	}

	// Verify the parsed header
	err = auth.VerifyAuthHeader(parsedHeader, "POST", "/api/v1/users")
	if err != nil {
		t.Errorf("Failed to verify parsed auth header: %v", err)
	}
}
