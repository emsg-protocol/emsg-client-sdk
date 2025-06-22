package test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/groups"
	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
	"github.com/emsg-protocol/emsg-client-sdk/message"
)

// TestGroupManager tests the basic GroupManager functionality
func TestGroupManager(t *testing.T) {
	gm := groups.NewGroupManager()
	if gm == nil {
		t.Fatal("Failed to create GroupManager")
	}

	// Test creating a group
	groupID := "test-group#example.com"
	groupName := "Test Group"
	createdBy := "alice#example.com"

	group, err := gm.CreateGroup(groupID, groupName, createdBy, nil)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	if group.ID != groupID {
		t.Errorf("Expected group ID %s, got %s", groupID, group.ID)
	}

	if group.Name != groupName {
		t.Errorf("Expected group name %s, got %s", groupName, group.Name)
	}

	if group.CreatedBy != createdBy {
		t.Errorf("Expected created by %s, got %s", createdBy, group.CreatedBy)
	}

	// Check that creator is added as owner
	member, err := group.GetMember(createdBy)
	if err != nil {
		t.Fatalf("Failed to get creator member: %v", err)
	}

	if member.Role != groups.RoleOwner {
		t.Errorf("Expected creator role %s, got %s", groups.RoleOwner, member.Role)
	}

	// Test duplicate group creation
	_, err = gm.CreateGroup(groupID, "Duplicate", createdBy, nil)
	if err == nil {
		t.Error("Expected error when creating duplicate group")
	}

	// Test getting group
	retrievedGroup, err := gm.GetGroup(groupID)
	if err != nil {
		t.Fatalf("Failed to get group: %v", err)
	}

	if retrievedGroup.ID != groupID {
		t.Errorf("Retrieved group ID mismatch: expected %s, got %s", groupID, retrievedGroup.ID)
	}

	// Test getting non-existent group
	_, err = gm.GetGroup("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent group")
	}

	// Test listing groups
	groups := gm.ListGroups()
	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}

	// Test deleting group
	err = gm.DeleteGroup(groupID, createdBy)
	if err != nil {
		t.Fatalf("Failed to delete group: %v", err)
	}

	// Verify group is deleted
	_, err = gm.GetGroup(groupID)
	if err == nil {
		t.Error("Expected error when getting deleted group")
	}

	// Test deleting non-existent group
	err = gm.DeleteGroup("non-existent", createdBy)
	if err == nil {
		t.Error("Expected error when deleting non-existent group")
	}
}

// TestGroupSettings tests group settings functionality
func TestGroupSettings(t *testing.T) {
	// Test default settings
	defaultSettings := groups.DefaultGroupSettings()
	if defaultSettings == nil {
		t.Fatal("Failed to get default settings")
	}

	if defaultSettings.IsPublic {
		t.Error("Default settings should not be public")
	}

	if !defaultSettings.RequireInvite {
		t.Error("Default settings should require invite")
	}

	if defaultSettings.MaxMembers != 100 {
		t.Errorf("Expected max members 100, got %d", defaultSettings.MaxMembers)
	}

	// Test custom settings
	customSettings := &groups.GroupSettings{
		IsPublic:           true,
		RequireInvite:      false,
		AllowGuestMessages: true,
		MaxMembers:         50,
		MessageRetention:   7 * 24 * time.Hour,
		Permissions: map[groups.GroupRole][]groups.Permission{
			groups.RoleOwner: {groups.PermissionSendMessage, groups.PermissionDeleteGroup},
			groups.RoleMember: {groups.PermissionSendMessage},
		},
	}

	gm := groups.NewGroupManager()
	group, err := gm.CreateGroup("custom-group#example.com", "Custom Group", "alice#example.com", customSettings)
	if err != nil {
		t.Fatalf("Failed to create group with custom settings: %v", err)
	}

	if !group.Settings.IsPublic {
		t.Error("Custom settings should be public")
	}

	if group.Settings.RequireInvite {
		t.Error("Custom settings should not require invite")
	}

	if group.Settings.MaxMembers != 50 {
		t.Errorf("Expected max members 50, got %d", group.Settings.MaxMembers)
	}
}

