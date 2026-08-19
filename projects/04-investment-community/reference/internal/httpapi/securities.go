package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/usecase"
)

type CommunityApplication interface {
	ListSecurities(context.Context, usecase.SecurityListInput) (usecase.SecurityPage, error)
	ListCircles(context.Context, usecase.CircleListInput) (usecase.CirclePage, error)
	SetCircleMembership(context.Context, usecase.SetCircleMembershipInput) (domain.CircleMembership, error)
}

type securityResponse struct {
	ID       int64  `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Exchange string `json:"exchange"`
}

type cursorPageResponse struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

type securityListResponse struct {
	Items []securityResponse `json:"items"`
	Page  cursorPageResponse `json:"page"`
}

type securitiesHandler struct {
	application CommunityApplication
	cursors     *CursorCodec
}

func registerCommunityRoutes(mux *http.ServeMux, auth AuthApplication, application CommunityApplication, cursors *CursorCodec) {
	securities := securitiesHandler{application: application, cursors: cursors}
	circles := circlesHandler{application: application, cursors: cursors}
	mux.Handle("/api/v1/securities", requireMethod(http.MethodGet,
		authenticate(auth, http.HandlerFunc(securities.list))))
	mux.Handle("/api/v1/circles", requireMethod(http.MethodGet,
		authenticate(auth, http.HandlerFunc(circles.list))))
	mux.Handle("/api/v1/circles/{circleId}/membership", requireMethod(http.MethodPut,
		authenticate(auth, http.HandlerFunc(circles.setMembership))))
}

func (handler securitiesHandler) list(writer http.ResponseWriter, request *http.Request) {
	query, failure := parseSecurityListQuery(request)
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	if handler.application == nil || handler.cursors == nil {
		writeInternalError(writer, request)
		return
	}

	input := usecase.SecurityListInput{Query: query.query, Exchange: query.exchange, Limit: query.limit}
	if query.cursor != "" {
		position, err := handler.cursors.DecodeSecurity(query.cursor, securityCursorBinding{
			Query: query.query, Exchange: query.exchange, Limit: query.limit,
		})
		if err != nil {
			WriteError(writer, request, http.StatusBadRequest, "invalid_cursor", "分页游标无效", nil)
			return
		}
		input.After = &position
	}

	page, err := handler.application.ListSecurities(request.Context(), input)
	if err != nil {
		writeCommunityApplicationError(writer, request, err)
		return
	}
	response := securityListResponse{
		Items: make([]securityResponse, 0, len(page.Items)),
		Page:  cursorPageResponse{HasMore: page.Next != nil},
	}
	for _, security := range page.Items {
		response.Items = append(response.Items, securityResponse{
			ID: security.ID, Code: security.Code, Name: security.Name, Exchange: security.Exchange,
		})
	}
	if page.Next != nil {
		token, err := handler.cursors.EncodeSecurity(securityCursorBinding{
			Query: query.query, Exchange: query.exchange, Limit: query.limit,
		}, *page.Next)
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		response.Page.NextCursor = &token
	}
	WriteJSON(writer, http.StatusOK, response)
}

type parsedSecurityListQuery struct {
	query    string
	exchange string
	limit    int
	cursor   string
}

func parseSecurityListQuery(request *http.Request) (parsedSecurityListQuery, *decodeFailure) {
	values, failure := parseKnownQuery(request, map[string]struct{}{
		"q": {}, "exchange": {}, "limit": {}, "cursor": {},
	})
	if failure != nil {
		return parsedSecurityListQuery{}, failure
	}
	query, failure := singleQueryValue(values, "q")
	if failure != nil {
		return parsedSecurityListQuery{}, failure
	}
	exchange, failure := singleQueryValue(values, "exchange")
	if failure != nil {
		return parsedSecurityListQuery{}, failure
	}
	cursor, failure := singleQueryValue(values, "cursor")
	if failure != nil {
		return parsedSecurityListQuery{}, failure
	}
	query = strings.TrimSpace(query)
	exchange = strings.TrimSpace(exchange)
	if !utf8.ValidString(query) || utf8.RuneCountInString(query) > 50 {
		return parsedSecurityListQuery{}, invalidRequestFailure("q 最多 50 个字符")
	}
	if !utf8.ValidString(exchange) || utf8.RuneCountInString(exchange) > 16 {
		return parsedSecurityListQuery{}, invalidRequestFailure("exchange 最多 16 个字符")
	}
	limit, failure := parsePageLimit(values)
	if failure != nil {
		return parsedSecurityListQuery{}, failure
	}
	if _, present := values["cursor"]; present && cursor == "" {
		return parsedSecurityListQuery{}, &decodeFailure{
			status: http.StatusBadRequest, code: "invalid_cursor", message: "分页游标无效",
		}
	}
	return parsedSecurityListQuery{query: query, exchange: exchange, limit: limit, cursor: cursor}, nil
}

func parseKnownQuery(request *http.Request, allowed map[string]struct{}) (url.Values, *decodeFailure) {
	values, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return nil, invalidRequestFailure("查询参数无法解析")
	}
	for key := range values {
		if _, ok := allowed[key]; !ok {
			return nil, invalidRequestFailure("请求包含未知查询参数")
		}
	}
	return values, nil
}

func singleQueryValue(values url.Values, key string) (string, *decodeFailure) {
	entries, present := values[key]
	if !present {
		return "", nil
	}
	if len(entries) != 1 {
		return "", invalidRequestFailure("查询参数不能重复")
	}
	return entries[0], nil
}

func parsePageLimit(values url.Values) (int, *decodeFailure) {
	raw, present := values["limit"]
	if !present {
		return usecase.DefaultPageLimit, nil
	}
	if len(raw) != 1 || raw[0] == "" {
		return 0, invalidRequestFailure("limit 必须是 1 到 100 的整数")
	}
	limit, err := strconv.Atoi(raw[0])
	if err != nil || limit < 1 || limit > usecase.MaximumPageLimit {
		return 0, invalidRequestFailure("limit 必须是 1 到 100 的整数")
	}
	return limit, nil
}

func invalidRequestFailure(message string) *decodeFailure {
	return &decodeFailure{status: http.StatusBadRequest, code: "invalid_request", message: message}
}

func writeCommunityApplicationError(writer http.ResponseWriter, request *http.Request, err error) {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		WriteError(writer, request, http.StatusBadRequest, "invalid_request", "请求参数无效", []FieldViolation{{
			Field: validation.Field, Reason: validation.Reason,
		}})
	case errors.Is(err, domain.ErrCircleNotFound):
		WriteError(writer, request, http.StatusNotFound, "not_found", "圈子不存在", nil)
	default:
		writeInternalError(writer, request)
	}
}
