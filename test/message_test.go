package test

import (
	"testing"

	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
	"github.com/emsg-protocol/emsg-client-sdk/message"
)

func TestMessageBuilder(t *testing.T) {
	builder := message.NewMessageBuilder()

	msg, err := builder.
		From("alice#example.com").
		To("bob#test.org", "charlie#example.net").
		CC("dave#example.org").
		Subject("Test Message").
		Body("Hello, world!").
		GroupID("group123").
		MessageID("msg123").
		Build()

	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}

	if msg.From != "alice#example.com" {
		t.Errorf("Expected from alice#example.com, got %s", msg.From)
	}

	if len(msg.To) != 2 {
		t.Errorf("Expected 2 recipients, got %d", len(msg.To))
	}

	if msg.To[0] != "bob#test.org" {
		t.Errorf("Expected first recipient bob#test.org, got %s", msg.To[0])
	}

	if msg.To[1] != "charlie#example.net" {
		t.Errorf("Expected second recipient charlie#example.net, got %s", msg.To[1])
	}

	if len(msg.CC) != 1 {
		t.Errorf("Expected 1 CC recipient, got %d", len(msg.CC))
	}

	if msg.CC[0] != "dave#example.org" {
		t.Errorf("Expected CC recipient dave#example.org, got %s", msg.CC[0])
	}

	if msg.Subject != "Test Message" {
		t.Errorf("Expected subject 'Test Message', got %s", msg.Subject)
	}

	if msg.Body != "Hello, world!" {
		t.Errorf("Expected body 'Hello, world!', got %s", msg.Body)
	}

	if msg.GroupID != "group123" {
		t.Errorf("Expected group ID 'group123', got %s", msg.GroupID)
	}

	if msg.MessageID != "msg123" {
		t.Errorf("Expected message ID 'msg123', got %s", msg.MessageID)
	}

	if msg.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}
}

func TestMessageBuilderValidation(t *testing.T) {
	// Test missing from address
	builder1 := message.NewMessageBuilder()
	_, err := builder1.To("bob#test.org").Body("Hello").Build()
	if err == nil {
		t.Error("Expected error for missing from address")
	}

	// Test invalid from address
	builder2 := message.NewMessageBuilder()
	_, err = builder2.From("invalid@format").To("bob#test.org").Body("Hello").Build()
	if err == nil {
		t.Error("Expected error for invalid from address")
	}

	// Test missing recipients
	builder3 := message.NewMessageBuilder()
	_, err = builder3.From("alice#example.com").Body("Hello").Build()
	if err == nil {
		t.Error("Expected error for missing recipients")
	}

	// Test invalid recipient
	builder4 := message.NewMessageBuilder()
	_, err = builder4.From("alice#example.com").To("invalid@format").Body("Hello").Build()
	if err == nil {
		t.Error("Expected error for invalid recipient")
	}

	// Test missing body
	builder5 := message.NewMessageBuilder()
	_, err = builder5.From("alice#example.com").To("bob#test.org").Build()
	if err == nil {
		t.Error("Expected error for missing body")
	}
}

func TestMessageBuilderAutoMessageID(t *testing.T) {
	builder1 := message.NewMessageBuilder()

	msg, err := builder1.
		From("alice#example.com").
		To("bob#test.org").
		Body("Hello").
		Build()

	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}

	if msg.MessageID == "" {
		t.Error("Message ID should be auto-generated when not provided")
	}

	// Build another message with same content but different builder
	builder2 := message.NewMessageBuilder()
	msg2, err := builder2.
		From("alice#example.com").
		To("bob#test.org").
		Body("Hello Different"). // Make content different to ensure different ID
		Build()

	if err != nil {
		t.Fatalf("Failed to build second message: %v", err)
	}

	// Message IDs should be different due to different content
	if msg.MessageID == msg2.MessageID {
		t.Error("Auto-generated message IDs should be different")
	}
}

func TestMessageSigning(t *testing.T) {
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	builder := message.NewMessageBuilder()
	msg, err := builder.
		From("alice#example.com").
		To("bob#test.org").
		Body("Hello").
		Build()

	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}

	// Message should not be signed initially
	if msg.IsSigned() {
		t.Error("Message should not be signed initially")
	}

	// Sign the message
	err = msg.Sign(keyPair)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Message should now be signed
	if !msg.IsSigned() {
		t.Error("Message should be signed after signing")
	}

	if msg.Signature == "" {
		t.Error("Signature should not be empty")
	}
}

func TestMessageVerification(t *testing.T) {
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	builder := message.NewMessageBuilder()
	msg, err := builder.
		From("alice#example.com").
		To("bob#test.org").
		Body("Hello").
		Build()

	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}

	// Sign the message
	err = msg.Sign(keyPair)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Verify with correct public key
	err = msg.Verify(keyPair.PublicKeyBase64())
	if err != nil {
		t.Errorf("Failed to verify message with correct key: %v", err)
	}

	// Generate different key pair
	otherKeyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate other key pair: %v", err)
	}

	// Verify with wrong public key should fail
	err = msg.Verify(otherKeyPair.PublicKeyBase64())
	if err == nil {
		t.Error("Expected verification to fail with wrong public key")
	}

	// Test verification of unsigned message
	unsignedMsg, _ := builder.From("alice#example.com").To("bob#test.org").Body("Hello").Build()
	err = unsignedMsg.Verify(keyPair.PublicKeyBase64())
	if err == nil {
		t.Error("Expected verification to fail for unsigned message")
	}
}

