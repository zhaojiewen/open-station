package role

import (
	"encoding/json"

	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

var (
	// ErrInvalidTenantRole is returned when an invalid tenant role string is used.
	ErrInvalidTenantRole = apperrors.NewAppError("ROLE_001", "invalid tenant role", nil)

	// ErrInvalidPlatformRole is returned when an invalid platform admin role string is used.
	ErrInvalidPlatformRole = apperrors.NewAppError("ROLE_002", "invalid platform admin role", nil)

	// ErrPermissionDenied is returned when a role lacks the required permission.
	ErrPermissionDenied = apperrors.NewAppError("ROLE_003", "permission denied for this role", nil)

	// ErrRoleNotRequestable is returned when trying to request/invite a non-requestable role (e.g., admin).
	ErrRoleNotRequestable = apperrors.NewAppError("ROLE_004", "role is not requestable, only member and viewer can be requested", nil)
)

// parsePermissionsJSON unmarshals a JSONB permissions string into a []string slice.
func parsePermissionsJSON(raw string, target *[]string) error {
	if raw == "" || raw == "[]" {
		return nil
	}
	return json.Unmarshal([]byte(raw), target)
}