// TestGroupMemberManagement tests adding, removing, and managing group members
func TestGroupMemberManagement(t *testing.T) {
	gm := groups.NewGroupManager()
	groupID := "member-test#example.com"
	owner := "alice#example.com"
	
	group, err := gm.CreateGroup(groupID, "Member Test Group", owner, nil)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Test adding members
	member1 := "bob#example.com"
	member2 := "charlie#example.com"

	err = group.AddMember(member1, owner, groups.RoleMember)
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	err = group.AddMember(member2, owner, groups.RoleAdmin)
	if err != nil {
		t.Fatalf("Failed to add admin: %v", err)
	}

	// Test getting members
	bobMember, err := group.GetMember(member1)
	if err != nil {
		t.Fatalf("Failed to get member: %v", err)
	}

	if bobMember.Role != groups.RoleMember {
		t.Errorf("Expected role %s, got %s", groups.RoleMember, bobMember.Role)
	}

	if bobMember.InvitedBy != owner {
		t.Errorf("Expected invited by %s, got %s", owner, bobMember.InvitedBy)
	}

	// Test duplicate member addition
	err = group.AddMember(member1, owner, groups.RoleMember)
	if err == nil {
		t.Error("Expected error when adding duplicate member")
	}

	// Test getting all members
	allMembers := group.GetMembers()
	if len(allMembers) != 3 { // owner + 2 members
		t.Errorf("Expected 3 members, got %d", len(allMembers))
	}

	// Test getting members by role
	admins := group.GetMembersByRole(groups.RoleAdmin)
	if len(admins) != 1 {
		t.Errorf("Expected 1 admin, got %d", len(admins))
	}

	members := group.GetMembersByRole(groups.RoleMember)
	if len(members) != 1 {
		t.Errorf("Expected 1 member, got %d", len(members))
	}

	owners := group.GetMembersByRole(groups.RoleOwner)
	if len(owners) != 1 {
		t.Errorf("Expected 1 owner, got %d", len(owners))
	}

	// Test removing members
	err = group.RemoveMember(member1, owner)
	if err != nil {
		t.Fatalf("Failed to remove member: %v", err)
	}

	// Verify member is removed
	_, err = group.GetMember(member1)
	if err == nil {
		t.Error("Expected error when getting removed member")
	}

	// Test removing non-existent member
	err = group.RemoveMember("non-existent#example.com", owner)
	if err == nil {
		t.Error("Expected error when removing non-existent member")
	}

	// Test removing owner (should fail)
	err = group.RemoveMember(owner, owner)
	if err == nil {
		t.Error("Expected error when removing owner")
	}
}

// TestGroupRoleManagement tests role changes and permissions
func TestGroupRoleManagement(t *testing.T) {
	gm := groups.NewGroupManager()
	groupID := "role-test#example.com"
	owner := "alice#example.com"

	group, err := gm.CreateGroup(groupID, "Role Test Group", owner, nil)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Add members with different roles
	admin := "bob#example.com"
	moderator := "charlie#example.com"
	member := "dave#example.com"

	err = group.AddMember(admin, owner, groups.RoleAdmin)
	if err != nil {
		t.Fatalf("Failed to add admin: %v", err)
	}

	err = group.AddMember(moderator, owner, groups.RoleModerator)
	if err != nil {
		t.Fatalf("Failed to add moderator: %v", err)
	}

	err = group.AddMember(member, owner, groups.RoleMember)
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Test role changes
	err = group.ChangeRole(member, owner, groups.RoleModerator)
	if err != nil {
		t.Fatalf("Failed to change role: %v", err)
	}

	// Verify role change
	updatedMember, err := group.GetMember(member)
	if err != nil {
		t.Fatalf("Failed to get updated member: %v", err)
	}

	if updatedMember.Role != groups.RoleModerator {
		t.Errorf("Expected role %s, got %s", groups.RoleModerator, updatedMember.Role)
	}

	// Test changing to owner role (should fail)
	err = group.ChangeRole(member, owner, groups.RoleOwner)
	if err == nil {
		t.Error("Expected error when changing to owner role")
	}

	// Test changing owner role (should fail)
	err = group.ChangeRole(owner, owner, groups.RoleAdmin)
	if err == nil {
		t.Error("Expected error when changing owner role")
	}

	// Test insufficient permissions for role change
	err = group.ChangeRole(admin, member, groups.RoleMember)
	if err == nil {
		t.Error("Expected error when member tries to change admin role")
	}
}

