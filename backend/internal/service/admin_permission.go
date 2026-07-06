package service

import (
	"context"
	"fmt"
	"sort"

	infraerrors "github.com/Wei-Shaw/nexus/internal/pkg/errors"
)

type AdminPermissionAction string
type AdminPermissionResource string

const (
	AdminActionView    AdminPermissionAction = "view"
	AdminActionCreate  AdminPermissionAction = "create"
	AdminActionUpdate  AdminPermissionAction = "update"
	AdminActionDelete  AdminPermissionAction = "delete"
	AdminActionExport  AdminPermissionAction = "export"
	AdminActionExecute AdminPermissionAction = "execute"
)

const (
	AdminResourceDashboard              AdminPermissionResource = "dashboard"
	AdminResourceOps                    AdminPermissionResource = "ops"
	AdminResourceUsers                  AdminPermissionResource = "users"
	AdminResourceGroups                 AdminPermissionResource = "groups"
	AdminResourceChannels               AdminPermissionResource = "channels"
	AdminResourceChannelMonitor         AdminPermissionResource = "channel_monitor"
	AdminResourceSubscriptions          AdminPermissionResource = "subscriptions"
	AdminResourceAccounts               AdminPermissionResource = "accounts"
	AdminResourceAnnouncements          AdminPermissionResource = "announcements"
	AdminResourceProxies                AdminPermissionResource = "proxies"
	AdminResourceRiskControl            AdminPermissionResource = "risk_control"
	AdminResourceRedeemCodes            AdminPermissionResource = "redeem_codes"
	AdminResourcePromoCodes             AdminPermissionResource = "promo_codes"
	AdminResourceAffiliates             AdminPermissionResource = "affiliates"
	AdminResourceOrders                 AdminPermissionResource = "orders"
	AdminResourceUsage                  AdminPermissionResource = "usage"
	AdminResourceSettings               AdminPermissionResource = "settings"
	AdminResourceSystem                 AdminPermissionResource = "system"
	AdminResourceDataManagement         AdminPermissionResource = "data_management"
	AdminResourceBackups                AdminPermissionResource = "backups"
	AdminResourceUserAttributes         AdminPermissionResource = "user_attributes"
	AdminResourceErrorPassthroughRules  AdminPermissionResource = "error_passthrough_rules"
	AdminResourceTLSFingerprintProfiles AdminPermissionResource = "tls_fingerprint_profiles"
	AdminResourceScheduledTests         AdminPermissionResource = "scheduled_tests"
	AdminResourceAdminPermissions       AdminPermissionResource = "admin_permissions"
)

type AdminPermission struct {
	Resource AdminPermissionResource `json:"resource"`
	Actions  []AdminPermissionAction `json:"actions"`
}

type AdminPermissionDefinition struct {
	Resource       AdminPermissionResource `json:"resource"`
	Label          string                  `json:"label"`
	Actions        []AdminPermissionAction `json:"actions"`
	SuperAdminOnly bool                    `json:"super_admin_only"`
}

var allAdminActions = []AdminPermissionAction{
	AdminActionView,
	AdminActionCreate,
	AdminActionUpdate,
	AdminActionDelete,
	AdminActionExport,
	AdminActionExecute,
}

