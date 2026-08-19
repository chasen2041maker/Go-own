package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteErrorProducesStableEnvelope(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	response := httptest.NewRecorder()
	handler := WithRequestID(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		WriteError(writer, request, http.StatusUnprocessableEntity, "validation_failed", "请求字段未通过校验", []FieldViolation{
			{Field: "title", Reason: "不能为空"},
		})
	}))

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
	if got, want := response.Header().Get("Content-Type"), "application/json"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	var envelope ErrorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if got, want := envelope.Error.Code, "validation_failed"; got != want {
		t.Errorf("code = %q, want %q", got, want)
	}
	if got, want := envelope.Error.Message, "请求字段未通过校验"; got != want {
		t.Errorf("message = %q, want %q", got, want)
	}
	if len(envelope.Error.Details) != 1 || envelope.Error.Details[0].Field != "title" {
		t.Errorf("details = %#v, want title violation", envelope.Error.Details)
	}
	if got := response.Header().Get(HeaderRequestID); got == "" || got != envelope.Error.RequestID {
		t.Errorf("X-Request-ID = %q, body request_id = %q", got, envelope.Error.RequestID)
	}
}

func TestWriteErrorEncodesEmptyDetailsAsArray(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	response := httptest.NewRecorder()
	handler := WithRequestID(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		WriteError(writer, request, http.StatusInternalServerError, "internal_error", "服务暂时无法处理请求", nil)
	}))

	handler.ServeHTTP(response, request)

	var raw struct {
		Error struct {
			Details json.RawMessage `json:"details"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&raw); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if got, want := string(raw.Error.Details), "[]"; got != want {
		t.Errorf("details = %s, want %s", got, want)
	}
}
