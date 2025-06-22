package groups

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/emsg-protocol/emsg-client-sdk/message"
)

// GroupRole represents a user's role in a group
type GroupRole string

const (
	RoleOwner     GroupRole = "owner"
	RoleAdmin     GroupRole = "admin"
	RoleModerator GroupRole = "moderator"
	RoleMember    GroupRole = "member"
	RoleGuest     GroupRole = "guest"
)

// Permission represents a specific permission
type Permission string

const (
	PermissionSendMessage    Permission = "send_message"
	PermissionDeleteMessage  Permission = "delete_message"
	PermissionAddMember      Permission = "add_member"
	PermissionRemoveMember   Permission = "remove_member"
	PermissionChangeRole     Permission = "change_role"
	PermissionManageGroup    Permission = "manage_group"
	PermissionViewMembers    Permission = "view_members"
	PermissionViewHistory    Permission = "view_history"
	PermissionCreateSubgroup Permission = "create_subgroup"
	PermissionDeleteGroup    Permission = "delete_group"
)

// GroupMember represents a member of a group
type GroupMember struct {
	Address   string    `json:"address"`
	Role      GroupRole `json:"role"`
	JoinedAt  int64     `json:"joined_at"`
	InvitedBy string    `json:"invited_by,omitempty"`
	Nickname  string    `json:"nickname,omitempty"`
	Status    string    `json:"status,omitempty"` // active, inactive, banned
}

// Group represents a messaging group
type Group struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	CreatedAt   int64                   `json:"created_at"`
	CreatedBy   string                  `json:"created_by"`
	Members     map[string]*GroupMember `json:"members"`
	Settings    *GroupSettings          `json:"settings"`
	Metadata    map[string]any          `json:"metadata,omitempty"`
	mutex       sync.RWMutex            `json:"-"`
}

// GroupSettings holds group configuration
type GroupSettings struct {
	IsPublic           bool                       `json:"is_public"`
	RequireInvite      bool                       `json:"require_invite"`
	AllowGuestMessages bool                       `json:"allow_guest_messages"`
	MaxMembers         int                        `json:"max_members"`
	MessageRetention   time.Duration              `json:"message_retention"`
	Permissions        map[GroupRole][]Permission `json:"permissions"`
}

// GroupManager manages groups and their operations
type GroupManager struct {
	groups map[string]*Group
	mutex  sync.RWMutex
}

// NewGroupManager creates a new group manager
func NewGroupManager() *GroupManager {
	return &GroupManager{
		groups: make(map[string]*Group),
	}
}

// DefaultGroupSettings returns default group settings
func DefaultGroupSettings() *GroupSettings {
	return &GroupSettings{
		IsPublic:           false,
		RequireInvite:      true,
		AllowGuestMessages: false,
		MaxMembers:         100,
		MessageRetention:   30 * 24 * time.Hour, // 30 days
		Permissions: map[GroupRole][]Permission{
			RoleOwner: {
				PermissionSendMessage, PermissionDeleteMessage, PermissionAddMember,
				PermissionRemoveMember, PermissionChangeRole, PermissionManageGroup,
				PermissionViewMembers, PermissionViewHistory, PermissionCreateSubgroup,
				PermissionDeleteGroup,
			},
			RoleAdmin: {
				PermissionSendMessage, PermissionDeleteMessage, PermissionAddMember,
				PermissionRemoveMember, PermissionChangeRole, PermissionManageGroup,
				PermissionViewMembers, PermissionViewHistory, PermissionCreateSubgroup,
			},
			RoleModerator: {
				PermissionSendMessage, PermissionDeleteMessage, PermissionAddMember,
				PermissionViewMembers, PermissionViewHistory,
			},
			RoleMember: {
				PermissionSendMessage, PermissionViewMembers, PermissionViewHistory,
			},
			RoleGuest: {
				PermissionViewHistory,
			},
		},
	}
}

// CreateGroup creates a new group
func (gm *GroupManager) CreateGroup(id, name, createdBy string, settings *GroupSettings) (*Group, error) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	if _, exists := gm.groups[id]; exists {
		return nil, fmt.Errorf("group %s already exists", id)
	}

	if settings == nil {
		settings = DefaultGroupSettings()
	}

	group := &Group{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now().Unix(),
		CreatedBy: createdBy,
		Members:   make(map[string]*GroupMember),
		Settings:  settings,
		Metadata:  make(map[string]any),
	}

	// Add creator as owner
	group.Members[createdBy] = &GroupMember{
		Address:  createdBy,
		Role:     RoleOwner,
		JoinedAt: time.Now().Unix(),
		Status:   "active",
	}

	gm.groups[id] = group
	return group, nil
}

// GetGroup retrieves a group by ID
func (gm *GroupManager) GetGroup(id string) (*Group, error) {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	group, exists := gm.groups[id]
	if !exists {
		return nil, fmt.Errorf("group %s not found", id)
	}

	return group, nil
}

// DeleteGroup deletes a group
func (gm *GroupManager) DeleteGroup(id, requesterAddress string) error {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	group, exists := gm.groups[id]
	if !exists {
		return fmt.Errorf("group %s not found", id)
	}

	// Check permissions
	if !group.HasPermission(requesterAddress, PermissionDeleteGroup) {
		return fmt.Errorf("insufficient permissions to delete group")
	}

	delete(gm.groups, id)
	return nil
}

// ListGroups returns all groups
func (gm *GroupManager) ListGroups() []*Group {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	groups := make([]*Group, 0, len(gm.groups))
	for _, group := range gm.groups {
		groups = append(groups, group)
	}

	return groups
}

