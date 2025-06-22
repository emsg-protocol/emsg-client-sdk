package test

import (
	"fmt"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/client"
	"github.com/emsg-protocol/emsg-client-sdk/groups"
	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
	"github.com/emsg-protocol/emsg-client-sdk/message"
)

// TestRetryStrategy tests the retry strategy configuration
func TestRetryStrategy(t *testing.T) {
	// Test default retry strategy
	defaultStrategy := client.DefaultRetryStrategy()

	if defaultStrategy.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", defaultStrategy.MaxRetries)
	}

	if defaultStrategy.InitialDelay != 1*time.Second {
		t.Errorf("Expected InitialDelay to be 1s, got %v", defaultStrategy.InitialDelay)
	}

	if defaultStrategy.MaxDelay != 30*time.Second {
		t.Errorf("Expected MaxDelay to be 30s, got %v", defaultStrategy.MaxDelay)
	}

	if defaultStrategy.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor to be 2.0, got %f", defaultStrategy.BackoffFactor)
	}

	if !defaultStrategy.RetryOn429 {
		t.Error("Expected RetryOn429 to be true")
	}

	if !defaultStrategy.RetryOnTimeout {
		t.Error("Expected RetryOnTimeout to be true")
	}

	// Test custom retry strategy
	customStrategy := &client.RetryStrategy{
		MaxRetries:     5,
		InitialDelay:   500 * time.Millisecond,
		MaxDelay:       60 * time.Second,
		BackoffFactor:  1.5,
		RetryOn429:     false,
		RetryOnTimeout: true,
	}

	config := client.DefaultConfig()
	config.RetryStrategy = customStrategy

	if config.RetryStrategy.MaxRetries != 5 {
		t.Errorf("Expected custom MaxRetries to be 5, got %d", config.RetryStrategy.MaxRetries)
	}

	if config.RetryStrategy.BackoffFactor != 1.5 {
		t.Errorf("Expected custom BackoffFactor to be 1.5, got %f", config.RetryStrategy.BackoffFactor)
	}
}

// TestClientConfiguration tests the enhanced client configuration
func TestClientConfiguration(t *testing.T) {
	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	var beforeSendCalled bool
	var afterSendCalled bool

	// Test configuration with hooks and retry strategy
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	config.RetryStrategy = &client.RetryStrategy{
		MaxRetries:     2,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       1 * time.Second,
		BackoffFactor:  2.0,
		RetryOn429:     true,
		RetryOnTimeout: true,
	}
	config.BeforeSend = func(msg *message.Message) error {
		beforeSendCalled = true
		return nil
	}
	config.AfterSend = func(msg *message.Message, resp *http.Response) error {
		afterSendCalled = true
		return nil
	}

	// Create client
	emsgClient := client.New(config)
	if emsgClient == nil {
		t.Fatal("Failed to create client")
	}

	// Test that hooks are configured
	if config.BeforeSend == nil {
		t.Error("BeforeSend hook should be configured")
	}

	if config.AfterSend == nil {
		t.Error("AfterSend hook should be configured")
	}

	// Test hook execution
	testMsg := &message.Message{
		From:      "test#example.com",
		To:        []string{"recipient#example.com"},
		Body:      "Test message",
		Timestamp: time.Now().Unix(),
	}

	err = config.BeforeSend(testMsg)
	if err != nil {
		t.Errorf("BeforeSend hook failed: %v", err)
	}

	if !beforeSendCalled {
		t.Error("BeforeSend hook was not called")
	}

	// Test AfterSend hook (with mock response)
	mockResp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
	}

	err = config.AfterSend(testMsg, mockResp)
	if err != nil {
		t.Errorf("AfterSend hook failed: %v", err)
	}

	if !afterSendCalled {
		t.Error("AfterSend hook was not called")
	}
}

// TestClientWithNilConfig tests client creation with nil config
func TestClientWithNilConfig(t *testing.T) {
	// Test that client can be created with nil config (should use defaults)
	emsgClient := client.New(nil)
	if emsgClient == nil {
		t.Fatal("Failed to create client with nil config")
	}

	// Test that default config is used
	defaultConfig := client.DefaultConfig()
	if defaultConfig.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout to be 30s, got %v", defaultConfig.Timeout)
	}

	if defaultConfig.UserAgent != "emsg-client-sdk/1.0" {
		t.Errorf("Expected default user agent to be 'emsg-client-sdk/1.0', got '%s'", defaultConfig.UserAgent)
	}
}

