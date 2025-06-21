package message

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
	"github.com/emsg-protocol/emsg-client-sdk/utils"
)

// Message represents an EMSG message structure
type Message struct {
	From      string   `json:"from"`
	To        []string `json:"to"`
	CC        []string `json:"cc,omitempty"`
	Subject   string   `json:"subject,omitempty"`
	Body      string   `json:"body"`
	GroupID   string   `json:"group_id,omitempty"`
	Timestamp int64    `json:"timestamp"`
	MessageID string   `json:"message_id,omitempty"`
	Signature string   `json:"signature,omitempty"`
}

// MessageBuilder helps construct EMSG messages
type MessageBuilder struct {
	message *Message
}

// NewMessageBuilder creates a new message builder
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		message: &Message{
			Timestamp: time.Now().Unix(),
		},
	}
}

// From sets the sender address
func (mb *MessageBuilder) From(address string) *MessageBuilder {
	mb.message.From = address
	return mb
}

// To sets the recipient addresses
func (mb *MessageBuilder) To(addresses ...string) *MessageBuilder {
	mb.message.To = addresses
	return mb
}

// CC sets the carbon copy addresses
func (mb *MessageBuilder) CC(addresses ...string) *MessageBuilder {
	mb.message.CC = addresses
	return mb
}

// Subject sets the message subject
func (mb *MessageBuilder) Subject(subject string) *MessageBuilder {
	mb.message.Subject = subject
	return mb
}

// Body sets the message body
func (mb *MessageBuilder) Body(body string) *MessageBuilder {
	mb.message.Body = body
	return mb
}

// GroupID sets the group ID for group messages
func (mb *MessageBuilder) GroupID(groupID string) *MessageBuilder {
	mb.message.GroupID = groupID
	return mb
}

// MessageID sets a custom message ID
func (mb *MessageBuilder) MessageID(messageID string) *MessageBuilder {
	mb.message.MessageID = messageID
	return mb
}

// Build validates and returns the constructed message
func (mb *MessageBuilder) Build() (*Message, error) {
	if err := mb.validate(); err != nil {
		return nil, err
	}

	// Generate message ID if not provided
	if mb.message.MessageID == "" {
		mb.message.MessageID = mb.generateMessageID()
	}

	// Create a copy to avoid mutations
	msg := *mb.message
	return &msg, nil
}

// validate validates the message structure
func (mb *MessageBuilder) validate() error {
	if mb.message.From == "" {
		return fmt.Errorf("from address is required")
	}

	if !utils.IsValidEMSGAddress(mb.message.From) {
		return fmt.Errorf("invalid from address: %s", mb.message.From)
	}

	if len(mb.message.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	// Validate all recipient addresses
	allRecipients := append(mb.message.To, mb.message.CC...)
	if err := utils.ValidateEMSGAddressList(allRecipients); err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}

	if mb.message.Body == "" {
		return fmt.Errorf("message body is required")
	}

	return nil
}

// generateMessageID generates a unique message ID
func (mb *MessageBuilder) generateMessageID() string {
	// Create a hash based on message content and timestamp
	content := fmt.Sprintf("%s:%s:%s:%d",
		mb.message.From,
		strings.Join(mb.message.To, ","),
		mb.message.Body,
		mb.message.Timestamp,
	)

	hash := sha256.Sum256([]byte(content))
	return base64.URLEncoding.EncodeToString(hash[:16]) // Use first 16 bytes
}

// Sign signs the message with the provided key pair
func (msg *Message) Sign(keyPair *keymgmt.KeyPair) error {
	// Validate that the key pair matches the from address
	// This is a simplified check - in practice, you might want more sophisticated validation

	// Create the signing payload
	payload, err := msg.getSigningPayload()
	if err != nil {
		return fmt.Errorf("failed to create signing payload: %w", err)
	}

	// Sign the payload
	signature := keyPair.Sign(payload)
	msg.Signature = base64.StdEncoding.EncodeToString(signature)

	return nil
}

// getSigningPayload creates the payload for message signing
func (msg *Message) getSigningPayload() ([]byte, error) {
	// Create a copy without signature for signing
	signingMsg := *msg
	signingMsg.Signature = ""

	// Serialize to JSON for consistent signing
	payload, err := json.Marshal(signingMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message for signing: %w", err)
	}

	return payload, nil
}

// Verify verifies the message signature
func (msg *Message) Verify(publicKey string) error {
	if msg.Signature == "" {
		return fmt.Errorf("message is not signed")
	}

	// Load the public key
	pubKey, err := keymgmt.LoadPublicKeyFromBase64(publicKey)
	if err != nil {
		return fmt.Errorf("failed to load public key: %w", err)
	}

	// Get the signing payload
	payload, err := msg.getSigningPayload()
	if err != nil {
		return fmt.Errorf("failed to create signing payload: %w", err)
	}

	// Decode the signature
	signature, err := base64.StdEncoding.DecodeString(msg.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify the signature
	if !ed25519.Verify(pubKey, payload, signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// ToJSON serializes the message to JSON
func (msg *Message) ToJSON() ([]byte, error) {
	return json.Marshal(msg)
}

// FromJSON deserializes a message from JSON
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	return &msg, nil
}

// Validate validates the message structure
func (msg *Message) Validate() error {
	if msg.From == "" {
		return fmt.Errorf("from address is required")
	}

	if !utils.IsValidEMSGAddress(msg.From) {
		return fmt.Errorf("invalid from address: %s", msg.From)
	}

	if len(msg.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	// Validate all recipient addresses
	allRecipients := append(msg.To, msg.CC...)
	if err := utils.ValidateEMSGAddressList(allRecipients); err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}

	if msg.Body == "" {
		return fmt.Errorf("message body is required")
	}

	if msg.Timestamp <= 0 {
		return fmt.Errorf("invalid timestamp")
	}

	return nil
}

// GetRecipients returns all recipients (To + CC)
func (msg *Message) GetRecipients() []string {
	recipients := make([]string, 0, len(msg.To)+len(msg.CC))
	recipients = append(recipients, msg.To...)
	recipients = append(recipients, msg.CC...)
	return recipients
}

// IsSigned returns true if the message has a signature
func (msg *Message) IsSigned() bool {
	return msg.Signature != ""
}

// Clone creates a deep copy of the message
func (msg *Message) Clone() *Message {
	clone := *msg
	clone.To = make([]string, len(msg.To))
	copy(clone.To, msg.To)

	if len(msg.CC) > 0 {
		clone.CC = make([]string, len(msg.CC))
		copy(clone.CC, msg.CC)
	}

	return &clone
}
