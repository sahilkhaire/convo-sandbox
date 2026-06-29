package admin

import (
	"context"
	"net/http"
	"strings"

	"github.com/zixflow/messaging-simulator/internal/auth"
)

type ctxKey int

const claimsKey ctxKey = 1

func WithClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*auth.Claims)
	return claims, ok
}

type Auth struct {
	tokens *auth.TokenService
}

func NewAuth(tokens *auth.TokenService) *Auth {
	return &Auth{tokens: tokens}
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			writeErr(w, http.StatusUnauthorized, errMsg("authentication required"))
			return
		}
		claims, err := a.tokens.Verify(token)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, errMsg("invalid or expired token"))
			return
		}
		next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
	})
}

func (a *Auth) RequirePerm(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeErr(w, http.StatusUnauthorized, errMsg("authentication required"))
				return
			}
			if !auth.HasPermission(claims.IsAdmin, claims.Permissions, perm) {
				writeErr(w, http.StatusForbidden, errMsg("permission denied"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) string {
	if q := r.URL.Query().Get("token"); q != "" {
		return q
	}
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	return ""
}

type errMsg string

func (e errMsg) Error() string { return string(e) }
