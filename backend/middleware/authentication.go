package middleware

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
)

func NewAuthMiddleware(key string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headerValue := r.Header.Get("Authorization")

			if headerValue == "" {
				http.Error(w, "Authorization Empty", http.StatusBadRequest)
				return
			}

			token := strings.TrimPrefix(headerValue, "Bearer ")

			parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}

				return []byte(key), nil
			})

			if !parsedToken.Valid || err != nil {
				http.Error(w, "invalid credentials", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

}
