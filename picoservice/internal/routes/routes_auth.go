package routes

import (
	"github.com/go-chi/chi/v5"

	"picoservice/internal/controllers"
)

func authRoutes(users *controllers.UserController) chi.Router {
	r := chi.NewRouter()

	r.Post("/verify-email", users.VerifyEmail)
	r.Post("/verify-email/resend", users.ResendVerification)

	return r
}
