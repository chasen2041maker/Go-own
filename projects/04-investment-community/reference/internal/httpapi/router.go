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
func NewRouter(checker ReadinessChecker, readinessTimeout time.Duration, authApplications ...AuthApplication) http.Handler {
	var authApplication AuthApplication
	if len(authApplications) > 0 {
		authApplication = authApplications[0]
	}
	return newRouter(checker, readinessTimeout, authApplication, nil, nil, nil, nil, nil, nil)
}

// NewRouterWithCommunity 保留原有 NewRouter 的窄测试入口，同时为业务进程显式装配本阶段能力。
func NewRouterWithCommunity(
	checker ReadinessChecker,
	readinessTimeout time.Duration,
	authApplication AuthApplication,
	communityApplication CommunityApplication,
	cursors *CursorCodec,
) http.Handler {
	return newRouter(checker, readinessTimeout, authApplication, communityApplication, nil, nil, nil, nil, cursors)
}

// NewRouterWithCommunityAndPosts 把阶段能力显式注入，后续模块无需让 Handler 直接依赖具体仓储。
func NewRouterWithCommunityAndPosts(checker ReadinessChecker, readinessTimeout time.Duration, auth AuthApplication,
	community CommunityApplication, posts PostsApplication, cursors *CursorCodec) http.Handler {
	return newRouter(checker, readinessTimeout, auth, community, posts, nil, nil, nil, cursors)
}

// NewRouterWithInteractions 组装当前已完成的全部业务切片；每个 Handler 仍只依赖自己的用例接口。
func NewRouterWithInteractions(checker ReadinessChecker, readinessTimeout time.Duration, auth AuthApplication,
	community CommunityApplication, posts PostsApplication, interactions InteractionsApplication, cursors *CursorCodec) http.Handler {
	return newRouter(checker, readinessTimeout, auth, community, posts, interactions, nil, nil, cursors)
}

func NewRouterWithReports(checker ReadinessChecker, readinessTimeout time.Duration, auth AuthApplication,
	community CommunityApplication, posts PostsApplication, interactions InteractionsApplication,
	reports ReportsApplication, cursors *CursorCodec) http.Handler {
	return newRouter(checker, readinessTimeout, auth, community, posts, interactions, reports, nil, cursors)
}

// NewRouterWithGovernance 是生产进程的完整装配入口；窄构造器继续服务较早阶段的隔离测试。
func NewRouterWithGovernance(checker ReadinessChecker, readinessTimeout time.Duration, auth AuthApplication,
	community CommunityApplication, posts PostsApplication, interactions InteractionsApplication,
	reports ReportsApplication, governance GovernanceApplication, cursors *CursorCodec) http.Handler {
	return newRouter(checker, readinessTimeout, auth, community, posts, interactions, reports, governance, cursors)
}

func newRouter(
	checker ReadinessChecker,
	readinessTimeout time.Duration,
	authApplication AuthApplication,
	communityApplication CommunityApplication,
	postsApplication PostsApplication,
	interactionsApplication InteractionsApplication,
	reportsApplication ReportsApplication,
	governanceApplication GovernanceApplication,
	cursors *CursorCodec,
) http.Handler {
	if readinessTimeout <= 0 {
		readinessTimeout = defaultReadinessTimeout
	}

	mux := http.NewServeMux()
	mux.Handle("/healthz", requireMethod(http.MethodGet, http.HandlerFunc(healthz)))
	mux.Handle("/readyz", requireMethod(http.MethodGet, http.HandlerFunc(readyz(checker, readinessTimeout))))
	registerAuthRoutes(mux, authApplication)
	if communityApplication != nil {
		registerCommunityRoutes(mux, authApplication, communityApplication, cursors)
	}
	if postsApplication != nil {
		registerPostRoutes(mux, authApplication, postsApplication, cursors)
	}
	if interactionsApplication != nil {
		registerInteractionRoutes(mux, authApplication, interactionsApplication, cursors)
	}
	if reportsApplication != nil {
		registerReportRoutes(mux, authApplication, reportsApplication, cursors)
	}
	if governanceApplication != nil {
		registerGovernanceRoutes(mux, authApplication, governanceApplication, cursors)
	}
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		WriteError(writer, request, http.StatusNotFound, "not_found", "资源不存在", nil)
	})

	logger := slog.Default()
	// Recover 必须位于 AccessLog 内侧：panic 先被转成最终 500，访问日志才能记录真实状态。
	return SecurityHeaders(WithRequestID(AccessLog(logger, Recover(logger, mux))))
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