// TestGroupPermissions tests permission checking functionality
func TestGroupPermissions(t *testing.T) {
	gm := groups.NewGroupManager()
	groupID := "permission-test#example.com"
	owner := "alice#example.com"

	group, err := gm.CreateGroup(groupID, "Permission Test Group", owner, nil)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Add members with different roles
	admin := "bob#example.com"
	moderator := "charlie#example.com"
	member := "dave#example.com"
	guest := "eve#example.com"

	err = group.AddMember(admin, owner, groups.RoleAdmin)
	if err != nil {
		t.Fatalf("Failed to add admin: %v", err)
	}

	err = group.AddMember(moderator, owner, groups.RoleModerator)
	if err != nil {
		t.Fatalf("Failed to add moderator: %v", err)
	}

	err = group.AddMember(member, owner, groups.RoleMember)
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	err = group.AddMember(guest, owner, groups.RoleGuest)
	if err != nil {
		t.Fatalf("Failed to add guest: %v", err)
	}

	// Test owner permissions
	if !group.HasPermission(owner, groups.PermissionDeleteGroup) {
		t.Error("Owner should have delete group permission")
	}

	if !group.HasPermission(owner, groups.PermissionSendMessage) {
		t.Error("Owner should have send message permission")
	}

	// Test admin permissions
	if !group.HasPermission(admin, groups.PermissionAddMember) {
		t.Error("Admin should have add member permission")
	}

	if group.HasPermission(admin, groups.PermissionDeleteGroup) {
		t.Error("Admin should not have delete group permission")
	}

	// Test moderator permissions
	if !group.HasPermission(moderator, groups.PermissionDeleteMessage) {
		t.Error("Moderator should have delete message permission")
	}

	if group.HasPermission(moderator, groups.PermissionRemoveMember) {
		t.Error("Moderator should not have remove member permission")
	}

	// Test member permissions
	if !group.HasPermission(member, groups.PermissionSendMessage) {
		t.Error("Member should have send message permission")
	}

	if group.HasPermission(member, groups.PermissionAddMember) {
		t.Error("Member should not have add member permission")
	}

	// Test guest permissions
	if !group.HasPermission(guest, groups.PermissionViewHistory) {
		t.Error("Guest should have view history permission")
	}

	if group.HasPermission(guest, groups.PermissionSendMessage) {
		t.Error("Guest should not have send message permission")
	}

	// Test non-member permissions
	if group.HasPermission("stranger#example.com", groups.PermissionViewHistory) {
		t.Error("Non-member should not have any permissions")
	}
}

// TestGroupMessageCreation tests creating group management messages
func TestGroupMessageCreation(t *testing.T) {
	groupID := "message-test#example.com"
	actor := "alice#example.com"

	// Test creating group message
	data := map[string]any{
		"member": "bob#example.com",
		"role":   "member",
	}

	msg, err := groups.CreateGroupMessage(groupID, "member_added", actor, data)
	if err != nil {
		t.Fatalf("Failed to create group message: %v", err)
	}

	if msg.Type != "group:member_added" {
		t.Errorf("Expected type 'group:member_added', got '%s'", msg.Type)
	}

	if msg.GroupID != groupID {
		t.Errorf("Expected group ID %s, got %s", groupID, msg.GroupID)
	}

	// Parse system message from body
	var systemMsg message.SystemMessage
	err = json.Unmarshal([]byte(msg.Body), &systemMsg)
	if err != nil {
		t.Fatalf("Failed to parse system message: %v", err)
	}

	if systemMsg.Actor != actor {
		t.Errorf("Expected actor %s, got %s", actor, systemMsg.Actor)
	}

	if systemMsg.GroupID != groupID {
		t.Errorf("Expected group ID %s, got %s", groupID, systemMsg.GroupID)
	}

	// Test message signing
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	err = msg.Sign(keyPair)
	if err != nil {
		t.Fatalf("Failed to sign group message: %v", err)
	}

	// Verify signature
	err = msg.Verify(keyPair.PublicKeyBase64())
	if err != nil {
		t.Fatalf("Failed to verify group message signature: %v", err)
	}
}

