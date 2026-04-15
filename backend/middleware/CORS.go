package middleware

import (
	"github.com/rs/cors"
	"net/http"
)

func CorsHandler(next http.Handler) http.Handler {

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://localhost:3000", "https://localhost:3001", "https://localhost:3002"},
		AllowCredentials: true,
		Debug: true,
	})
	return c.Handler(next)
}
