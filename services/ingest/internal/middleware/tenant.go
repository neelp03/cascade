// Package middleware provides HTTP middleware for the ingest service.
package middleware

import (
	"net/http"

	"github.com/cascade-analytics/cascade/services/ingest/internal/tenant"
)

// RequireTenantID reads X-Tenant-ID from the request header, injects it into
// the context, and rejects requests without it. A5 (Auth) will replace this
// header-based stub with JWT-derived tenant context in M4.
func RequireTenantID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Tenant-ID")
		if id == "" {
			http.Error(w, `{"error":"X-Tenant-ID header required"}`, http.StatusUnauthorized)
			return
		}
		ctx := tenant.WithTenantID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
