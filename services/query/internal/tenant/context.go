package tenant

import "context"

type contextKey struct{}

func FromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(contextKey{}).(string)
	return id, ok && id != ""
}

func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, contextKey{}, tenantID)
}
