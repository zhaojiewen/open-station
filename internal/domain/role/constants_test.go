package role

import (
	"testing"
)

func TestTenantRoleConstants(t *testing.T) {
	if TenantRoleAdmin != "admin" {
		t.Errorf("TenantRoleAdmin = %q, want %q", TenantRoleAdmin, "admin")
	}
	if TenantRoleMember != "member" {
		t.Errorf("TenantRoleMember = %q, want %q", TenantRoleMember, "member")
	}
	if TenantRoleViewer != "viewer" {
		t.Errorf("TenantRoleViewer = %q, want %q", TenantRoleViewer, "viewer")
	}
}

func TestPlatformRoleConstants(t *testing.T) {
	if PlatformRoleSuperAdmin != "super_admin" {
		t.Errorf("PlatformRoleSuperAdmin = %q, want %q", PlatformRoleSuperAdmin, "super_admin")
	}
	if PlatformRoleSupport != "support" {
		t.Errorf("PlatformRoleSupport = %q, want %q", PlatformRoleSupport, "support")
	}
	if PlatformRoleBillingAdmin != "billing_admin" {
		t.Errorf("PlatformRoleBillingAdmin = %q, want %q", PlatformRoleBillingAdmin, "billing_admin")
	}
}

func TestPermConstants(t *testing.T) {
	if PermChat != "chat" {
		t.Errorf("PermChat = %q, want %q", PermChat, "chat")
	}
	if PermEmbeddings != "embeddings" {
		t.Errorf("PermEmbeddings = %q, want %q", PermEmbeddings, "embeddings")
	}
	if PermAdmin != "admin" {
		t.Errorf("PermAdmin = %q, want %q", PermAdmin, "admin")
	}
	if PermManage != "manage" {
		t.Errorf("PermManage = %q, want %q", PermManage, "manage")
	}
}

func TestIsValidTenantRole(t *testing.T) {
	tests := []struct {
		role  string
		valid bool
	}{
		{"admin", true},
		{"member", true},
		{"viewer", true},
		{"", false},
		{"superadmin", false},
		{"Admin", false},
		{"ADMIN", false},
		{"super_admin", false},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			if got := IsValidTenantRole(tt.role); got != tt.valid {
				t.Errorf("IsValidTenantRole(%q) = %v, want %v", tt.role, got, tt.valid)
			}
		})
	}
}

func TestRequireTenantRole(t *testing.T) {
	if err := RequireTenantRole(TenantRoleAdmin); err != nil {
		t.Errorf("RequireTenantRole(admin) should not error, got %v", err)
	}
	if err := RequireTenantRole(TenantRoleMember); err != nil {
		t.Errorf("RequireTenantRole(member) should not error, got %v", err)
	}
	if err := RequireTenantRole("invalid"); err != ErrInvalidTenantRole {
		t.Errorf("RequireTenantRole(invalid) = %v, want %v", err, ErrInvalidTenantRole)
	}
}

func TestIsTenantAdmin(t *testing.T) {
	if !IsTenantAdmin("admin") {
		t.Error("IsTenantAdmin(admin) should be true")
	}
	if IsTenantAdmin("member") {
		t.Error("IsTenantAdmin(member) should be false")
	}
	if IsTenantAdmin("viewer") {
		t.Error("IsTenantAdmin(viewer) should be false")
	}
	if IsTenantAdmin("") {
		t.Error("IsTenantAdmin(\"\") should be false")
	}
}

func TestIsTenantMember(t *testing.T) {
	if IsTenantMember("admin") {
		t.Error("IsTenantMember(admin) should be false")
	}
	if !IsTenantMember("member") {
		t.Error("IsTenantMember(member) should be true")
	}
	if IsTenantMember("viewer") {
		t.Error("IsTenantMember(viewer) should be false")
	}
}

func TestIsTenantViewer(t *testing.T) {
	if IsTenantViewer("admin") {
		t.Error("IsTenantViewer(admin) should be false")
	}
	if IsTenantViewer("member") {
		t.Error("IsTenantViewer(member) should be false")
	}
	if !IsTenantViewer("viewer") {
		t.Error("IsTenantViewer(viewer) should be true")
	}
}

func TestTenantRoleHigherOrEqual(t *testing.T) {
	tests := []struct {
		a, b   string
		result bool
	}{
		{TenantRoleAdmin, TenantRoleAdmin, true},
		{TenantRoleAdmin, TenantRoleMember, true},
		{TenantRoleAdmin, TenantRoleViewer, true},
		{TenantRoleMember, TenantRoleAdmin, false},
		{TenantRoleMember, TenantRoleMember, true},
		{TenantRoleMember, TenantRoleViewer, true},
		{TenantRoleViewer, TenantRoleAdmin, false},
		{TenantRoleViewer, TenantRoleMember, false},
		{TenantRoleViewer, TenantRoleViewer, true},
		{"invalid", TenantRoleViewer, false},
		{TenantRoleAdmin, "invalid", true}, // invalid role maps to level 0, so any valid role >= 0
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			if got := TenantRoleHigherOrEqual(tt.a, tt.b); got != tt.result {
				t.Errorf("TenantRoleHigherOrEqual(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.result)
			}
		})
	}
}

