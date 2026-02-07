// Package middleware provides authentication and authorization middlewares for Fiber applications.
//
// The middleware package offers robust authentication and authorization capabilities
// for Fiber web applications using Zarv's authentication system.
//
// Example usage:
//
//	import (
//		"github.com/gofiber/fiber/v2"
//		"github.com/zarvhq/zarv-go/pkg/fiber/v2/middleware"
//	)
//
//	func main() {
//		app := fiber.New()
//		app.Use(middleware.Authenticate)
//
//		app.Get("/api/resource", func(c *fiber.Ctx) error {
//			profile := middleware.GetAuthProfile(c)
//			if profile.IsViewer() {
//				return c.Status(403).JSON(fiber.Map{"error": "Permission denied"})
//			}
//			return c.JSON(profile)
//		})
//
//		app.Listen(":3000")
//	}
package middleware
