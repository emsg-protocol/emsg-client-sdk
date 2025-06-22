package test

import (
	"testing"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
	"github.com/emsg-protocol/emsg-client-sdk/message"
)

// TestSystemMessageBuilder tests the system message builder functionality
func TestSystemMessageBuilder(t *testing.T) {
	builder := message.NewSystemMessageBuilder()

	systemMsg, err := builder.
		Type(message.SystemJoined).
		Actor("user#example.com").
		Target("group#example.com").
		GroupID("test-group").
		Metadata("timestamp", time.Now().Unix()).
		Metadata("reason", "invited").
		Build("system#example.com", []string{"group#example.com"})

	if err != nil {
		t.Fatalf("Failed to build system message: %v", err)
	}

	// Test message properties
	if systemMsg.From != "system#example.com" {
		t.Errorf("Expected from 'system#example.com', got '%s'", systemMsg.From)
	}

	if len(systemMsg.To) != 1 || systemMsg.To[0] != "group#example.com" {
		t.Errorf("Expected to ['group#example.com'], got %v", systemMsg.To)
	}

	if systemMsg.Type != message.SystemJoined {
		t.Errorf("Expected type '%s', got '%s'", message.SystemJoined, systemMsg.Type)
	}

	// Test that it's recognized as a system message
	if !systemMsg.IsSystemMessage() {
		t.Error("Message should be recognized as system message")
	}

	// Test parsing system message data
	parsedSystemMsg, err := systemMsg.GetSystemMessage()
	if err != nil {
		t.Fatalf("Failed to parse system message: %v", err)
	}

	if parsedSystemMsg.Type != message.SystemJoined {
		t.Errorf("Expected system message type '%s', got '%s'", message.SystemJoined, parsedSystemMsg.Type)
	}

	if parsedSystemMsg.Actor != "user#example.com" {
		t.Errorf("Expected actor 'user#example.com', got '%s'", parsedSystemMsg.Actor)
	}

	if parsedSystemMsg.Target != "group#example.com" {
		t.Errorf("Expected target 'group#example.com', got '%s'", parsedSystemMsg.Target)
	}

	if parsedSystemMsg.GroupID != "test-group" {
		t.Errorf("Expected group ID 'test-group', got '%s'", parsedSystemMsg.GroupID)
	}

	// Test metadata
	if reason, ok := parsedSystemMsg.Metadata["reason"]; !ok || reason != "invited" {
		t.Errorf("Expected metadata reason 'invited', got %v", reason)
	}
}

// TestSystemMessageValidation tests system message validation
func TestSystemMessageValidation(t *testing.T) {
	// Test missing type
	builder := message.NewSystemMessageBuilder()
	_, err := builder.Build("system#example.com", []string{"group#example.com"})
	if err == nil {
		t.Error("Expected error for missing system message type")
	}

	// Test valid system message
	systemMsg, err := builder.
		Type(message.SystemLeft).
		Actor("user#example.com").
		Build("system#example.com", []string{"group#example.com"})

	if err != nil {
		t.Fatalf("Failed to build valid system message: %v", err)
	}

	// Test validation
	if err := systemMsg.Validate(); err != nil {
		t.Errorf("Valid system message failed validation: %v", err)
	}
}

// TestSystemMessageHelpers tests the helper functions for common system messages
func TestSystemMessageHelpers(t *testing.T) {
	from := "system#example.com"
	to := []string{"group#example.com"}
	actor := "user#example.com"
	target := "other#example.com"
	groupID := "test-group"

	// Test user joined message
	joinedMsg, err := message.NewUserJoinedMessage(from, to, actor, groupID)
	if err != nil {
		t.Fatalf("Failed to create user joined message: %v", err)
	}

	if !joinedMsg.IsSystemMessage() {
		t.Error("Joined message should be a system message")
	}

	parsedJoined, err := joinedMsg.GetSystemMessage()
	if err != nil {
		t.Fatalf("Failed to parse joined message: %v", err)
	}

	if parsedJoined.Type != message.SystemJoined {
		t.Errorf("Expected type '%s', got '%s'", message.SystemJoined, parsedJoined.Type)
	}

	// Test user left message
	leftMsg, err := message.NewUserLeftMessage(from, to, actor, groupID)
	if err != nil {
		t.Fatalf("Failed to create user left message: %v", err)
	}

	parsedLeft, err := leftMsg.GetSystemMessage()
	if err != nil {
		t.Fatalf("Failed to parse left message: %v", err)
	}

	if parsedLeft.Type != message.SystemLeft {
		t.Errorf("Expected type '%s', got '%s'", message.SystemLeft, parsedLeft.Type)
	}

	// Test user removed message
	removedMsg, err := message.NewUserRemovedMessage(from, to, actor, target, groupID)
	if err != nil {
		t.Fatalf("Failed to create user removed message: %v", err)
	}

	parsedRemoved, err := removedMsg.GetSystemMessage()
	if err != nil {
		t.Fatalf("Failed to parse removed message: %v", err)
	}

	if parsedRemoved.Type != message.SystemRemoved {
		t.Errorf("Expected type '%s', got '%s'", message.SystemRemoved, parsedRemoved.Type)
	}

	if parsedRemoved.Target != target {
		t.Errorf("Expected target '%s', got '%s'", target, parsedRemoved.Target)
	}

	// Test admin changed message
	adminMsg, err := message.NewAdminChangedMessage(from, to, actor, target, groupID)
	if err != nil {
		t.Fatalf("Failed to create admin changed message: %v", err)
	}

	parsedAdmin, err := adminMsg.GetSystemMessage()
	if err != nil {
		t.Fatalf("Failed to parse admin message: %v", err)
	}

	if parsedAdmin.Type != message.SystemAdminChanged {
		t.Errorf("Expected type '%s', got '%s'", message.SystemAdminChanged, parsedAdmin.Type)
	}

	// Test group created message
	groupMsg, err := message.NewGroupCreatedMessage(from, to, actor, groupID)
	if err != nil {
		t.Fatalf("Failed to create group created message: %v", err)
	}

	parsedGroup, err := groupMsg.GetSystemMessage()
	if err != nil {
		t.Fatalf("Failed to parse group message: %v", err)
	}

	if parsedGroup.Type != message.SystemGroupCreated {
		t.Errorf("Expected type '%s', got '%s'", message.SystemGroupCreated, parsedGroup.Type)
	}
}

