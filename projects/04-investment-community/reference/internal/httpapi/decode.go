package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strings"
	"unicode/utf8"
)

const maxJSONBodyBytes int64 = 1 << 20

type decodeFailure struct {
	status  int
	code    string
	message string
}

// decodeJSON 在所有有请求体的端点统一执行协议边界校验，避免某个 Handler 漏掉
// 未知字段、尾随 JSON 或大小限制后形成绕过路径。
func decodeJSON(_ http.ResponseWriter, request *http.Request, destination any) *decodeFailure {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &decodeFailure{
			status:  http.StatusUnsupportedMediaType,
			code:    "unsupported_media_type",
			message: "Content-Type 必须是 application/json",
		}
	}

	// 先把最多 limit+1 个原始字节读完，再解析 JSON。若解析器先遇到语法错误就提前返回，
	// MaxBytesReader 无法证明整个请求体是否超限；预读让“任意 >1MiB 都是 413”成为确定契约。
	contents, err := io.ReadAll(io.LimitReader(request.Body, maxJSONBodyBytes+1))
	if err != nil {
		return invalidJSONFailure()
	}
	if int64(len(contents)) > maxJSONBodyBytes {
		return &decodeFailure{
			status:  http.StatusRequestEntityTooLarge,
			code:    "payload_too_large",
			message: "请求体过大",
		}
	}
	if !utf8.Valid(contents) {
		return invalidJSONFailure()
	}
	if failure := rejectInexactObjectFields(contents, destination); failure != nil {
		return failure
	}

	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return classifyDecodeError(err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return classifyDecodeError(err)
		}
		return invalidJSONFailure()
	}
	return nil
}

func classifyDecodeError(err error) *decodeFailure {
	return invalidJSONFailure()
}

func rejectInexactObjectFields(contents []byte, destination any) *decodeFailure {
	destinationType := reflect.TypeOf(destination)
	if destinationType == nil || destinationType.Kind() != reflect.Pointer || destinationType.Elem().Kind() != reflect.Struct {
		return &decodeFailure{status: http.StatusInternalServerError, code: "internal_error", message: "服务暂时无法处理请求"}
	}

	if bytes.Equal(bytes.TrimSpace(contents), []byte("null")) {
		return &decodeFailure{
			status:  http.StatusBadRequest,
			code:    "invalid_request",
			message: "请求体必须是非 null JSON 对象",
		}
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(contents, &object); err != nil {
		return nil // 具体 JSON 语法/类型错误由主 decoder 生成 invalid_json。
	}
	if object == nil {
		return &decodeFailure{status: http.StatusBadRequest, code: "invalid_request", message: "请求体必须是 JSON 对象"}
	}
	allowed := make(map[string]struct{}, destinationType.Elem().NumField())
	for index := 0; index < destinationType.Elem().NumField(); index++ {
		field := destinationType.Elem().Field(index)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			allowed[name] = struct{}{}
		}
	}
	for name := range object {
		if _, ok := allowed[name]; !ok {
			return &decodeFailure{
				status:  http.StatusBadRequest,
				code:    "invalid_request",
				message: "请求包含未知字段或字段名大小写不正确",
			}
		}
	}
	return nil
}

func invalidJSONFailure() *decodeFailure {
	return &decodeFailure{
		status:  http.StatusBadRequest,
		code:    "invalid_json",
		message: "请求 JSON 无法解析",
	}
}