// TestGroupJSONSerialization tests JSON serialization of groups
func TestGroupJSONSerialization(t *testing.T) {
	gm := groups.NewGroupManager()
	groupID := "json-test#example.com"
	owner := "alice#example.com"

	// Create group with custom settings
	customSettings := &groups.GroupSettings{
		IsPublic:           true,
		RequireInvite:      false,
		AllowGuestMessages: true,
		MaxMembers:         25,
		MessageRetention:   14 * 24 * time.Hour,
		Permissions: map[groups.GroupRole][]groups.Permission{
			groups.RoleOwner: {
				groups.PermissionSendMessage, groups.PermissionDeleteGroup,
				groups.PermissionAddMember, groups.PermissionRemoveMember,
				groups.PermissionChangeRole, groups.PermissionManageGroup,
				groups.PermissionViewMembers, groups.PermissionViewHistory,
			},
			groups.RoleMember: {groups.PermissionSendMessage},
		},
	}

	group, err := gm.CreateGroup(groupID, "JSON Test Group", owner, customSettings)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Add some members
	err = group.AddMember("bob#example.com", owner, groups.RoleAdmin)
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	err = group.AddMember("charlie#example.com", owner, groups.RoleMember)
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Add metadata
	group.Metadata["description"] = "A test group for JSON serialization"
	group.Metadata["tags"] = []string{"test", "json", "serialization"}

	// Serialize to JSON
	jsonData, err := group.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize group to JSON: %v", err)
	}

	// Deserialize from JSON
	deserializedGroup, err := groups.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize group from JSON: %v", err)
	}

	// Verify deserialized group
	if deserializedGroup.ID != group.ID {
		t.Errorf("ID mismatch: expected %s, got %s", group.ID, deserializedGroup.ID)
	}

	if deserializedGroup.Name != group.Name {
		t.Errorf("Name mismatch: expected %s, got %s", group.Name, deserializedGroup.Name)
	}

	if deserializedGroup.CreatedBy != group.CreatedBy {
		t.Errorf("CreatedBy mismatch: expected %s, got %s", group.CreatedBy, deserializedGroup.CreatedBy)
	}

	if len(deserializedGroup.Members) != len(group.Members) {
		t.Errorf("Members count mismatch: expected %d, got %d", len(group.Members), len(deserializedGroup.Members))
	}

	// Verify settings
	if deserializedGroup.Settings.IsPublic != group.Settings.IsPublic {
		t.Error("IsPublic setting mismatch")
	}

	if deserializedGroup.Settings.MaxMembers != group.Settings.MaxMembers {
		t.Errorf("MaxMembers mismatch: expected %d, got %d", group.Settings.MaxMembers, deserializedGroup.Settings.MaxMembers)
	}

	// Verify metadata
	if deserializedGroup.Metadata["description"] != group.Metadata["description"] {
		t.Error("Metadata description mismatch")
	}
}

// TestGroupMemberLimits tests member limit enforcement
func TestGroupMemberLimits(t *testing.T) {
	gm := groups.NewGroupManager()
	groupID := "limit-test#example.com"
	owner := "alice#example.com"

	// Create group with small member limit
	settings := &groups.GroupSettings{
		IsPublic:           false,
		RequireInvite:      true,
		AllowGuestMessages: false,
		MaxMembers:         3, // Small limit for testing
		MessageRetention:   24 * time.Hour,
		Permissions:        groups.DefaultGroupSettings().Permissions,
	}

	group, err := gm.CreateGroup(groupID, "Limit Test Group", owner, settings)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Add members up to limit (owner counts as 1)
	err = group.AddMember("bob#example.com", owner, groups.RoleMember)
	if err != nil {
		t.Fatalf("Failed to add first member: %v", err)
	}

	err = group.AddMember("charlie#example.com", owner, groups.RoleMember)
	if err != nil {
		t.Fatalf("Failed to add second member: %v", err)
	}

	// Try to add one more member (should fail)
	err = group.AddMember("dave#example.com", owner, groups.RoleMember)
	if err == nil {
		t.Error("Expected error when exceeding member limit")
	}

	// Verify current member count
	members := group.GetMembers()
	if len(members) != 3 {
		t.Errorf("Expected 3 members, got %d", len(members))
	}
}

