package auth

import (
	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App) {
	group := app.Group("/api/auth")

	group.Post("/sign-in", signInController)
	group.Post("/sign-up", signUpController)
}