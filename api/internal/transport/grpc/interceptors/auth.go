// Package interceptors holds the gRPC server interceptors.
//
// Before this existed the gRPC surface had no authentication whatsoever: any
// caller who could reach :3002 could create, update and delete events. The
// handler compensated by setting CreatedBy to uuid.New() behind a
// `TODO: Get from context`, attributing every event to a user that does not
// exist.
package interceptors

import (
	"context"

	"eventify/api/internal/shared/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ctxKey int

const claimsKey ctxKey = iota

// Claims returns the validated claims attached by Auth.
func Claims(ctx context.Context) (*auth.CustomClaims, bool) {
	c, ok := ctx.Value(claimsKey).(*auth.CustomClaims)
	return c, ok
}

// publicMethods may be called without a token.
var publicMethods = map[string]bool{}

// Auth validates the bearer token in the `authorization` metadata header and
// attaches the claims to the context.
func Auth(jwtProvider auth.IJWTProvider) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		const prefix = "Bearer "
		header := values[0]
		if len(header) <= len(prefix) {
			return nil, status.Error(codes.Unauthenticated, "bearer token required")
		}

		claims, err := jwtProvider.ValidateToken(header[len(prefix):])
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		return handler(context.WithValue(ctx, claimsKey, claims), req)
	}
}

// RequirePermission returns an error unless the claims grant one of permissions.
func RequirePermission(ctx context.Context, permissions ...string) error {
	claims, ok := Claims(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "authentication required")
	}
	for _, granted := range claims.Permissions {
		for _, required := range permissions {
			if granted == required {
				return nil
			}
		}
	}
	return status.Error(codes.PermissionDenied, "insufficient permissions")
}