// TestGroupRoleHierarchy tests role hierarchy enforcement
func TestGroupRoleHierarchy(t *testing.T) {
	gm := groups.NewGroupManager()
	groupID := "hierarchy-test#example.com"
	owner := "alice#example.com"

	group, err := gm.CreateGroup(groupID, "Hierarchy Test Group", owner, nil)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Add members with different roles
	admin := "bob#example.com"
	moderator := "charlie#example.com"
	member := "dave#example.com"

	err = group.AddMember(admin, owner, groups.RoleAdmin)
	if err != nil {
		t.Fatalf("Failed to add admin: %v", err)
	}

	err = group.AddMember(moderator, owner, groups.RoleModerator)
	if err != nil {
		t.Fatalf("Failed to add moderator: %v", err)
	}

	err = group.AddMember(member, owner, groups.RoleMember)
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Test that admin cannot remove owner
	err = group.RemoveMember(owner, admin)
	if err == nil {
		t.Error("Admin should not be able to remove owner")
	}

	// Test that moderator cannot remove admin
	err = group.RemoveMember(admin, moderator)
	if err == nil {
		t.Error("Moderator should not be able to remove admin")
	}

	// Test that member cannot remove moderator
	err = group.RemoveMember(moderator, member)
	if err == nil {
		t.Error("Member should not be able to remove moderator")
	}

	// Test that admin can remove moderator
	err = group.RemoveMember(moderator, admin)
	if err != nil {
		t.Errorf("Admin should be able to remove moderator: %v", err)
	}

	// Test role change hierarchy
	// Re-add moderator
	err = group.AddMember(moderator, owner, groups.RoleModerator)
	if err != nil {
		t.Fatalf("Failed to re-add moderator: %v", err)
	}

	// Test that moderator cannot promote member to admin
	err = group.ChangeRole(member, moderator, groups.RoleAdmin)
	if err == nil {
		t.Error("Moderator should not be able to promote member to admin")
	}

	// Test that admin can promote member to moderator
	err = group.ChangeRole(member, admin, groups.RoleModerator)
	if err != nil {
		t.Errorf("Admin should be able to promote member to moderator: %v", err)
	}
}

// TestGroupInvalidOperations tests various invalid operations
func TestGroupInvalidOperations(t *testing.T) {
	gm := groups.NewGroupManager()
	groupID := "invalid-test#example.com"
	owner := "alice#example.com"

	group, err := gm.CreateGroup(groupID, "Invalid Test Group", owner, nil)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Test adding member without permission
	unauthorizedUser := "eve#example.com"
	err = group.AddMember("bob#example.com", unauthorizedUser, groups.RoleMember)
	if err == nil {
		t.Error("Expected error when unauthorized user tries to add member")
	}

	// Test removing member without permission
	err = group.AddMember("bob#example.com", owner, groups.RoleMember)
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	err = group.RemoveMember("bob#example.com", unauthorizedUser)
	if err == nil {
		t.Error("Expected error when unauthorized user tries to remove member")
	}

	// Test changing role without permission
	err = group.ChangeRole("bob#example.com", unauthorizedUser, groups.RoleAdmin)
	if err == nil {
		t.Error("Expected error when unauthorized user tries to change role")
	}

	// Test deleting group without permission
	err = gm.DeleteGroup(groupID, unauthorizedUser)
	if err == nil {
		t.Error("Expected error when unauthorized user tries to delete group")
	}

	// Test operations on non-existent members
	err = group.RemoveMember("nonexistent#example.com", owner)
	if err == nil {
		t.Error("Expected error when removing non-existent member")
	}

	err = group.ChangeRole("nonexistent#example.com", owner, groups.RoleAdmin)
	if err == nil {
		t.Error("Expected error when changing role of non-existent member")
	}

	_, err = group.GetMember("nonexistent#example.com")
	if err == nil {
		t.Error("Expected error when getting non-existent member")
	}
}
