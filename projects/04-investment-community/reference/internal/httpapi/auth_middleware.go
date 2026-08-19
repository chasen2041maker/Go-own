package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

type currentUserContextKey struct{}

func authenticate(application AuthApplication, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		rawToken, ok := bearerToken(request.Header.Values("Authorization"))
		if !ok || application == nil {
			writeUnauthenticated(writer, request)
			return
		}
		user, err := application.Authenticate(request.Context(), rawToken)
		if errors.Is(err, domain.ErrUnauthenticated) {
			writeUnauthenticated(writer, request)
			return
		}
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		if user.Status != domain.UserStatusActive {
			writeUnauthenticated(writer, request)
			return
		}
		ctx := context.WithValue(request.Context(), currentUserContextKey{}, user)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func bearerToken(values []string) (string, bool) {
	if len(values) != 1 {
		return "", false
	}
	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

// CurrentUserFromContext 返回认证中间件刚从数据库加载的用户快照。
func CurrentUserFromContext(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(currentUserContextKey{}).(domain.User)
	return user, ok
}

// RequireRole 只读取认证中间件放入的 DB 用户，绝不读取客户端或 JWT 中的 role。
func RequireRole(role domain.Role, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, ok := CurrentUserFromContext(request.Context())
		if !ok || user.Status != domain.UserStatusActive {
			writeUnauthenticated(writer, request)
			return
		}
		if err := usecase.RequireRole(user, role); err != nil {
			WriteError(writer, request, http.StatusForbidden, "forbidden", "没有执行此操作的权限", nil)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return RequireRole(domain.RoleAdmin, next)
}
