package encryption

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/box"
)

// EncryptionKeyPair represents a NaCl encryption key pair
type EncryptionKeyPair struct {
	PublicKey  [32]byte
	PrivateKey [32]byte
}

// EncryptedMessage represents an encrypted message with metadata
type EncryptedMessage struct {
	Nonce      [24]byte `json:"nonce"`
	Ciphertext []byte   `json:"ciphertext"`
	PublicKey  [32]byte `json:"sender_public_key"`
}

// KeyStore interface for managing encryption keys
type KeyStore interface {
	StorePublicKey(address string, publicKey [32]byte) error
	GetPublicKey(address string) ([32]byte, error)
	HasPublicKey(address string) bool
}

// MemoryKeyStore is an in-memory implementation of KeyStore
type MemoryKeyStore struct {
	keys map[string][32]byte
}

// NewMemoryKeyStore creates a new in-memory key store
func NewMemoryKeyStore() *MemoryKeyStore {
	return &MemoryKeyStore{
		keys: make(map[string][32]byte),
	}
}

// StorePublicKey stores a public key for an address
func (m *MemoryKeyStore) StorePublicKey(address string, publicKey [32]byte) error {
	m.keys[address] = publicKey
	return nil
}

// GetPublicKey retrieves a public key for an address
func (m *MemoryKeyStore) GetPublicKey(address string) ([32]byte, error) {
	key, exists := m.keys[address]
	if !exists {
		return [32]byte{}, fmt.Errorf("public key not found for address: %s", address)
	}
	return key, nil
}

// HasPublicKey checks if a public key exists for an address
func (m *MemoryKeyStore) HasPublicKey(address string) bool {
	_, exists := m.keys[address]
	return exists
}

// GenerateEncryptionKeyPair generates a new NaCl encryption key pair
func GenerateEncryptionKeyPair() (*EncryptionKeyPair, error) {
	publicKey, privateKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate encryption key pair: %w", err)
	}

	return &EncryptionKeyPair{
		PublicKey:  *publicKey,
		PrivateKey: *privateKey,
	}, nil
}

// PublicKeyBase64 returns the base64-encoded public key
func (kp *EncryptionKeyPair) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(kp.PublicKey[:])
}

// PrivateKeyBase64 returns the base64-encoded private key
func (kp *EncryptionKeyPair) PrivateKeyBase64() string {
	return base64.StdEncoding.EncodeToString(kp.PrivateKey[:])
}

// Encrypt encrypts a message for a recipient
func (kp *EncryptionKeyPair) Encrypt(message []byte, recipientPublicKey [32]byte) (*EncryptedMessage, error) {
	// Generate a random nonce
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the message
	ciphertext := box.Seal(nil, message, &nonce, &recipientPublicKey, &kp.PrivateKey)

	return &EncryptedMessage{
		Nonce:      nonce,
		Ciphertext: ciphertext,
		PublicKey:  kp.PublicKey,
	}, nil
}

// Decrypt decrypts a message from a sender
func (kp *EncryptionKeyPair) Decrypt(encMsg *EncryptedMessage) ([]byte, error) {
	// Decrypt the message
	plaintext, ok := box.Open(nil, encMsg.Ciphertext, &encMsg.Nonce, &encMsg.PublicKey, &kp.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("failed to decrypt message")
	}

	return plaintext, nil
}

// EncryptionManager manages encryption operations and key storage
type EncryptionManager struct {
	keyPair  *EncryptionKeyPair
	keyStore KeyStore
}

// NewEncryptionManager creates a new encryption manager
func NewEncryptionManager(keyPair *EncryptionKeyPair, keyStore KeyStore) *EncryptionManager {
	return &EncryptionManager{
		keyPair:  keyPair,
		keyStore: keyStore,
	}
}

// EncryptForRecipient encrypts a message for a specific recipient
func (em *EncryptionManager) EncryptForRecipient(message []byte, recipientAddress string) (*EncryptedMessage, error) {
	// Get recipient's public key
	recipientPublicKey, err := em.keyStore.GetPublicKey(recipientAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipient public key: %w", err)
	}

	// Encrypt the message
	return em.keyPair.Encrypt(message, recipientPublicKey)
}

// DecryptMessage decrypts a message from a sender
func (em *EncryptionManager) DecryptMessage(encMsg *EncryptedMessage) ([]byte, error) {
	return em.keyPair.Decrypt(encMsg)
}

// CanEncryptFor checks if we can encrypt for a recipient
func (em *EncryptionManager) CanEncryptFor(recipientAddress string) bool {
	return em.keyStore.HasPublicKey(recipientAddress)
}

// GetPublicKey returns our public key
func (em *EncryptionManager) GetPublicKey() [32]byte {
	return em.keyPair.PublicKey
}

// RegisterPublicKey registers a public key for an address
func (em *EncryptionManager) RegisterPublicKey(address string, publicKeyBase64 string) error {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return fmt.Errorf("invalid public key format: %w", err)
	}

	if len(publicKeyBytes) != 32 {
		return fmt.Errorf("invalid public key length: expected 32 bytes, got %d", len(publicKeyBytes))
	}

	var publicKey [32]byte
	copy(publicKey[:], publicKeyBytes)

	return em.keyStore.StorePublicKey(address, publicKey)
}

// LoadEncryptionKeyPairFromBase64 loads a key pair from base64 strings
func LoadEncryptionKeyPairFromBase64(publicKeyB64, privateKeyB64 string) (*EncryptionKeyPair, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return nil, fmt.Errorf("invalid public key format: %w", err)
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return nil, fmt.Errorf("invalid private key format: %w", err)
	}

	if len(publicKeyBytes) != 32 || len(privateKeyBytes) != 32 {
		return nil, fmt.Errorf("invalid key length: expected 32 bytes each")
	}

	var publicKey, privateKey [32]byte
	copy(publicKey[:], publicKeyBytes)
	copy(privateKey[:], privateKeyBytes)

	return &EncryptionKeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

// EncryptionConfig holds encryption configuration
type EncryptionConfig struct {
	Enabled           bool
	KeyPair           *EncryptionKeyPair
	KeyStore          KeyStore
	FallbackOnFailure bool // If true, send unencrypted if encryption fails
}

// DefaultEncryptionConfig returns a default encryption configuration
func DefaultEncryptionConfig() *EncryptionConfig {
	return &EncryptionConfig{
		Enabled:           false,
		KeyStore:          NewMemoryKeyStore(),
		FallbackOnFailure: true,
	}
}
