package test

import (
	"encoding/base64"
	"testing"

	"github.com/emsg-protocol/emsg-client-sdk/encryption"
)

func TestGenerateEncryptionKeyPair(t *testing.T) {
	keyPair, err := encryption.GenerateEncryptionKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate encryption key pair: %v", err)
	}

	if len(keyPair.PublicKey) != 32 {
		t.Errorf("Expected public key length 32, got %d", len(keyPair.PublicKey))
	}

	if len(keyPair.PrivateKey) != 32 {
		t.Errorf("Expected private key length 32, got %d", len(keyPair.PrivateKey))
	}

	// Test base64 encoding
	pubKeyB64 := keyPair.PublicKeyBase64()
	if len(pubKeyB64) == 0 {
		t.Error("Public key base64 encoding is empty")
	}

	privKeyB64 := keyPair.PrivateKeyBase64()
	if len(privKeyB64) == 0 {
		t.Error("Private key base64 encoding is empty")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	// Generate two key pairs (sender and recipient)
	senderKeyPair, err := encryption.GenerateEncryptionKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key pair: %v", err)
	}

	recipientKeyPair, err := encryption.GenerateEncryptionKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key pair: %v", err)
	}

	// Test message
	originalMessage := "This is a secret message for testing encryption!"

	// Encrypt message from sender to recipient
	encryptedMsg, err := senderKeyPair.Encrypt([]byte(originalMessage), recipientKeyPair.PublicKey)
	if err != nil {
		t.Fatalf("Failed to encrypt message: %v", err)
	}

	// Verify encrypted message structure
	if len(encryptedMsg.Nonce) != 24 {
		t.Errorf("Expected nonce length 24, got %d", len(encryptedMsg.Nonce))
	}

	if len(encryptedMsg.Ciphertext) == 0 {
		t.Error("Ciphertext is empty")
	}

	if encryptedMsg.PublicKey != senderKeyPair.PublicKey {
		t.Error("Sender public key mismatch in encrypted message")
	}

	// Decrypt message at recipient
	decryptedBytes, err := recipientKeyPair.Decrypt(encryptedMsg)
	if err != nil {
		t.Fatalf("Failed to decrypt message: %v", err)
	}

	decryptedMessage := string(decryptedBytes)
	if decryptedMessage != originalMessage {
		t.Errorf("Decrypted message mismatch. Expected: %s, Got: %s", originalMessage, decryptedMessage)
	}
}

func TestEncryptionWithWrongKey(t *testing.T) {
	// Generate three key pairs
	senderKeyPair, _ := encryption.GenerateEncryptionKeyPair()
	recipientKeyPair, _ := encryption.GenerateEncryptionKeyPair()
	wrongKeyPair, _ := encryption.GenerateEncryptionKeyPair()

	originalMessage := "Secret message"

	// Encrypt with sender to recipient
	encryptedMsg, err := senderKeyPair.Encrypt([]byte(originalMessage), recipientKeyPair.PublicKey)
	if err != nil {
		t.Fatalf("Failed to encrypt message: %v", err)
	}

	// Try to decrypt with wrong key - should fail
	_, err = wrongKeyPair.Decrypt(encryptedMsg)
	if err == nil {
		t.Error("Expected decryption to fail with wrong key, but it succeeded")
	}
}

func TestMemoryKeyStore(t *testing.T) {
	keyStore := encryption.NewMemoryKeyStore()

	// Generate a key pair for testing
	keyPair, _ := encryption.GenerateEncryptionKeyPair()
	address := "alice#example.com"

	// Test storing and retrieving
	err := keyStore.StorePublicKey(address, keyPair.PublicKey)
	if err != nil {
		t.Fatalf("Failed to store public key: %v", err)
	}

	// Check if key exists
	if !keyStore.HasPublicKey(address) {
		t.Error("Key store should have the public key")
	}

	// Retrieve the key
	retrievedKey, err := keyStore.GetPublicKey(address)
	if err != nil {
		t.Fatalf("Failed to retrieve public key: %v", err)
	}

	if retrievedKey != keyPair.PublicKey {
		t.Error("Retrieved public key doesn't match stored key")
	}

	// Test non-existent key
	if keyStore.HasPublicKey("nonexistent#example.com") {
		t.Error("Key store should not have non-existent key")
	}

	_, err = keyStore.GetPublicKey("nonexistent#example.com")
	if err == nil {
		t.Error("Expected error when retrieving non-existent key")
	}
}

