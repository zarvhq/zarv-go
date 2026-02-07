// Package middleware provides authentication and authorization middlewares for Fiber applications.
package middleware

import (
	"github.com/gofiber/fiber/v2"
)

const (
	// local keys
	localWorkspaceId string = "workspaceId"
	localUserId      string = "userId"
	localSource      string = "source"
	localRole        string = "role"
	localAccessLevel string = "accessLevel"

	// header keys
	headerXIssuer      string = "X-Issuer"       // zarv auth issuer
	headerXWorkspaceId string = "X-Workspace-Id" // zarv workspace id
	headerXUserId      string = "X-User-Id"      // zarv user id
	headerXZarvRole    string = "X-Zarv-Role"    // zarv role
	headerXAccessLevel string = "X-Access-Level" // zarv access level
	headerXInternal    string = "X-Internal"     // zarv internal request indicator

	// header values
	headerZarverRole string = "zarver" // zarv role

	accessLevelViewer     string = "viewer"     // zarv access level viewer
	accessLevelUser       string = "user"       // zarv access level user
	accessLevelSupervisor string = "supervisor" // zarv access level supervisor
	accessLevelAdmin      string = "admin"      // zarv access level admin

	internalSource        string = "internal" // zarv internal request source
	verificationSourceUI  string = "UI"
	verificationSourceAPI string = "API"
)

// AuthProfile contains the authenticated user's profile information
// extracted from request headers by the Authenticate middleware.
type AuthProfile struct {
	// WorkspaceID is the Zarv workspace identifier
	WorkspaceID string
	// UserID is the Zarv user identifier
	UserID string
	// Role is the user's Zarv role (e.g., "zarver")
	Role string
	// Source indicates the request origin ("UI", "API", or "internal")
	Source string
	// AccessLevel defines the user's permission level ("viewer", "user", "supervisor", "admin")
	AccessLevel string
}

// Authenticate is a Fiber middleware that validates authentication headers
// and stores user profile information in the request context.
//
// Required headers:
//   - X-Issuer: Authentication issuer identifier
//   - X-Workspace-Id: Zarv workspace ID
//   - X-User-Id: Zarv user ID
//   - X-Zarv-Role: User role
//   - X-Access-Level: User access level
//
// Optional headers:
//   - X-Internal: Indicates an internal service request
//
// Returns 400 Bad Request if any required header is missing.
func Authenticate(c *fiber.Ctx) error {
	internal := c.Get(headerXInternal)
	if internal != "" {
		c.Locals(localSource, internal)
		c.Locals(localWorkspaceId, internalSource)
		c.Locals(localUserId, internalSource)
		c.Locals(localRole, headerZarverRole)
		c.Locals(localAccessLevel, accessLevelAdmin)

		return c.Next()
	}

	issuer := c.Get(headerXIssuer)
	if issuer == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unauthorized issuer",
		})
	}

	workspaceId := c.Get(headerXWorkspaceId)
	if workspaceId == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unauthorized workspace",
		})
	}

	userId := c.Get(headerXUserId)
	if userId == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unauthorized user",
		})
	}

	role := c.Get(headerXZarvRole)
	if role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unauthorized role",
		})
	}

	accessLevel := c.Get(headerXAccessLevel)
	if accessLevel == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unauthorized access level",
		})
	}

	var source string

	switch issuer {
	case "ultron-app", "vision-app":
		source = verificationSourceUI
	default:
		source = verificationSourceAPI
	}

	c.Locals(localSource, source)
	c.Locals(localWorkspaceId, workspaceId)
	c.Locals(localUserId, userId)
	c.Locals(localRole, role)
	c.Locals(localAccessLevel, accessLevel)

	return c.Next()
}

// GetAuthProfile retrieves the authenticated user's profile from the request context.
// This function should be called after the Authenticate middleware has run.
// Returns a pointer to AuthProfile with the user's authentication information.
func GetAuthProfile(c *fiber.Ctx) *AuthProfile {
	workspaceID, _ := c.Locals(localWorkspaceId).(string)
	userID, _ := c.Locals(localUserId).(string)
	role, _ := c.Locals(localRole).(string)
	source, _ := c.Locals(localSource).(string)
	accessLevel, _ := c.Locals(localAccessLevel).(string)

	return &AuthProfile{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
		Source:      source,
		AccessLevel: accessLevel,
	}
}

// IsZarvAdmin checks if the user is a Zarv administrator.
// Returns true if the user has the "zarver" role and "admin" or "supervisor" access level.
// Zarv admins can override workspace restrictions.
func (profile *AuthProfile) IsZarvAdmin() bool {
	if profile.Role != headerZarverRole { // only zarver role can override workspace id
		return false
	}
	// zarver, admin and supervisor can override workspace id
	return profile.AccessLevel == accessLevelAdmin || profile.AccessLevel == accessLevelSupervisor
}

// IsUserAdmin checks if the user has administrative privileges in their workspace.
// Returns true if the user has "admin" or "supervisor" access level.
func (profile *AuthProfile) IsUserAdmin() bool {
	return profile.AccessLevel == accessLevelAdmin || profile.AccessLevel == accessLevelSupervisor
}

// IsViewer checks if the user has only view-only permissions.
// Returns true if the user's access level is "viewer".
func (profile *AuthProfile) IsViewer() bool {
	return profile.AccessLevel == accessLevelViewer
}
