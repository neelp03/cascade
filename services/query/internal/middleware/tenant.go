package middleware

import (
	"net/http"

	"github.com/cascade-analytics/cascade/services/query/internal/tenant"
)

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