func TestTenantRoleStrictlyHigher(t *testing.T) {
	tests := []struct {
		a, b   string
		result bool
	}{
		{TenantRoleAdmin, TenantRoleAdmin, false},
		{TenantRoleAdmin, TenantRoleMember, true},
		{TenantRoleAdmin, TenantRoleViewer, true},
		{TenantRoleMember, TenantRoleAdmin, false},
		{TenantRoleMember, TenantRoleMember, false},
		{TenantRoleMember, TenantRoleViewer, true},
		{TenantRoleViewer, TenantRoleAdmin, false},
		{TenantRoleViewer, TenantRoleMember, false},
		{TenantRoleViewer, TenantRoleViewer, false},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			if got := TenantRoleStrictlyHigher(tt.a, tt.b); got != tt.result {
				t.Errorf("TenantRoleStrictlyHigher(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.result)
			}
		})
	}
}

func TestMinTenantRole(t *testing.T) {
	if !MinTenantRole(TenantRoleAdmin, TenantRoleAdmin) {
		t.Error("admin should meet admin minimum")
	}
	if !MinTenantRole(TenantRoleAdmin, TenantRoleMember) {
		t.Error("admin should meet member minimum")
	}
	if !MinTenantRole(TenantRoleMember, TenantRoleMember) {
		t.Error("member should meet member minimum")
	}
	if MinTenantRole(TenantRoleViewer, TenantRoleAdmin) {
		t.Error("viewer should not meet admin minimum")
	}
	if MinTenantRole(TenantRoleMember, TenantRoleAdmin) {
		t.Error("member should not meet admin minimum")
	}
}

func TestIsValidRequestableRole(t *testing.T) {
	tests := []struct {
		role   string
		expect bool
	}{
		{TenantRoleMember, true},
		{TenantRoleViewer, true},
		{TenantRoleAdmin, false},
		{"", false},
		{"superadmin", false},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			if got := IsValidRequestableRole(tt.role); got != tt.expect {
				t.Errorf("IsValidRequestableRole(%q) = %v, want %v", tt.role, got, tt.expect)
			}
		})
	}
}

func TestRequireRequestableRole(t *testing.T) {
	if err := RequireRequestableRole(TenantRoleMember); err != nil {
		t.Errorf("RequireRequestableRole(member) should not error, got %v", err)
	}
	if err := RequireRequestableRole(TenantRoleViewer); err != nil {
		t.Errorf("RequireRequestableRole(viewer) should not error, got %v", err)
	}
	if err := RequireRequestableRole(TenantRoleAdmin); err != ErrRoleNotRequestable {
		t.Errorf("RequireRequestableRole(admin) = %v, want %v", err, ErrRoleNotRequestable)
	}
	if err := RequireRequestableRole(""); err != ErrRoleNotRequestable {
		t.Errorf(`RequireRequestableRole("") = %v, want %v`, err, ErrRoleNotRequestable)
	}
}

func TestValidTenantRoles(t *testing.T) {
	roles := ValidTenantRoles()
	if len(roles) != 3 {
		t.Errorf("expected 3 roles, got %d", len(roles))
	}
	seen := make(map[string]bool)
	for _, r := range roles {
		if seen[r] {
			t.Errorf("duplicate role: %s", r)
		}
		seen[r] = true
	}
}

func TestIsValidPlatformRole(t *testing.T) {
	tests := []struct {
		role  string
		valid bool
	}{
		{PlatformRoleSuperAdmin, true},
		{PlatformRoleSupport, true},
		{PlatformRoleBillingAdmin, true},
		{"", false},
		{"admin", false},
		{"superadmin", false},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			if got := IsValidPlatformRole(tt.role); got != tt.valid {
				t.Errorf("IsValidPlatformRole(%q) = %v, want %v", tt.role, got, tt.valid)
			}
		})
	}
}

func TestRequirePlatformRole(t *testing.T) {
	if err := RequirePlatformRole(PlatformRoleSuperAdmin); err != nil {
		t.Errorf("RequirePlatformRole(super_admin) should not error, got %v", err)
	}
	if err := RequirePlatformRole("invalid"); err != ErrInvalidPlatformRole {
		t.Errorf("RequirePlatformRole(invalid) = %v, want %v", err, ErrInvalidPlatformRole)
	}
}

