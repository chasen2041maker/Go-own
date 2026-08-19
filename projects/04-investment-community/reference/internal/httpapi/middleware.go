package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// HeaderRequestID 是每个响应都会返回的请求追踪标识。
const HeaderRequestID = "X-Request-ID"

type requestIDContextKey struct{}

var fallbackRequestIDCounter atomic.Uint64

// WithRequestID 只接受能完整写入审计表的安全上游标识，否则生成新值。
// 这里统一限制长度，避免治理事务最后写审计时才因字段溢出而整体回滚。
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := requestIDForHeader(request.Header.Get(HeaderRequestID))
		writer.Header().Set(HeaderRequestID, requestID)
		ctx := context.WithValue(request.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// RequestIDFromContext 返回 WithRequestID 放入请求上下文的标识。
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

// Recover 把 panic 转成安全的统一错误信封。
// 下游响应先写入内存缓冲，只有正常返回后才提交；这样即使 Handler 已写了一半再 panic，
// 客户端也不会收到“原状态码 + 半段业务 JSON + 半段错误 JSON”的损坏响应。
func Recover(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		buffer := newBufferedResponse()
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("HTTP handler panic recovered",
					"request_id", RequestIDFromContext(request.Context()),
					"method", request.Method,
					"path", request.URL.Path,
				)
				WriteError(writer, request, http.StatusInternalServerError, "internal_error", "服务暂时无法处理请求", nil)
				return
			}
			buffer.flushTo(writer)
		}()
		next.ServeHTTP(buffer, request)
	})
}

type bufferedResponse struct {
	header      http.Header
	status      int
	wroteHeader bool
	body        bytes.Buffer
}

func newBufferedResponse() *bufferedResponse {
	return &bufferedResponse{header: make(http.Header), status: http.StatusOK}
}

func (response *bufferedResponse) Header() http.Header {
	return response.header
}

func (response *bufferedResponse) WriteHeader(status int) {
	if response.wroteHeader {
		return
	}
	response.status = status
	response.wroteHeader = true
}

func (response *bufferedResponse) Write(contents []byte) (int, error) {
	if !response.wroteHeader {
		response.WriteHeader(http.StatusOK)
	}
	return response.body.Write(contents)
}

func (response *bufferedResponse) flushTo(writer http.ResponseWriter) {
	for name, values := range response.header {
		writer.Header()[name] = append([]string(nil), values...)
	}
	writer.WriteHeader(response.status)
	_, _ = response.body.WriteTo(writer)
}

func requestIDForHeader(upstream string) string {
	if upstream = strings.TrimSpace(upstream); validRequestID(upstream) {
		return upstream
	}

	var entropy [12]byte
	if _, err := rand.Read(entropy[:]); err == nil {
		return "req_" + hex.EncodeToString(entropy[:])
	}
	return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), fallbackRequestIDCounter.Add(1))
}

func validRequestID(requestID string) bool {
	if len(requestID) == 0 || len(requestID) > 64 {
		return false
	}
	for _, character := range requestID {
		if character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			strings.ContainsRune("._:-", character) {
			continue
		}
		return false
	}
	return true
}
