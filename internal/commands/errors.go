package commands

import "fmt"

// ErrUnauthorized is returned when a user lacks required permissions to execute a command
type ErrUnauthorized struct {
	Username        string
	PermissionLevel string
	RequiredLevels  []string
}

func (e *ErrUnauthorized) Error() string {
	return fmt.Sprintf("permission denied: user '%s' does not have required permissions\n"+
		"  → Current permission level: %s\n"+
		"  → Required: write, admin, or maintain access to repository",
		e.Username, e.PermissionLevel)
}

// NewErrUnauthorized creates a new unauthorized error
func NewErrUnauthorized(username, permissionLevel string) *ErrUnauthorized {
	return &ErrUnauthorized{
		Username:        username,
		PermissionLevel: permissionLevel,
		RequiredLevels:  []string{"write", "admin", "maintain"},
	}
}
