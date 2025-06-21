package test

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"

	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
)

func TestGenerateKeyPair(t *testing.T) {
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if keyPair == nil {
		t.Fatal("Key pair is nil")
	}

	if len(keyPair.PrivateKey) != ed25519.PrivateKeySize {
		t.Errorf("Invalid private key size: expected %d, got %d", ed25519.PrivateKeySize, len(keyPair.PrivateKey))
	}

	if len(keyPair.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("Invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(keyPair.PublicKey))
	}
}

func TestKeyPairSigning(t *testing.T) {
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	message := []byte("test message")
	signature := keyPair.Sign(message)

	if len(signature) == 0 {
		t.Fatal("Signature is empty")
	}

	// Verify with the same key pair
	if !keyPair.Verify(message, signature) {
		t.Error("Failed to verify signature with same key pair")
	}

	// Verify with different message should fail
	if keyPair.Verify([]byte("different message"), signature) {
		t.Error("Signature verification should fail with different message")
	}
}

func TestKeyPairEncodingDecoding(t *testing.T) {
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Test hex encoding/decoding
	privateKeyHex := keyPair.PrivateKeyHex()
	publicKeyHex := keyPair.PublicKeyHex()

	if privateKeyHex == "" {
		t.Error("Private key hex is empty")
	}

	if publicKeyHex == "" {
		t.Error("Public key hex is empty")
	}

	// Test loading from hex
	loadedKeyPair, err := keymgmt.LoadPrivateKeyFromHex(privateKeyHex)
	if err != nil {
		t.Fatalf("Failed to load key pair from hex: %v", err)
	}

	if loadedKeyPair.PrivateKeyHex() != privateKeyHex {
		t.Error("Loaded private key hex doesn't match original")
	}

	if loadedKeyPair.PublicKeyHex() != publicKeyHex {
		t.Error("Loaded public key hex doesn't match original")
	}

	// Test base64 encoding
	publicKeyBase64 := keyPair.PublicKeyBase64()
	if publicKeyBase64 == "" {
		t.Error("Public key base64 is empty")
	}

	loadedPublicKey, err := keymgmt.LoadPublicKeyFromBase64(publicKeyBase64)
	if err != nil {
		t.Fatalf("Failed to load public key from base64: %v", err)
	}

	if string(loadedPublicKey) != string(keyPair.PublicKey) {
		t.Error("Loaded public key doesn't match original")
	}
}

func TestSaveLoadPrivateKeyFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	keyFile := filepath.Join(tempDir, "test_key.txt")

	// Generate key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Save to file
	if err := keyPair.SavePrivateKeyToFile(keyFile); err != nil {
		t.Fatalf("Failed to save private key to file: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		t.Fatal("Key file was not created")
	}

	// Load from file
	loadedKeyPair, err := keymgmt.LoadPrivateKeyFromFile(keyFile)
	if err != nil {
		t.Fatalf("Failed to load private key from file: %v", err)
	}

	// Compare keys
	if loadedKeyPair.PrivateKeyHex() != keyPair.PrivateKeyHex() {
		t.Error("Loaded private key doesn't match original")
	}

	if loadedKeyPair.PublicKeyHex() != keyPair.PublicKeyHex() {
		t.Error("Loaded public key doesn't match original")
	}

	// Test signing with loaded key
	message := []byte("test message")
	originalSignature := keyPair.Sign(message)
	loadedSignature := loadedKeyPair.Sign(message)

	if !keyPair.Verify(message, loadedSignature) {
		t.Error("Original key pair cannot verify signature from loaded key pair")
	}

	if !loadedKeyPair.Verify(message, originalSignature) {
		t.Error("Loaded key pair cannot verify signature from original key pair")
	}
}

func TestInvalidKeyLoading(t *testing.T) {
	// Test invalid hex
	_, err := keymgmt.LoadPrivateKeyFromHex("invalid_hex")
	if err == nil {
		t.Error("Expected error when loading invalid hex")
	}

	// Test invalid key size
	_, err = keymgmt.LoadPrivateKeyFromHex("deadbeef")
	if err == nil {
		t.Error("Expected error when loading key with invalid size")
	}

	// Test invalid base64
	_, err = keymgmt.LoadPublicKeyFromBase64("invalid_base64!")
	if err == nil {
		t.Error("Expected error when loading invalid base64")
	}

	// Test non-existent file
	_, err = keymgmt.LoadPrivateKeyFromFile("non_existent_file.txt")
	if err == nil {
		t.Error("Expected error when loading from non-existent file")
	}
}

func TestKeyPairConsistency(t *testing.T) {
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// The public key should be derivable from the private key
	expectedPublicKey := keyPair.PrivateKey.Public().(ed25519.PublicKey)
	
	if string(expectedPublicKey) != string(keyPair.PublicKey) {
		t.Error("Public key is not consistent with private key")
	}
}
