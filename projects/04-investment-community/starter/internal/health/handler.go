// Package health 提供不依赖数据库的进程存活检查。
package health

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
)

// NewHandler 返回 /healthz 的处理器。
//
// 教学顺序：先确认 HTTP 方法，再写响应头，最后提交状态码和 JSON。
// 健康检查故意不访问数据库；数据库故障应影响 readiness，而不是 liveness。
func NewHandler() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("X-Request-ID", rand.Text())

		if request.Method != http.MethodGet {
			response.Header().Set("Allow", http.MethodGet)
			http.Error(response, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(struct {
			Status string `json:"status"`
		}{Status: "ok"})
	})
}
