package routes

import (
	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func authRoutes(users *controllers.UserController, auth *controllers.AuthController) chi.Router {
	r := chi.NewRouter()

	r.Post("/login", auth.Login)
	r.Post("/logout", auth.Logout)
	r.Get("/session", auth.Session)
	r.Head("/session", auth.Session)
	r.Post("/verify-email", users.VerifyEmail)
	r.Post("/verify-email/resend", users.ResendVerification)

	return r
}