func TestIsSuperAdmin(t *testing.T) {
	if !IsSuperAdmin(PlatformRoleSuperAdmin) {
		t.Error("IsSuperAdmin(super_admin) should be true")
	}
	if IsSuperAdmin(PlatformRoleSupport) {
		t.Error("IsSuperAdmin(support) should be false")
	}
	if IsSuperAdmin(PlatformRoleBillingAdmin) {
		t.Error("IsSuperAdmin(billing_admin) should be false")
	}
}

func TestIsSupport(t *testing.T) {
	if IsSupport(PlatformRoleSuperAdmin) {
		t.Error("IsSupport(super_admin) should be false")
	}
	if !IsSupport(PlatformRoleSupport) {
		t.Error("IsSupport(support) should be true")
	}
	if IsSupport(PlatformRoleBillingAdmin) {
		t.Error("IsSupport(billing_admin) should be false")
	}
}

func TestIsBillingAdmin(t *testing.T) {
	if IsBillingAdmin(PlatformRoleSuperAdmin) {
		t.Error("IsBillingAdmin(super_admin) should be false")
	}
	if IsBillingAdmin(PlatformRoleSupport) {
		t.Error("IsBillingAdmin(support) should be false")
	}
	if !IsBillingAdmin(PlatformRoleBillingAdmin) {
		t.Error("IsBillingAdmin(billing_admin) should be true")
	}
}

func TestPlatformRoleHigherOrEqual(t *testing.T) {
	tests := []struct {
		a, b   string
		result bool
	}{
		{PlatformRoleSuperAdmin, PlatformRoleSuperAdmin, true},
		{PlatformRoleSuperAdmin, PlatformRoleBillingAdmin, true},
		{PlatformRoleSuperAdmin, PlatformRoleSupport, true},
		{PlatformRoleBillingAdmin, PlatformRoleSuperAdmin, false},
		{PlatformRoleBillingAdmin, PlatformRoleBillingAdmin, true},
		{PlatformRoleBillingAdmin, PlatformRoleSupport, true},
		{PlatformRoleSupport, PlatformRoleSuperAdmin, false},
		{PlatformRoleSupport, PlatformRoleBillingAdmin, false},
		{PlatformRoleSupport, PlatformRoleSupport, true},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			if got := PlatformRoleHigherOrEqual(tt.a, tt.b); got != tt.result {
				t.Errorf("PlatformRoleHigherOrEqual(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.result)
			}
		})
	}
}

func TestHasPlatformPermission(t *testing.T) {
	tests := []struct {
		name        string
		role        string
		permissions string
		check       string
		want        bool
	}{
		{
			name:        "super_admin has all permissions",
			role:        PlatformRoleSuperAdmin,
			permissions: "[]",
			check:       "any_permission",
			want:        true,
		},
		{
			name:        "explicit permission granted",
			role:        PlatformRoleSupport,
			permissions: `["tenant:read", "user:read"]`,
			check:       "tenant:read",
			want:        true,
		},
		{
			name:        "explicit permission denied",
			role:        PlatformRoleSupport,
			permissions: `["tenant:read"]`,
			check:       "admin:write",
			want:        false,
		},
		{
			name:        "wildcard grants all",
			role:        PlatformRoleBillingAdmin,
			permissions: `["*"]`,
			check:       "anything",
			want:        true,
		},
		{
			name:        "empty permissions",
			role:        PlatformRoleSupport,
			permissions: "[]",
			check:       "read",
			want:        false,
		},
		{
			name:        "empty string permissions",
			role:        PlatformRoleSupport,
			permissions: "",
			check:       "read",
			want:        false,
		},
		{
			name:        "invalid JSON defaults to no permission",
			role:        PlatformRoleSupport,
			permissions: "{invalid}",
			check:       "read",
			want:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasPlatformPermission(tt.role, tt.permissions, tt.check); got != tt.want {
				t.Errorf("HasPlatformPermission(%q, %q, %q) = %v, want %v",
					tt.role, tt.permissions, tt.check, got, tt.want)
			}
		})
	}
}

func TestPlatformRolePermissions(t *testing.T) {
	if perms, ok := PlatformRolePermissions[PlatformRoleBillingAdmin]; !ok || len(perms) == 0 {
		t.Error("billing_admin should have default permissions defined")
	}
	if perms, ok := PlatformRolePermissions[PlatformRoleSupport]; !ok || len(perms) == 0 {
		t.Error("support should have default permissions defined")
	}
	if _, ok := PlatformRolePermissions[PlatformRoleSuperAdmin]; ok {
		t.Error("super_admin should not have default permissions (implicit all-access)")
	}
}

func TestValidPlatformRoles(t *testing.T) {
	roles := ValidPlatformRoles()
	if len(roles) != 3 {
		t.Errorf("expected 3 roles, got %d", len(roles))
	}
	seen := make(map[string]bool)
	for _, r := range roles {
		if seen[r] {
			t.Errorf("duplicate role: %s", r)
		}
		seen[r] = true
	}
}