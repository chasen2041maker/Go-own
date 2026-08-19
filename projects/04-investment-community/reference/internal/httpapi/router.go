package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

const defaultReadinessTimeout = 2 * time.Second

// ReadinessChecker 与 *sql.DB 的探活能力一致，但不让 HTTP 测试依赖真实数据库。
type ReadinessChecker interface {
	PingContext(context.Context) error
}

// NewRouter 组装运维端点和公共安全中间件。
func NewRouter(checker ReadinessChecker, readinessTimeout time.Duration) http.Handler {
	if readinessTimeout <= 0 {
		readinessTimeout = defaultReadinessTimeout
	}

	mux := http.NewServeMux()
	mux.Handle("/healthz", requireMethod(http.MethodGet, http.HandlerFunc(healthz)))
	mux.Handle("/readyz", requireMethod(http.MethodGet, http.HandlerFunc(readyz(checker, readinessTimeout))))
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		WriteError(writer, request, http.StatusNotFound, "not_found", "资源不存在", nil)
	})

	return WithRequestID(Recover(slog.Default(), mux))
}

func healthz(writer http.ResponseWriter, _ *http.Request) {
	// Liveness 故意不访问外部依赖，数据库故障不应触发进程反复重启。
	WriteJSON(writer, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: "ok"})
}

func readyz(checker ReadinessChecker, timeout time.Duration) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if checker == nil {
			WriteError(writer, request, http.StatusServiceUnavailable, "service_unavailable", "服务尚未就绪", nil)
			return
		}

		// Readiness 使用独立短超时，避免卡住的依赖耗尽更长的 HTTP 写超时。
		ctx, cancel := context.WithTimeout(request.Context(), timeout)
		defer cancel()
		if err := checker.PingContext(ctx); err != nil {
			WriteError(writer, request, http.StatusServiceUnavailable, "service_unavailable", "服务尚未就绪", nil)
			return
		}
		WriteJSON(writer, http.StatusOK, struct {
			Status string `json:"status"`
		}{Status: "ok"})
	}
}

func requireMethod(method string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != method {
			writer.Header().Set("Allow", method)
			WriteError(writer, request, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不受支持", nil)
			return
		}
		next.ServeHTTP(writer, request)
	})
}