var adminPermissionDefinitions = []AdminPermissionDefinition{
	{Resource: AdminResourceDashboard, Label: "Dashboard", Actions: []AdminPermissionAction{AdminActionView, AdminActionExecute}},
	{Resource: AdminResourceOps, Label: "Ops", Actions: allAdminActions},
	{Resource: AdminResourceUsers, Label: "Users", Actions: allAdminActions},
	{Resource: AdminResourceGroups, Label: "Groups", Actions: allAdminActions},
	{Resource: AdminResourceChannels, Label: "Channels", Actions: allAdminActions},
	{Resource: AdminResourceChannelMonitor, Label: "Channel Monitor", Actions: allAdminActions},
	{Resource: AdminResourceSubscriptions, Label: "Subscriptions", Actions: allAdminActions},
	{Resource: AdminResourceAccounts, Label: "Accounts", Actions: allAdminActions},
	{Resource: AdminResourceAnnouncements, Label: "Announcements", Actions: allAdminActions},
	{Resource: AdminResourceProxies, Label: "Proxies", Actions: allAdminActions},
	{Resource: AdminResourceRiskControl, Label: "Risk Control", Actions: allAdminActions},
	{Resource: AdminResourceRedeemCodes, Label: "Redeem Codes", Actions: allAdminActions},
	{Resource: AdminResourcePromoCodes, Label: "Promo Codes", Actions: allAdminActions},
	{Resource: AdminResourceAffiliates, Label: "Affiliates", Actions: allAdminActions},
	{Resource: AdminResourceOrders, Label: "Orders", Actions: allAdminActions},
	{Resource: AdminResourceUsage, Label: "Usage", Actions: []AdminPermissionAction{AdminActionView, AdminActionExport, AdminActionDelete, AdminActionExecute}},
	{Resource: AdminResourceSettings, Label: "Settings", Actions: allAdminActions, SuperAdminOnly: true},
	{Resource: AdminResourceSystem, Label: "System", Actions: allAdminActions, SuperAdminOnly: true},
	{Resource: AdminResourceDataManagement, Label: "Data Management", Actions: allAdminActions},
	{Resource: AdminResourceBackups, Label: "Backups", Actions: allAdminActions},
	{Resource: AdminResourceUserAttributes, Label: "User Attributes", Actions: allAdminActions},
	{Resource: AdminResourceErrorPassthroughRules, Label: "Error Passthrough Rules", Actions: allAdminActions},
	{Resource: AdminResourceTLSFingerprintProfiles, Label: "TLS Fingerprint Profiles", Actions: allAdminActions},
	{Resource: AdminResourceScheduledTests, Label: "Scheduled Tests", Actions: allAdminActions},
	{Resource: AdminResourceAdminPermissions, Label: "Admin Permissions", Actions: allAdminActions, SuperAdminOnly: true},
}

func AdminPermissionRegistry() []AdminPermissionDefinition {
	out := make([]AdminPermissionDefinition, len(adminPermissionDefinitions))
	copy(out, adminPermissionDefinitions)
	return out
}

func ValidateAdminPermissions(perms []AdminPermission) error {
	defs := make(map[AdminPermissionResource]AdminPermissionDefinition, len(adminPermissionDefinitions))
	for _, def := range adminPermissionDefinitions {
		defs[def.Resource] = def
	}
	for i := range perms {
		perm := &perms[i]
		def, ok := defs[perm.Resource]
		if !ok {
			return infraerrors.BadRequest("INVALID_ADMIN_PERMISSION", fmt.Sprintf("invalid admin permission resource: %s", perm.Resource))
		}
		if def.SuperAdminOnly {
			return infraerrors.Forbidden("SUPER_ADMIN_REQUIRED", fmt.Sprintf("resource requires super administrator: %s", perm.Resource))
		}
		allowed := make(map[AdminPermissionAction]struct{}, len(def.Actions))
		for _, action := range def.Actions {
			allowed[action] = struct{}{}
		}
		hasView := false
		seen := map[AdminPermissionAction]struct{}{}
		for _, action := range perm.Actions {
			if _, ok := allowed[action]; !ok {
				return infraerrors.BadRequest("INVALID_ADMIN_PERMISSION", fmt.Sprintf("invalid admin permission action: %s:%s", perm.Resource, action))
			}
			if _, exists := seen[action]; exists {
				return infraerrors.BadRequest("INVALID_ADMIN_PERMISSION", fmt.Sprintf("duplicate admin permission action: %s:%s", perm.Resource, action))
			}
			seen[action] = struct{}{}
			if action == AdminActionView {
				hasView = true
			}
		}
		if len(perm.Actions) > 0 && !hasView {
			return infraerrors.BadRequest("INVALID_ADMIN_PERMISSION", fmt.Sprintf("admin permission requires view action: %s", perm.Resource))
		}
		sort.Slice(perm.Actions, func(i, j int) bool { return perm.Actions[i] < perm.Actions[j] })
	}
	return nil
}

type AdminPermissionRepository interface {
	ListByUserID(ctx context.Context, userID int64) ([]AdminPermission, error)
	ReplaceForUser(ctx context.Context, userID int64, permissions []AdminPermission) error
	DeleteForUser(ctx context.Context, userID int64) error
	HasPermission(ctx context.Context, userID int64, resource AdminPermissionResource, action AdminPermissionAction) (bool, error)
}