func TestEncryptionManager(t *testing.T) {
	// Generate key pairs
	aliceKeyPair, _ := encryption.GenerateEncryptionKeyPair()
	bobKeyPair, _ := encryption.GenerateEncryptionKeyPair()

	// Create key store and add Bob's public key
	keyStore := encryption.NewMemoryKeyStore()
	keyStore.StorePublicKey("bob#example.com", bobKeyPair.PublicKey)

	// Create encryption manager for Alice
	encManager := encryption.NewEncryptionManager(aliceKeyPair, keyStore)

	// Test encryption for Bob
	originalMessage := "Hello Bob, this is Alice!"
	encryptedMsg, err := encManager.EncryptForRecipient([]byte(originalMessage), "bob#example.com")
	if err != nil {
		t.Fatalf("Failed to encrypt for recipient: %v", err)
	}

	// Create encryption manager for Bob to decrypt
	bobEncManager := encryption.NewEncryptionManager(bobKeyPair, encryption.NewMemoryKeyStore())

	// Decrypt the message
	decryptedBytes, err := bobEncManager.DecryptMessage(encryptedMsg)
	if err != nil {
		t.Fatalf("Failed to decrypt message: %v", err)
	}

	if string(decryptedBytes) != originalMessage {
		t.Errorf("Decrypted message mismatch. Expected: %s, Got: %s", originalMessage, string(decryptedBytes))
	}

	// Test encryption for unknown recipient
	_, err = encManager.EncryptForRecipient([]byte("test"), "unknown#example.com")
	if err == nil {
		t.Error("Expected error when encrypting for unknown recipient")
	}

	// Test CanEncryptFor
	if !encManager.CanEncryptFor("bob#example.com") {
		t.Error("Should be able to encrypt for Bob")
	}

	if encManager.CanEncryptFor("unknown#example.com") {
		t.Error("Should not be able to encrypt for unknown recipient")
	}
}

func TestLoadEncryptionKeyPairFromBase64(t *testing.T) {
	// Generate original key pair
	originalKeyPair, _ := encryption.GenerateEncryptionKeyPair()

	// Get base64 representations
	pubKeyB64 := originalKeyPair.PublicKeyBase64()
	privKeyB64 := originalKeyPair.PrivateKeyBase64()

	// Load key pair from base64
	loadedKeyPair, err := encryption.LoadEncryptionKeyPairFromBase64(pubKeyB64, privKeyB64)
	if err != nil {
		t.Fatalf("Failed to load key pair from base64: %v", err)
	}

	// Verify keys match
	if loadedKeyPair.PublicKey != originalKeyPair.PublicKey {
		t.Error("Loaded public key doesn't match original")
	}

	if loadedKeyPair.PrivateKey != originalKeyPair.PrivateKey {
		t.Error("Loaded private key doesn't match original")
	}

	// Test with invalid base64
	_, err = encryption.LoadEncryptionKeyPairFromBase64("invalid", privKeyB64)
	if err == nil {
		t.Error("Expected error with invalid public key base64")
	}

	_, err = encryption.LoadEncryptionKeyPairFromBase64(pubKeyB64, "invalid")
	if err == nil {
		t.Error("Expected error with invalid private key base64")
	}

	// Test with wrong length
	shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err = encryption.LoadEncryptionKeyPairFromBase64(shortKey, privKeyB64)
	if err == nil {
		t.Error("Expected error with short public key")
	}
}

func TestEncryptionConfig(t *testing.T) {
	config := encryption.DefaultEncryptionConfig()

	if config.Enabled {
		t.Error("Default encryption config should be disabled")
	}

	if config.KeyStore == nil {
		t.Error("Default encryption config should have a key store")
	}

	if !config.FallbackOnFailure {
		t.Error("Default encryption config should have fallback enabled")
	}
}

func TestRegisterPublicKey(t *testing.T) {
	keyPair, _ := encryption.GenerateEncryptionKeyPair()
	keyStore := encryption.NewMemoryKeyStore()
	encManager := encryption.NewEncryptionManager(keyPair, keyStore)

	// Generate another key pair for testing
	testKeyPair, _ := encryption.GenerateEncryptionKeyPair()
	testAddress := "test#example.com"

	// Register the public key
	err := encManager.RegisterPublicKey(testAddress, testKeyPair.PublicKeyBase64())
	if err != nil {
		t.Fatalf("Failed to register public key: %v", err)
	}

	// Verify it was stored
	if !encManager.CanEncryptFor(testAddress) {
		t.Error("Should be able to encrypt for registered address")
	}

	// Test with invalid base64
	err = encManager.RegisterPublicKey("invalid#example.com", "invalid-base64")
	if err == nil {
		t.Error("Expected error with invalid base64")
	}

	// Test with wrong length
	shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
	err = encManager.RegisterPublicKey("short#example.com", shortKey)
	if err == nil {
		t.Error("Expected error with short key")
	}
}
