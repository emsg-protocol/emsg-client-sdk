package main

import (
	"fmt"
	"log"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/client"
	"github.com/emsg-protocol/emsg-client-sdk/groups"
	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
)

func main() {
	fmt.Println("=== EMSG Group Management Demo ===")
	fmt.Println()

	// Generate key pairs for demo users
	fmt.Println("1. Generating key pairs for demo users...")
	aliceKeyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate Alice's key pair: %v", err)
	}

	bobKeyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate Bob's key pair: %v", err)
	}

	charlieKeyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate Charlie's key pair: %v", err)
	}

	// Create client for Alice (group owner)
	fmt.Println("2. Creating EMSG client for Alice (group owner)...")
	config := client.DefaultConfig()
	config.KeyPair = aliceKeyPair
	config.EnableGroupManagement = true
	aliceClient := client.New(config)

	// Demo addresses
	aliceAddr := "alice@example.com"
	bobAddr := "bob@example.com"
	charlieAddr := "charlie@example.com"
	daveAddr := "dave@example.com"

	fmt.Printf("   Alice: %s\n", aliceAddr)
	fmt.Printf("   Bob: %s\n", bobAddr)
	fmt.Printf("   Charlie: %s\n", charlieAddr)
	fmt.Printf("   Dave: %s\n", daveAddr)
	fmt.Println()

	// Create a group
	fmt.Println("3. Creating a new group...")
	groupID := "dev-team#example.com"
	groupName := "Development Team"

	// Custom group settings
	groupSettings := &groups.GroupSettings{
		IsPublic:           false,
		RequireInvite:      true,
		AllowGuestMessages: false,
		MaxMembers:         10,
		MessageRetention:   30 * 24 * time.Hour, // 30 days
		Permissions:        groups.DefaultGroupSettings().Permissions,
	}

	group, err := aliceClient.CreateGroupWithMessage(groupID, groupName, aliceAddr, groupSettings)
	if err != nil {
		log.Fatalf("Failed to create group: %v", err)
	}

	fmt.Printf("   ✓ Created group: %s (%s)\n", group.Name, group.ID)
	fmt.Printf("   ✓ Owner: %s\n", group.CreatedBy)
	fmt.Printf("   ✓ Max members: %d\n", group.Settings.MaxMembers)
	fmt.Printf("   ✓ Public: %t\n", group.Settings.IsPublic)
	fmt.Println()

	// Add members to the group
	fmt.Println("4. Adding members to the group...")

	// Add Bob as admin
	err = aliceClient.AddGroupMemberWithMessage(groupID, bobAddr, aliceAddr, groups.RoleAdmin)
	if err != nil {
		log.Fatalf("Failed to add Bob as admin: %v", err)
	}
	fmt.Printf("   ✓ Added %s as %s\n", bobAddr, groups.RoleAdmin)

	// Add Charlie as member
	err = aliceClient.AddGroupMemberWithMessage(groupID, charlieAddr, aliceAddr, groups.RoleMember)
	if err != nil {
		log.Fatalf("Failed to add Charlie as member: %v", err)
	}
	fmt.Printf("   ✓ Added %s as %s\n", charlieAddr, groups.RoleMember)

	// Add Dave as guest
	err = aliceClient.AddGroupMemberWithMessage(groupID, daveAddr, aliceAddr, groups.RoleGuest)
	if err != nil {
		log.Fatalf("Failed to add Dave as guest: %v", err)
	}
	fmt.Printf("   ✓ Added %s as %s\n", daveAddr, groups.RoleGuest)
	fmt.Println()

	// Display group members
	fmt.Println("5. Current group members:")
	members, err := aliceClient.GetGroupMembers(groupID)
	if err != nil {
		log.Fatalf("Failed to get group members: %v", err)
	}

	for _, member := range members {
		joinedTime := time.Unix(member.JoinedAt, 0).Format("2006-01-02 15:04:05")
		fmt.Printf("   • %s (%s) - joined %s", member.Address, member.Role, joinedTime)
		if member.InvitedBy != "" {
			fmt.Printf(" - invited by %s", member.InvitedBy)
		}
		fmt.Println()
	}
	fmt.Println()

	// Demonstrate permission checking
	fmt.Println("6. Checking permissions...")
	
	// Check Alice's permissions (owner)
	canDelete, _ := aliceClient.HasGroupPermission(groupID, aliceAddr, groups.PermissionDeleteGroup)
	canAddMember, _ := aliceClient.HasGroupPermission(groupID, aliceAddr, groups.PermissionAddMember)
	fmt.Printf("   Alice (owner) - can delete group: %t, can add members: %t\n", canDelete, canAddMember)

	// Check Bob's permissions (admin)
	canDelete, _ = aliceClient.HasGroupPermission(groupID, bobAddr, groups.PermissionDeleteGroup)
	canAddMember, _ = aliceClient.HasGroupPermission(groupID, bobAddr, groups.PermissionAddMember)
	fmt.Printf("   Bob (admin) - can delete group: %t, can add members: %t\n", canDelete, canAddMember)

	// Check Charlie's permissions (member)
	canDelete, _ = aliceClient.HasGroupPermission(groupID, charlieAddr, groups.PermissionDeleteGroup)
	canSendMessage, _ := aliceClient.HasGroupPermission(groupID, charlieAddr, groups.PermissionSendMessage)
	fmt.Printf("   Charlie (member) - can delete group: %t, can send messages: %t\n", canDelete, canSendMessage)

	// Check Dave's permissions (guest)
	canSendMessage, _ = aliceClient.HasGroupPermission(groupID, daveAddr, groups.PermissionSendMessage)
	canViewHistory, _ := aliceClient.HasGroupPermission(groupID, daveAddr, groups.PermissionViewHistory)
	fmt.Printf("   Dave (guest) - can send messages: %t, can view history: %t\n", canSendMessage, canViewHistory)
	fmt.Println()

	// Demonstrate role changes
	fmt.Println("7. Promoting Charlie from member to moderator...")
	err = aliceClient.ChangeGroupMemberRoleWithMessage(groupID, charlieAddr, aliceAddr, groups.RoleModerator)
	if err != nil {
		log.Fatalf("Failed to promote Charlie: %v", err)
	}

	// Verify role change
	charlie, err := aliceClient.GetGroupMember(groupID, charlieAddr)
	if err != nil {
		log.Fatalf("Failed to get Charlie's updated info: %v", err)
	}
	fmt.Printf("   ✓ Charlie's new role: %s\n", charlie.Role)

	// Check Charlie's new permissions
	canDeleteMessage, _ := aliceClient.HasGroupPermission(groupID, charlieAddr, groups.PermissionDeleteMessage)
	fmt.Printf("   ✓ Charlie can now delete messages: %t\n", canDeleteMessage)
	fmt.Println()

	// Demonstrate sending group messages
	fmt.Println("8. Sending group messages...")
	
	// Alice sends a welcome message
	err = aliceClient.SendGroupMessage(groupID, aliceAddr, "Welcome to the Development Team group! 🎉")
	if err != nil {
		log.Printf("Warning: Failed to send Alice's message: %v", err)
	} else {
		fmt.Printf("   ✓ Alice sent welcome message\n")
	}

	// Bob sends a message
	err = aliceClient.SendGroupMessage(groupID, bobAddr, "Thanks Alice! Excited to be part of the team.")
	if err != nil {
		log.Printf("Warning: Failed to send Bob's message: %v", err)
	} else {
		fmt.Printf("   ✓ Bob sent response message\n")
	}
	fmt.Println()

	// Get members by role
	fmt.Println("9. Members by role:")
	
	owners, _ := aliceClient.GetGroupMembersByRole(groupID, groups.RoleOwner)
	fmt.Printf("   Owners (%d): ", len(owners))
	for _, owner := range owners {
		fmt.Printf("%s ", owner.Address)
	}
	fmt.Println()

	admins, _ := aliceClient.GetGroupMembersByRole(groupID, groups.RoleAdmin)
	fmt.Printf("   Admins (%d): ", len(admins))
	for _, admin := range admins {
		fmt.Printf("%s ", admin.Address)
	}
	fmt.Println()

	moderators, _ := aliceClient.GetGroupMembersByRole(groupID, groups.RoleModerator)
	fmt.Printf("   Moderators (%d): ", len(moderators))
	for _, moderator := range moderators {
		fmt.Printf("%s ", moderator.Address)
	}
	fmt.Println()

	regularMembers, _ := aliceClient.GetGroupMembersByRole(groupID, groups.RoleMember)
	fmt.Printf("   Members (%d): ", len(regularMembers))
	for _, member := range regularMembers {
		fmt.Printf("%s ", member.Address)
	}
	fmt.Println()

	guests, _ := aliceClient.GetGroupMembersByRole(groupID, groups.RoleGuest)
	fmt.Printf("   Guests (%d): ", len(guests))
	for _, guest := range guests {
		fmt.Printf("%s ", guest.Address)
	}
	fmt.Println()
	fmt.Println()

	// Demonstrate member removal
	fmt.Println("10. Removing Dave from the group...")
	err = aliceClient.RemoveGroupMemberWithMessage(groupID, daveAddr, aliceAddr)
	if err != nil {
		log.Fatalf("Failed to remove Dave: %v", err)
	}
	fmt.Printf("    ✓ Dave has been removed from the group\n")

	// Verify removal
	_, err = aliceClient.GetGroupMember(groupID, daveAddr)
	if err != nil {
		fmt.Printf("    ✓ Confirmed: Dave is no longer a member\n")
	}
	fmt.Println()

	// List all groups
	fmt.Println("11. All groups managed by this client:")
	allGroups := aliceClient.ListGroups()
	for i, g := range allGroups {
		memberCount := len(g.Members)
		fmt.Printf("    %d. %s (%s) - %d members\n", i+1, g.Name, g.ID, memberCount)
	}
	fmt.Println()

	// Clean up - delete the group
	fmt.Println("12. Cleaning up - deleting the group...")
	err = aliceClient.DeleteGroup(groupID, aliceAddr)
	if err != nil {
		log.Fatalf("Failed to delete group: %v", err)
	}
	fmt.Printf("    ✓ Group %s has been deleted\n", groupID)

	// Verify deletion
	_, err = aliceClient.GetGroup(groupID)
	if err != nil {
		fmt.Printf("    ✓ Confirmed: Group no longer exists\n")
	}

	fmt.Println()
	fmt.Println("=== Group Management Demo Complete! ===")
	fmt.Println()
	fmt.Println("This demo showed:")
	fmt.Println("• Creating groups with custom settings")
	fmt.Println("• Adding members with different roles (Owner, Admin, Moderator, Member, Guest)")
	fmt.Println("• Checking role-based permissions")
	fmt.Println("• Promoting/demoting members")
	fmt.Println("• Sending group messages and management notifications")
	fmt.Println("• Listing members by role")
	fmt.Println("• Removing members")
	fmt.Println("• Deleting groups")
	fmt.Println()
	fmt.Println("All group control messages are signed and verifiable!")
}
