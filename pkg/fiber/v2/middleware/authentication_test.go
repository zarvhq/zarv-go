package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func setupApp() *fiber.App {
	app := fiber.New()
	app.Use(Authenticate)
	app.Get("/profile", func(c *fiber.Ctx) error {
		profile := GetAuthProfile(c)
		return c.JSON(profile)
	})
	return app
}

func TestAuthenticate_InternalHeader(t *testing.T) {
	app := setupApp()
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	req.Header.Set("X-Internal", "internal")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAuthenticate_MissingHeaders(t *testing.T) {
	app := setupApp()
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAuthenticate_ValidHeaders(t *testing.T) {
	app := setupApp()
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	req.Header.Set("X-Issuer", "ultron-app")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-User-Id", "user1")
	req.Header.Set("X-Zarv-Role", "zarver")
	req.Header.Set("X-Access-Level", "admin")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetAuthProfile(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(localWorkspaceId, "ws1")
		c.Locals(localUserId, "user1")
		c.Locals(localRole, "zarver")
		c.Locals(localSource, "UI")
		c.Locals(localAccessLevel, "admin")
		return c.Next()
	})
	app.Get("/profile", func(c *fiber.Ctx) error {
		profile := GetAuthProfile(c)
		return c.JSON(profile)
	})

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAuthProfileMethods(t *testing.T) {
	profile := &AuthProfile{
		Role:        "zarver",
		AccessLevel: "admin",
	}
	assert.True(t, profile.IsZarvAdmin())
	assert.True(t, profile.IsUserAdmin())
	assert.False(t, profile.IsViewer())

	profile = &AuthProfile{
		Role:        "zarver",
		AccessLevel: "viewer",
	}
	assert.False(t, profile.IsZarvAdmin())
	assert.False(t, profile.IsUserAdmin())
	assert.True(t, profile.IsViewer())
}
