package middleware

import (
	"context"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
)

type id string

const idKey id = "requestID"

func GenerateLogID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currCtx := r.Context()
		newCtx := context.WithValue(currCtx, idKey, uuid.New().String())
		requestWithNewCtx := r.WithContext(newCtx)
		next.ServeHTTP(w, requestWithNewCtx)
	})
}

func NewLogRequest(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Context().Value(idKey)
			logger.Info("incoming request", "requestID", reqID)
			next.ServeHTTP(w, r)
		})
	}
}
