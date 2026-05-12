package role

// Tenant-level roles (User.Role, UserTenant.Role)
const (
	TenantRoleAdmin  = "admin"
	TenantRoleMember = "member"
	TenantRoleViewer = "viewer"
)

// Platform admin roles (PlatformAdmin.Role)
const (
	PlatformRoleSuperAdmin   = "super_admin"
	PlatformRoleSupport      = "support"
	PlatformRoleBillingAdmin = "billing_admin"
)

// API key permissions (APIKey.Permissions JSONB)
const (
	PermChat       = "chat"
	PermEmbeddings = "embeddings"
	PermAdmin      = "admin"
	PermManage     = "manage"
)

// TenantRoleLevel defines the privilege ordering. Higher value = more privileges.
var tenantRoleLevel = map[string]int{
	TenantRoleAdmin:  3,
	TenantRoleMember: 2,
	TenantRoleViewer: 1,
}

// PlatformRoleLevel defines the privilege ordering for platform roles.
var platformRoleLevel = map[string]int{
	PlatformRoleSuperAdmin:   3,
	PlatformRoleBillingAdmin: 2,
	PlatformRoleSupport:      1,
}

// PlatformRolePermissions maps each platform role to its default permissions.
// super_admin implicitly has all permissions (bypasses checks).
var PlatformRolePermissions = map[string][]string{
	PlatformRoleBillingAdmin: {"billing:read", "billing:write", "tenant:read"},
	PlatformRoleSupport:      {"tenant:read", "user:read"},
}

var validTenantRoles = map[string]bool{
	TenantRoleAdmin:  true,
	TenantRoleMember: true,
	TenantRoleViewer: true,
}

var validPlatformRoles = map[string]bool{
	PlatformRoleSuperAdmin:   true,
	PlatformRoleSupport:      true,
	PlatformRoleBillingAdmin: true,
}

// --- Tenant Role Helpers ---

// IsValidTenantRole checks if the given string is a valid tenant role.
func IsValidTenantRole(r string) bool {
	return validTenantRoles[r]
}

// RequireTenantRole returns an error if the role is not a valid tenant role.
func RequireTenantRole(r string) error {
	if !IsValidTenantRole(r) {
		return ErrInvalidTenantRole
	}
	return nil
}

// IsTenantAdmin returns true if the role is a tenant admin.
func IsTenantAdmin(r string) bool {
	return r == TenantRoleAdmin
}

// IsTenantMember returns true if the role is a tenant member.
func IsTenantMember(r string) bool {
	return r == TenantRoleMember
}

// IsTenantViewer returns true if the role is a tenant viewer.
func IsTenantViewer(r string) bool {
	return r == TenantRoleViewer
}

// IsValidRequestableRole checks if the role can be requested/invited (member or viewer, not admin).
func IsValidRequestableRole(r string) bool {
	return r == TenantRoleMember || r == TenantRoleViewer
}

// RequireRequestableRole returns an error if the role is not requestable.
func RequireRequestableRole(r string) error {
	if !IsValidRequestableRole(r) {
		return ErrRoleNotRequestable
	}
	return nil
}

// TenantRoleHigherOrEqual returns true if role a has privileges >= role b.
func TenantRoleHigherOrEqual(a, b string) bool {
	return tenantRoleLevel[a] >= tenantRoleLevel[b]
}

// TenantRoleStrictlyHigher returns true if role a has strictly higher privileges than role b.
func TenantRoleStrictlyHigher(a, b string) bool {
	return tenantRoleLevel[a] > tenantRoleLevel[b]
}

// MinTenantRole returns the minimum tenant role required to perform an operation.
// Returns true if role meets or exceeds the minimum requirement.
func MinTenantRole(role, minimum string) bool {
	return TenantRoleHigherOrEqual(role, minimum)
}

// ValidTenantRoles returns all valid tenant role strings.
func ValidTenantRoles() []string {
	return []string{TenantRoleAdmin, TenantRoleMember, TenantRoleViewer}
}

// --- Platform Role Helpers ---

// IsValidPlatformRole checks if the given string is a valid platform admin role.
func IsValidPlatformRole(r string) bool {
	return validPlatformRoles[r]
}

// RequirePlatformRole returns an error if the role is not a valid platform admin role.
func RequirePlatformRole(r string) error {
	if !IsValidPlatformRole(r) {
		return ErrInvalidPlatformRole
	}
	return nil
}

// IsSuperAdmin returns true if the role is super_admin.
func IsSuperAdmin(r string) bool {
	return r == PlatformRoleSuperAdmin
}

// IsSupport returns true if the role is support.
func IsSupport(r string) bool {
	return r == PlatformRoleSupport
}

// IsBillingAdmin returns true if the role is billing_admin.
func IsBillingAdmin(r string) bool {
	return r == PlatformRoleBillingAdmin
}

// PlatformRoleHigherOrEqual returns true if role a has privileges >= role b.
func PlatformRoleHigherOrEqual(a, b string) bool {
	return platformRoleLevel[a] >= platformRoleLevel[b]
}

// HasPlatformPermission checks if a platform admin with the given role and permissions JSONB string
// has a specific permission. super_admin automatically has all permissions.
func HasPlatformPermission(role string, permissionsJSON string, permission string) bool {
	if role == PlatformRoleSuperAdmin {
		return true
	}
	var perms []string
	if err := parsePermissionsJSON(permissionsJSON, &perms); err != nil {
		return false
	}
	for _, p := range perms {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}

// ValidPlatformRoles returns all valid platform admin role strings.
func ValidPlatformRoles() []string {
	return []string{PlatformRoleSuperAdmin, PlatformRoleSupport, PlatformRoleBillingAdmin}
}