// TestSystemMessageSigning tests that system messages can be signed and verified
func TestSystemMessageSigning(t *testing.T) {
	// Generate key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create system message
	systemMsg, err := message.NewUserJoinedMessage(
		"system#example.com",
		[]string{"group#example.com"},
		"user#example.com",
		"test-group",
	)
	if err != nil {
		t.Fatalf("Failed to create system message: %v", err)
	}

	// Sign the message
	if err := systemMsg.Sign(keyPair); err != nil {
		t.Fatalf("Failed to sign system message: %v", err)
	}

	// Verify the signature
	if err := systemMsg.Verify(keyPair.PublicKeyBase64()); err != nil {
		t.Fatalf("Failed to verify system message: %v", err)
	}

	// Test that message is marked as signed
	if !systemMsg.IsSigned() {
		t.Error("System message should be marked as signed")
	}
}

// TestSystemMessageJSON tests JSON serialization of system messages
func TestSystemMessageJSON(t *testing.T) {
	// Create system message
	systemMsg, err := message.NewUserJoinedMessage(
		"system#example.com",
		[]string{"group#example.com"},
		"user#example.com",
		"test-group",
	)
	if err != nil {
		t.Fatalf("Failed to create system message: %v", err)
	}

	// Serialize to JSON
	jsonData, err := systemMsg.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize system message to JSON: %v", err)
	}

	// Deserialize from JSON
	deserializedMsg, err := message.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize system message from JSON: %v", err)
	}

	// Verify it's still a system message
	if !deserializedMsg.IsSystemMessage() {
		t.Error("Deserialized message should be a system message")
	}

	// Verify system message data
	parsedSystemMsg, err := deserializedMsg.GetSystemMessage()
	if err != nil {
		t.Fatalf("Failed to parse deserialized system message: %v", err)
	}

	if parsedSystemMsg.Type != message.SystemJoined {
		t.Errorf("Expected type '%s', got '%s'", message.SystemJoined, parsedSystemMsg.Type)
	}

	if parsedSystemMsg.Actor != "user#example.com" {
		t.Errorf("Expected actor 'user#example.com', got '%s'", parsedSystemMsg.Actor)
	}
}

// TestSystemMessageConstants tests that all system message constants are defined
func TestSystemMessageConstants(t *testing.T) {
	constants := []string{
		message.SystemJoined,
		message.SystemLeft,
		message.SystemRemoved,
		message.SystemAdminChanged,
		message.SystemGroupCreated,
	}

	expectedConstants := []string{
		"system:joined",
		"system:left",
		"system:removed",
		"system:admin_changed",
		"system:group_created",
	}

	if len(constants) != len(expectedConstants) {
		t.Fatalf("Expected %d constants, got %d", len(expectedConstants), len(constants))
	}

	for i, expected := range expectedConstants {
		if constants[i] != expected {
			t.Errorf("Expected constant '%s', got '%s'", expected, constants[i])
		}
	}
}

// TestNonSystemMessage tests that regular messages are not identified as system messages
func TestNonSystemMessage(t *testing.T) {
	builder := message.NewMessageBuilder()

	regularMsg, err := builder.
		From("user#example.com").
		To("recipient#example.com").
		Subject("Regular Message").
		Body("This is a regular message").
		Build()

	if err != nil {
		t.Fatalf("Failed to build regular message: %v", err)
	}

	// Test that it's NOT recognized as a system message
	if regularMsg.IsSystemMessage() {
		t.Error("Regular message should not be recognized as system message")
	}

	// Test that GetSystemMessage fails for regular messages
	_, err = regularMsg.GetSystemMessage()
	if err == nil {
		t.Error("Expected error when calling GetSystemMessage on regular message")
	}
}

// TestSystemMessageWithInvalidJSON tests handling of invalid system message JSON
func TestSystemMessageWithInvalidJSON(t *testing.T) {
	// Create a message with invalid system message JSON in body
	msg := &message.Message{
		From:      "system#example.com",
		To:        []string{"group#example.com"},
		Body:      "invalid json {",
		Type:      message.SystemJoined,
		Timestamp: time.Now().Unix(),
	}

	// Test that validation fails
	if err := msg.Validate(); err == nil {
		t.Error("Expected validation to fail for invalid system message JSON")
	}

	// Test that GetSystemMessage fails
	_, err := msg.GetSystemMessage()
	if err == nil {
		t.Error("Expected GetSystemMessage to fail for invalid JSON")
	}
}