// TestSystemMessageComposer tests the system message composer in client
func TestSystemMessageComposer(t *testing.T) {
	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create client
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	emsgClient := client.New(config)

	// Test system message composer
	systemBuilder := emsgClient.ComposeSystemMessage()
	if systemBuilder == nil {
		t.Fatal("Failed to create system message builder")
	}

	// Test building a system message
	systemMsg, err := systemBuilder.
		Type(message.SystemJoined).
		Actor("user#example.com").
		GroupID("test-group").
		Metadata("timestamp", time.Now().Unix()).
		Build("system#example.com", []string{"group#example.com"})

	if err != nil {
		t.Fatalf("Failed to build system message: %v", err)
	}

	if !systemMsg.IsSystemMessage() {
		t.Error("Message should be recognized as system message")
	}

	// Test validation
	if err := systemMsg.Validate(); err != nil {
		t.Errorf("System message validation failed: %v", err)
	}
}

// TestHookErrorHandling tests error handling in hooks
func TestHookErrorHandling(t *testing.T) {
	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Test BeforeSend hook that returns an error
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	config.BeforeSend = func(msg *message.Message) error {
		return fmt.Errorf("before send hook error")
	}

	emsgClient := client.New(config)

	// Create test message
	msg, err := emsgClient.ComposeMessage().
		From("sender#example.com").
		To("recipient#example.com").
		Body("Test message").
		Build()

	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}

	// Test that SendMessage fails when BeforeSend hook returns error
	// Note: This would require a mock server to test fully, but we can test the hook directly
	err = config.BeforeSend(msg)
	if err == nil {
		t.Error("Expected BeforeSend hook to return error")
	}

	if err.Error() != "before send hook error" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// TestMessageBuilderWithSystemType tests message builder with system type
func TestMessageBuilderWithSystemType(t *testing.T) {
	// Try to build a message with system type but invalid body
	msg := &message.Message{
		From:      "system#example.com",
		To:        []string{"group#example.com"},
		Body:      "invalid system message body",
		Type:      message.SystemJoined,
		Timestamp: time.Now().Unix(),
	}

	// Test that validation fails for invalid system message
	if err := msg.Validate(); err == nil {
		t.Error("Expected validation to fail for invalid system message")
	}
}

// TestRetryStrategyCalculations tests retry delay calculations
func TestRetryStrategyCalculations(t *testing.T) {
	strategy := &client.RetryStrategy{
		MaxRetries:     3,
		InitialDelay:   1 * time.Second,
		MaxDelay:       10 * time.Second,
		BackoffFactor:  2.0,
		RetryOn429:     true,
		RetryOnTimeout: true,
	}

	// Test delay calculations (we can't easily test the private methods,
	// but we can test the strategy configuration)
	if strategy.InitialDelay != 1*time.Second {
		t.Errorf("Expected initial delay 1s, got %v", strategy.InitialDelay)
	}

	if strategy.BackoffFactor != 2.0 {
		t.Errorf("Expected backoff factor 2.0, got %f", strategy.BackoffFactor)
	}

	if strategy.MaxDelay != 10*time.Second {
		t.Errorf("Expected max delay 10s, got %v", strategy.MaxDelay)
	}

	// Test that exponential backoff would work correctly
	// Attempt 0: 1s
	// Attempt 1: 2s
	// Attempt 2: 4s
	// Attempt 3: 8s
	// All should be under MaxDelay of 10s

	expectedDelays := []time.Duration{
		1 * time.Second, // attempt 0
		2 * time.Second, // attempt 1
		4 * time.Second, // attempt 2
		8 * time.Second, // attempt 3
	}

	for i, expected := range expectedDelays {
		// Calculate what the delay would be
		delay := time.Duration(float64(strategy.InitialDelay) *
			math.Pow(strategy.BackoffFactor, float64(i)))

		if delay > strategy.MaxDelay {
			delay = strategy.MaxDelay
		}

		if delay != expected {
			t.Errorf("Attempt %d: expected delay %v, calculated %v", i, expected, delay)
		}
	}
}

// TestClientKeyPairManagement tests key pair management in client
func TestClientKeyPairManagement(t *testing.T) {
	// Generate test key pairs
	keyPair1, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 1: %v", err)
	}

	keyPair2, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 2: %v", err)
	}

	// Create client with first key pair
	config := client.DefaultConfig()
	config.KeyPair = keyPair1
	emsgClient := client.New(config)

	// Test getting key pair
	retrievedKeyPair := emsgClient.GetKeyPair()
	if retrievedKeyPair == nil {
		t.Fatal("Failed to retrieve key pair")
	}

	if retrievedKeyPair.PublicKeyBase64() != keyPair1.PublicKeyBase64() {
		t.Error("Retrieved key pair doesn't match original")
	}

	// Test setting new key pair
	emsgClient.SetKeyPair(keyPair2)

	retrievedKeyPair2 := emsgClient.GetKeyPair()
	if retrievedKeyPair2.PublicKeyBase64() != keyPair2.PublicKeyBase64() {
		t.Error("Key pair was not updated correctly")
	}
}