// AddMember adds a member to the group
func (g *Group) AddMember(address, invitedBy string, role GroupRole) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// Check if inviter has permission
	if !g.HasPermission(invitedBy, PermissionAddMember) {
		return fmt.Errorf("insufficient permissions to add member")
	}

	// Check if member already exists
	if _, exists := g.Members[address]; exists {
		return fmt.Errorf("member %s already exists in group", address)
	}

	// Check member limit
	if len(g.Members) >= g.Settings.MaxMembers {
		return fmt.Errorf("group has reached maximum member limit")
	}

	// Add member
	g.Members[address] = &GroupMember{
		Address:   address,
		Role:      role,
		JoinedAt:  time.Now().Unix(),
		InvitedBy: invitedBy,
		Status:    "active",
	}

	return nil
}

// RemoveMember removes a member from the group
func (g *Group) RemoveMember(address, requesterAddress string) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// Check if requester has permission
	if !g.HasPermission(requesterAddress, PermissionRemoveMember) {
		return fmt.Errorf("insufficient permissions to remove member")
	}

	// Check if member exists
	member, exists := g.Members[address]
	if !exists {
		return fmt.Errorf("member %s not found in group", address)
	}

	// Cannot remove owner
	if member.Role == RoleOwner {
		return fmt.Errorf("cannot remove group owner")
	}

	// Check role hierarchy (can't remove someone with equal or higher role)
	requesterMember := g.Members[requesterAddress]
	if !g.canModifyRole(requesterMember.Role, member.Role) {
		return fmt.Errorf("insufficient permissions to remove member with role %s", member.Role)
	}

	delete(g.Members, address)
	return nil
}

// ChangeRole changes a member's role
func (g *Group) ChangeRole(address, requesterAddress string, newRole GroupRole) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// Check if requester has permission
	if !g.HasPermission(requesterAddress, PermissionChangeRole) {
		return fmt.Errorf("insufficient permissions to change role")
	}

	// Check if member exists
	member, exists := g.Members[address]
	if !exists {
		return fmt.Errorf("member %s not found in group", address)
	}

	// Check role hierarchy
	requesterMember := g.Members[requesterAddress]
	if !g.canModifyRole(requesterMember.Role, member.Role) || !g.canModifyRole(requesterMember.Role, newRole) {
		return fmt.Errorf("insufficient permissions to change role")
	}

	// Cannot change owner role
	if member.Role == RoleOwner || newRole == RoleOwner {
		return fmt.Errorf("cannot change owner role")
	}

	member.Role = newRole
	return nil
}

// HasPermission checks if a member has a specific permission
func (g *Group) HasPermission(address string, permission Permission) bool {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	member, exists := g.Members[address]
	if !exists {
		return false
	}

	permissions, exists := g.Settings.Permissions[member.Role]
	if !exists {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}

	return false
}

// GetMember returns a member by address
func (g *Group) GetMember(address string) (*GroupMember, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	member, exists := g.Members[address]
	if !exists {
		return nil, fmt.Errorf("member %s not found", address)
	}

	// Return a copy to prevent external modification
	memberCopy := *member
	return &memberCopy, nil
}

// GetMembers returns all members
func (g *Group) GetMembers() []*GroupMember {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	members := make([]*GroupMember, 0, len(g.Members))
	for _, member := range g.Members {
		memberCopy := *member
		members = append(members, &memberCopy)
	}

	return members
}

// GetMembersByRole returns members with a specific role
func (g *Group) GetMembersByRole(role GroupRole) []*GroupMember {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	var members []*GroupMember
	for _, member := range g.Members {
		if member.Role == role {
			memberCopy := *member
			members = append(members, &memberCopy)
		}
	}

	return members
}

// canModifyRole checks if a role can modify another role
func (g *Group) canModifyRole(modifierRole, targetRole GroupRole) bool {
	roleHierarchy := map[GroupRole]int{
		RoleOwner:     5,
		RoleAdmin:     4,
		RoleModerator: 3,
		RoleMember:    2,
		RoleGuest:     1,
	}

	modifierLevel := roleHierarchy[modifierRole]
	targetLevel := roleHierarchy[targetRole]

	return modifierLevel > targetLevel
}

// ToJSON serializes a group to JSON
func (g *Group) ToJSON() ([]byte, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return json.Marshal(g)
}

// FromJSON deserializes a group from JSON
func FromJSON(data []byte) (*Group, error) {
	var group Group
	err := json.Unmarshal(data, &group)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal group: %w", err)
	}
	return &group, nil
}

// CreateGroupMessage creates a group management message
func CreateGroupMessage(groupID, action, actor string, data map[string]any) (*message.Message, error) {
	systemMsg := &message.SystemMessage{
		Type:      fmt.Sprintf("group:%s", action),
		Actor:     actor,
		GroupID:   groupID,
		Metadata:  data,
		Timestamp: time.Now().Unix(),
	}

	bodyData, err := json.Marshal(systemMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal system message: %w", err)
	}

	msg := &message.Message{
		From:      fmt.Sprintf("system#%s", extractDomain(groupID)),
		To:        []string{groupID},
		Type:      fmt.Sprintf("group:%s", action),
		Body:      string(bodyData),
		Timestamp: time.Now().Unix(),
		MessageID: fmt.Sprintf("group_%s_%d", action, time.Now().UnixNano()),
	}

	return msg, nil
}

// extractDomain extracts domain from group ID
func extractDomain(groupID string) string {
	parts := strings.Split(groupID, "#")
	if len(parts) > 1 {
		return parts[1]
	}
	return "localhost"
}
