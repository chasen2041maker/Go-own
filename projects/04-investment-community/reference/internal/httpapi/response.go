// Package httpapi 定义所有 Handler 共享的公开 HTTP 协议。
package httpapi

import (
	"encoding/json"
	"net/http"
)

// FieldViolation 描述一个无效输入字段，但不暴露内部实现细节。
type FieldViolation struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// ErrorObject 是稳定、可由客户端判断的错误契约。
type ErrorObject struct {
	Code      string           `json:"code"`
	Message   string           `json:"message"`
	RequestID string           `json:"request_id"`
	Details   []FieldViolation `json:"details"`
}

// ErrorEnvelope 让全部错误响应使用相同顶层结构。
type ErrorEnvelope struct {
	Error ErrorObject `json:"error"`
}

// WriteJSON 使用 API 统一的 Content-Type 写入 JSON。
func WriteJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

// WriteError 让路由、中间件和业务错误复用同一信封。
func WriteError(writer http.ResponseWriter, request *http.Request, status int, code, message string, details []FieldViolation) {
	requestID := RequestIDFromContext(request.Context())
	if requestID == "" {
		requestID = requestIDForHeader(request.Header.Get(HeaderRequestID))
		writer.Header().Set(HeaderRequestID, requestID)
	}
	if details == nil {
		details = []FieldViolation{}
	}
	WriteJSON(writer, status, ErrorEnvelope{Error: ErrorObject{
		Code:      code,
		Message:   message,
		RequestID: requestID,
		Details:   details,
	}})
}
