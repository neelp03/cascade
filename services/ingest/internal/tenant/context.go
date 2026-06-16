// Package tenant provides tenancy context injection and enforcement.
// In M1: reads X-Tenant-ID header. A5 (Auth) replaces this with JWT claims.
package tenant

import "context"

type contextKey struct{}

// FromContext retrieves the tenant ID from a request context.
// Returns ("", false) if not set — callers must reject the request.
func FromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(contextKey{}).(string)
	return id, ok && id != ""
}

// WithTenantID returns a copy of ctx with tenantID stored.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, contextKey{}, tenantID)
}
