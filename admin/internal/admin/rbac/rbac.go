package rbac

import (
	"strings"
)

// Role represents a staff access tier.
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleOps       Role = "ops"
	RoleSupport   Role = "support"
	RoleMarketing Role = "marketing"
)

// Capability represents a discrete feature toggle which can be checked in handlers and templates.
type Capability string

const (
	CapDashboardOverview Capability = "dashboard.view"
	CapOrdersList        Capability = "orders.list"
	CapOrdersDetail      Capability = "orders.detail"
	CapOrderRefund       Capability = "orders.refund"
	CapShipmentsMonitor  Capability = "shipments.monitor"
	CapProductionQueues  Capability = "production.queues"
	CapCatalogManage     Capability = "catalog.manage"
	CapCatalogFonts      Capability = "catalog.fonts"
	CapContentManage     Capability = "content.manage"
	CapPromotionsManage  Capability = "promotions.manage"
	CapPromotionsUsage   Capability = "promotions.usage"
	CapReviewsModerate   Capability = "reviews.moderate"
	CapCustomersView     Capability = "customers.view"
	CapNotificationsFeed Capability = "notifications.feed"
	CapAuditLogView      Capability = "auditlogs.view"
	CapSystemTasks       Capability = "system.tasks"
	CapSystemCounters    Capability = "system.counters"
	CapStaffManage       Capability = "org.staff"
	CapProfileSelf       Capability = "profile.self"
	CapSearchGlobal      Capability = "search.global"
)

// capabilityRoles maps each capability to the roles permitted to access it.
var capabilityRoles = map[Capability]Roles{
	CapDashboardOverview: {RoleAdmin, RoleOps, RoleSupport, RoleMarketing},
	CapOrdersList:        {RoleAdmin, RoleOps, RoleSupport},
	CapOrdersDetail:      {RoleAdmin, RoleOps, RoleSupport},
	CapOrderRefund:       {RoleAdmin, RoleSupport},
	CapShipmentsMonitor:  {RoleAdmin, RoleOps},
	CapProductionQueues:  {RoleAdmin, RoleOps},
	CapCatalogManage:     {RoleAdmin, RoleOps, RoleMarketing},
	CapCatalogFonts:      {RoleAdmin, RoleMarketing},
	CapContentManage:     {RoleAdmin, RoleMarketing},
	CapPromotionsManage:  {RoleAdmin, RoleMarketing},
	CapPromotionsUsage:   {RoleAdmin, RoleMarketing},
	CapReviewsModerate:   {RoleAdmin, RoleSupport, RoleMarketing},
	CapCustomersView:     {RoleAdmin, RoleOps, RoleSupport},
	CapNotificationsFeed: {RoleAdmin, RoleOps, RoleSupport},
	CapAuditLogView:      {RoleAdmin},
	CapSystemTasks:       {RoleAdmin, RoleOps},
	CapSystemCounters:    {RoleAdmin},
	CapStaffManage:       {RoleAdmin},
	CapProfileSelf:       {RoleAdmin, RoleOps, RoleSupport, RoleMarketing},
	CapSearchGlobal:      {RoleAdmin, RoleOps, RoleSupport},
}

// Roles captures a list of roles and exposes intersection checks used for RBAC evaluation.
type Roles []Role

// Has returns true if the provided role exists in the set.
func (rs Roles) Has(role Role) bool {
	for _, r := range rs {
		if r == role {
			return true
		}
	}
	return false
}

// Intersects returns true if any role in the candidate slice is also present in the set.
func (rs Roles) Intersects(candidate Roles) bool {
	for _, role := range candidate {
		if rs.Has(role) {
			return true
		}
	}
	return false
}

// NormaliseRoles converts raw role strings into canonical Role values.
func NormaliseRoles(raw []string) Roles {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[Role]struct{}, len(raw))
	roles := make(Roles, 0, len(raw))
	for _, val := range raw {
		role := Role(strings.ToLower(strings.TrimSpace(val)))
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	return roles
}

// RolesForCapability returns the configured roles able to access the capability.
func RolesForCapability(cap Capability) Roles {
	if roles, ok := capabilityRoles[cap]; ok {
		return roles
	}
	return nil
}

// HasRole reports whether the user roles include the required role. Admin users always satisfy checks.
func HasRole(userRoles []string, required Role) bool {
	roles := NormaliseRoles(userRoles)
	if roles.Has(RoleAdmin) {
		return true
	}
	return roles.Has(required)
}

// HasAnyRole reports whether the intersection between user roles and required roles is non-empty.
func HasAnyRole(userRoles []string, required Roles) bool {
	roles := NormaliseRoles(userRoles)
	if roles.Has(RoleAdmin) {
		return true
	}
	return required.Intersects(roles)
}

// HasCapability reports whether the provided roles grant access to the capability.
// Admin users implicitly possess every capability.
func HasCapability(userRoles []string, capability Capability) bool {
	if capability == "" {
		return true
	}
	allowed := RolesForCapability(capability)
	if len(allowed) == 0 {
		return false
	}
	roles := NormaliseRoles(userRoles)
	if roles.Has(RoleAdmin) {
		return true
	}
	return allowed.Intersects(roles)
}

// CapabilitiesForRoles enumerates the capabilities accessible to the provided user roles.
func CapabilitiesForRoles(userRoles []string) map[Capability]bool {
	roles := NormaliseRoles(userRoles)
	caps := make(map[Capability]bool, len(capabilityRoles))
	if roles.Has(RoleAdmin) {
		for cap := range capabilityRoles {
			caps[cap] = true
		}
		return caps
	}
	for capability, allowed := range capabilityRoles {
		if allowed.Intersects(roles) {
			caps[capability] = true
		}
	}
	return caps
}