func TestMessageJSONSerialization(t *testing.T) {
	builder := message.NewMessageBuilder()
	originalMsg, err := builder.
		From("alice#example.com").
		To("bob#test.org", "charlie#example.net").
		CC("dave#example.org").
		Subject("Test Message").
		Body("Hello, world!").
		GroupID("group123").
		Build()

	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}

	// Serialize to JSON
	jsonData, err := originalMsg.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize message to JSON: %v", err)
	}

	// Deserialize from JSON
	deserializedMsg, err := message.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize message from JSON: %v", err)
	}

	// Compare fields
	if deserializedMsg.From != originalMsg.From {
		t.Errorf("From field mismatch: expected %s, got %s", originalMsg.From, deserializedMsg.From)
	}

	if len(deserializedMsg.To) != len(originalMsg.To) {
		t.Errorf("To field length mismatch: expected %d, got %d", len(originalMsg.To), len(deserializedMsg.To))
	}

	for i, addr := range deserializedMsg.To {
		if addr != originalMsg.To[i] {
			t.Errorf("To field mismatch at index %d: expected %s, got %s", i, originalMsg.To[i], addr)
		}
	}

	if deserializedMsg.Subject != originalMsg.Subject {
		t.Errorf("Subject field mismatch: expected %s, got %s", originalMsg.Subject, deserializedMsg.Subject)
	}

	if deserializedMsg.Body != originalMsg.Body {
		t.Errorf("Body field mismatch: expected %s, got %s", originalMsg.Body, deserializedMsg.Body)
	}

	if deserializedMsg.GroupID != originalMsg.GroupID {
		t.Errorf("GroupID field mismatch: expected %s, got %s", originalMsg.GroupID, deserializedMsg.GroupID)
	}

	if deserializedMsg.Timestamp != originalMsg.Timestamp {
		t.Errorf("Timestamp field mismatch: expected %d, got %d", originalMsg.Timestamp, deserializedMsg.Timestamp)
	}
}

func TestMessageValidation(t *testing.T) {
	// Test valid message
	validMsg := &message.Message{
		From:      "alice#example.com",
		To:        []string{"bob#test.org"},
		Body:      "Hello",
		Timestamp: 1234567890,
	}

	err := validMsg.Validate()
	if err != nil {
		t.Errorf("Valid message should pass validation: %v", err)
	}

	// Test invalid messages
	invalidMessages := []*message.Message{
		{
			// Missing from
			To:        []string{"bob#test.org"},
			Body:      "Hello",
			Timestamp: 1234567890,
		},
		{
			// Invalid from
			From:      "invalid@format",
			To:        []string{"bob#test.org"},
			Body:      "Hello",
			Timestamp: 1234567890,
		},
		{
			// Missing recipients
			From:      "alice#example.com",
			Body:      "Hello",
			Timestamp: 1234567890,
		},
		{
			// Invalid recipient
			From:      "alice#example.com",
			To:        []string{"invalid@format"},
			Body:      "Hello",
			Timestamp: 1234567890,
		},
		{
			// Missing body
			From:      "alice#example.com",
			To:        []string{"bob#test.org"},
			Timestamp: 1234567890,
		},
		{
			// Invalid timestamp
			From: "alice#example.com",
			To:   []string{"bob#test.org"},
			Body: "Hello",
		},
	}

	for i, msg := range invalidMessages {
		err := msg.Validate()
		if err == nil {
			t.Errorf("Invalid message %d should fail validation", i)
		}
	}
}

func TestMessageGetRecipients(t *testing.T) {
	msg := &message.Message{
		To: []string{"alice#example.com", "bob#test.org"},
		CC: []string{"charlie#example.net", "dave#example.org"},
	}

	recipients := msg.GetRecipients()
	expected := []string{"alice#example.com", "bob#test.org", "charlie#example.net", "dave#example.org"}

	if len(recipients) != len(expected) {
		t.Errorf("Expected %d recipients, got %d", len(expected), len(recipients))
	}

	for i, recipient := range recipients {
		if recipient != expected[i] {
			t.Errorf("Expected recipient %s at index %d, got %s", expected[i], i, recipient)
		}
	}
}

func TestMessageClone(t *testing.T) {
	original := &message.Message{
		From:      "alice#example.com",
		To:        []string{"bob#test.org", "charlie#example.net"},
		CC:        []string{"dave#example.org"},
		Subject:   "Test",
		Body:      "Hello",
		GroupID:   "group123",
		Timestamp: 1234567890,
		MessageID: "msg123",
		Signature: "signature123",
	}

	clone := original.Clone()

	// Verify all fields are copied
	if clone.From != original.From {
		t.Error("From field not cloned correctly")
	}

	if clone.Subject != original.Subject {
		t.Error("Subject field not cloned correctly")
	}

	if clone.Body != original.Body {
		t.Error("Body field not cloned correctly")
	}

	// Verify slices are deep copied
	if &clone.To == &original.To {
		t.Error("To slice should be deep copied")
	}

	if &clone.CC == &original.CC {
		t.Error("CC slice should be deep copied")
	}

	// Modify clone and verify original is unchanged
	clone.To[0] = "modified#example.com"
	if original.To[0] == "modified#example.com" {
		t.Error("Modifying clone should not affect original")
	}
}
