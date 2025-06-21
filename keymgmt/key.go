package keymgmt

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// KeyPair represents an Ed25519 key pair
type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// GenerateKeyPair creates a new Ed25519 key pair
func GenerateKeyPair() (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// Sign signs a message with the private key
func (kp *KeyPair) Sign(message []byte) []byte {
	return ed25519.Sign(kp.PrivateKey, message)
}

// Verify verifies a signature with the public key
func (kp *KeyPair) Verify(message, signature []byte) bool {
	return ed25519.Verify(kp.PublicKey, message, signature)
}

// PublicKeyBase64 returns the public key as a base64 string
func (kp *KeyPair) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(kp.PublicKey)
}

// PrivateKeyHex returns the private key as a hex string
func (kp *KeyPair) PrivateKeyHex() string {
	return hex.EncodeToString(kp.PrivateKey)
}

// PublicKeyHex returns the public key as a hex string
func (kp *KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(kp.PublicKey)
}

// SavePrivateKeyToFile saves the private key to a file
func (kp *KeyPair) SavePrivateKeyToFile(filename string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write private key as hex to file
	privateKeyHex := kp.PrivateKeyHex()
	err := ioutil.WriteFile(filename, []byte(privateKeyHex), 0600)
	if err != nil {
		return fmt.Errorf("failed to write private key to file: %w", err)
	}

	return nil
}

// LoadPrivateKeyFromFile loads a private key from a file
func LoadPrivateKeyFromFile(filename string) (*KeyPair, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	privateKeyBytes, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key hex: %w", err)
	}

	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: expected %d, got %d", ed25519.PrivateKeySize, len(privateKeyBytes))
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// LoadPrivateKeyFromHex loads a private key from a hex string
func LoadPrivateKeyFromHex(hexKey string) (*KeyPair, error) {
	privateKeyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key hex: %w", err)
	}

	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: expected %d, got %d", ed25519.PrivateKeySize, len(privateKeyBytes))
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// LoadPublicKeyFromBase64 loads a public key from a base64 string
func LoadPublicKeyFromBase64(base64Key string) (ed25519.PublicKey, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key base64: %w", err)
	}

	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(publicKeyBytes))
	}

	return ed25519.PublicKey(publicKeyBytes), nil
}

// LoadPublicKeyFromHex loads a public key from a hex string
func LoadPublicKeyFromHex(hexKey string) (ed25519.PublicKey, error) {
	publicKeyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key hex: %w", err)
	}

	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(publicKeyBytes))
	}

	return ed25519.PublicKey(publicKeyBytes), nil
}
