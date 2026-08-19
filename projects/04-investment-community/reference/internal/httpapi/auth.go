package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

// AuthApplication 是 HTTP 层使用的最小认证能力，测试无需真实数据库或 bcrypt。
type AuthApplication interface {
	Register(context.Context, usecase.RegisterInput) (usecase.AuthResult, error)
	Login(context.Context, usecase.LoginInput) (usecase.AuthResult, error)
	Authenticate(context.Context, string) (domain.User, error)
}

type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type privateUserResponse struct {
	ID          int64             `json:"id"`
	Email       string            `json:"email"`
	DisplayName string            `json:"display_name"`
	Role        domain.Role       `json:"role"`
	Status      domain.UserStatus `json:"status"`
}

type authResponse struct {
	AccessToken string              `json:"access_token"`
	TokenType   string              `json:"token_type"`
	ExpiresIn   int64               `json:"expires_in"`
	User        privateUserResponse `json:"user"`
}

type authHandler struct {
	application AuthApplication
}

func registerAuthRoutes(mux *http.ServeMux, application AuthApplication) {
	handler := authHandler{application: application}
	mux.Handle("/api/v1/auth/register", requireMethod(http.MethodPost, http.HandlerFunc(handler.register)))
	mux.Handle("/api/v1/auth/login", requireMethod(http.MethodPost, http.HandlerFunc(handler.login)))
	mux.Handle("/api/v1/me", requireMethod(http.MethodGet, authenticate(application, http.HandlerFunc(handler.me))))
}

func (handler authHandler) register(writer http.ResponseWriter, request *http.Request) {
	var input registerRequest
	if failure := decodeJSON(writer, request, &input); failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	if handler.application == nil {
		writeInternalError(writer, request)
		return
	}
	result, err := handler.application.Register(request.Context(), usecase.RegisterInput{
		Email: input.Email, DisplayName: input.DisplayName, Password: input.Password,
	})
	if err != nil {
		writeAuthApplicationError(writer, request, err)
		return
	}
	writeAuthResult(writer, request, http.StatusCreated, result)
}

func (handler authHandler) login(writer http.ResponseWriter, request *http.Request) {
	var input loginRequest
	if failure := decodeJSON(writer, request, &input); failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	if handler.application == nil {
		writeInternalError(writer, request)
		return
	}
	result, err := handler.application.Login(request.Context(), usecase.LoginInput{
		Email: input.Email, Password: input.Password,
	})
	if err != nil {
		writeAuthApplicationError(writer, request, err)
		return
	}
	writeAuthResult(writer, request, http.StatusOK, result)
}

func (authHandler) me(writer http.ResponseWriter, request *http.Request) {
	user, ok := CurrentUserFromContext(request.Context())
	if !ok {
		writeUnauthenticated(writer, request)
		return
	}
	WriteJSON(writer, http.StatusOK, privateUser(user))
}

func writeAuthResult(writer http.ResponseWriter, request *http.Request, status int, result usecase.AuthResult) {
	expiresIn := int64(result.ExpiresIn / time.Second)
	if result.AccessToken == "" || expiresIn < 1 || result.User.ID <= 0 {
		writeInternalError(writer, request)
		return
	}
	WriteJSON(writer, status, authResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		User:        privateUser(result.User),
	})
}

func privateUser(user domain.User) privateUserResponse {
	return privateUserResponse{
		ID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Role: user.Role, Status: user.Status,
	}
}

func writeAuthApplicationError(writer http.ResponseWriter, request *http.Request, err error) {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		WriteError(writer, request, http.StatusUnprocessableEntity, "validation_failed", "请求字段未通过校验", []FieldViolation{{
			Field: validation.Field, Reason: validation.Reason,
		}})
	case errors.Is(err, domain.ErrEmailTaken):
		WriteError(writer, request, http.StatusConflict, "email_taken", "该邮箱已被注册", nil)
	case errors.Is(err, domain.ErrInvalidCredentials):
		WriteError(writer, request, http.StatusUnauthorized, "invalid_credentials", "邮箱或密码不正确", nil)
	case errors.Is(err, domain.ErrUnauthenticated):
		writeUnauthenticated(writer, request)
	case errors.Is(err, domain.ErrForbidden):
		WriteError(writer, request, http.StatusForbidden, "forbidden", "没有执行此操作的权限", nil)
	default:
		writeInternalError(writer, request)
	}
}

func writeUnauthenticated(writer http.ResponseWriter, request *http.Request) {
	WriteError(writer, request, http.StatusUnauthorized, "unauthenticated", "需要有效的访问令牌", nil)
}

func writeInternalError(writer http.ResponseWriter, request *http.Request) {
	WriteError(writer, request, http.StatusInternalServerError, "internal_error", "服务暂时无法处理请求", nil)
}