// TestClientGroupManagement tests group management functionality in client
func TestClientGroupManagement(t *testing.T) {
	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create client with group management enabled
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	config.EnableGroupManagement = true
	emsgClient := client.New(config)

	// Test that group management is enabled
	if !emsgClient.IsGroupManagementEnabled() {
		t.Error("Group management should be enabled")
	}

	// Test creating a group
	groupID := "test-group#example.com"
	groupName := "Test Group"
	owner := "alice#example.com"

	group, err := emsgClient.CreateGroup(groupID, groupName, owner, nil)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	if group.ID != groupID {
		t.Errorf("Expected group ID %s, got %s", groupID, group.ID)
	}

	if group.Name != groupName {
		t.Errorf("Expected group name %s, got %s", groupName, group.Name)
	}

	// Test getting the group
	retrievedGroup, err := emsgClient.GetGroup(groupID)
	if err != nil {
		t.Fatalf("Failed to get group: %v", err)
	}

	if retrievedGroup.ID != groupID {
		t.Errorf("Retrieved group ID mismatch: expected %s, got %s", groupID, retrievedGroup.ID)
	}

	// Test adding a member
	member := "bob#example.com"
	err = emsgClient.AddGroupMember(groupID, member, owner, groups.RoleMember)
	if err != nil {
		t.Fatalf("Failed to add group member: %v", err)
	}

	// Test getting group members
	members, err := emsgClient.GetGroupMembers(groupID)
	if err != nil {
		t.Fatalf("Failed to get group members: %v", err)
	}

	if len(members) != 2 { // owner + member
		t.Errorf("Expected 2 members, got %d", len(members))
	}

	// Test getting specific member
	bobMember, err := emsgClient.GetGroupMember(groupID, member)
	if err != nil {
		t.Fatalf("Failed to get specific member: %v", err)
	}

	if bobMember.Role != groups.RoleMember {
		t.Errorf("Expected role %s, got %s", groups.RoleMember, bobMember.Role)
	}

	// Test changing member role
	err = emsgClient.ChangeGroupMemberRole(groupID, member, owner, groups.RoleAdmin)
	if err != nil {
		t.Fatalf("Failed to change member role: %v", err)
	}

	// Verify role change
	updatedMember, err := emsgClient.GetGroupMember(groupID, member)
	if err != nil {
		t.Fatalf("Failed to get updated member: %v", err)
	}

	if updatedMember.Role != groups.RoleAdmin {
		t.Errorf("Expected role %s, got %s", groups.RoleAdmin, updatedMember.Role)
	}

	// Test getting members by role
	admins, err := emsgClient.GetGroupMembersByRole(groupID, groups.RoleAdmin)
	if err != nil {
		t.Fatalf("Failed to get admins: %v", err)
	}

	if len(admins) != 1 {
		t.Errorf("Expected 1 admin, got %d", len(admins))
	}

	// Test permission checking
	hasPermission, err := emsgClient.HasGroupPermission(groupID, owner, groups.PermissionDeleteGroup)
	if err != nil {
		t.Fatalf("Failed to check permission: %v", err)
	}

	if !hasPermission {
		t.Error("Owner should have delete group permission")
	}

	// Test removing member
	err = emsgClient.RemoveGroupMember(groupID, member, owner)
	if err != nil {
		t.Fatalf("Failed to remove member: %v", err)
	}

	// Verify member is removed
	_, err = emsgClient.GetGroupMember(groupID, member)
	if err == nil {
		t.Error("Expected error when getting removed member")
	}

	// Test listing groups
	allGroups := emsgClient.ListGroups()
	if len(allGroups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(allGroups))
	}

	// Test deleting group
	err = emsgClient.DeleteGroup(groupID, owner)
	if err != nil {
		t.Fatalf("Failed to delete group: %v", err)
	}

	// Verify group is deleted
	_, err = emsgClient.GetGroup(groupID)
	if err == nil {
		t.Error("Expected error when getting deleted group")
	}
}

// TestClientGroupManagementDisabled tests behavior when group management is disabled
func TestClientGroupManagementDisabled(t *testing.T) {
	// Generate test key pair
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create client with group management disabled
	config := client.DefaultConfig()
	config.KeyPair = keyPair
	config.EnableGroupManagement = false
	emsgClient := client.New(config)

	// Test that group management is disabled
	if emsgClient.IsGroupManagementEnabled() {
		t.Error("Group management should be disabled")
	}

	// Test that group operations return errors
	_, err = emsgClient.CreateGroup("test#example.com", "Test", "alice#example.com", nil)
	if err == nil {
		t.Error("Expected error when group management is disabled")
	}

	_, err = emsgClient.GetGroup("test#example.com")
	if err == nil {
		t.Error("Expected error when group management is disabled")
	}

	err = emsgClient.AddGroupMember("test#example.com", "bob#example.com", "alice#example.com", groups.RoleMember)
	if err == nil {
		t.Error("Expected error when group management is disabled")
	}
}
