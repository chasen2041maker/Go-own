package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"go-own/projects/04-investment-community/reference/internal/usecase"
)

type circleResponse struct {
	ID          int64     `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MemberCount int64     `json:"member_count"`
	IsMember    bool      `json:"is_member"`
	CreatedAt   time.Time `json:"created_at"`
}

type circleListResponse struct {
	Items []circleResponse   `json:"items"`
	Page  cursorPageResponse `json:"page"`
}

type setCircleMembershipRequest struct {
	Joined *bool `json:"joined"`
}

type circleMembershipResponse struct {
	CircleID int64      `json:"circle_id"`
	UserID   int64      `json:"user_id"`
	Joined   bool       `json:"joined"`
	JoinedAt *time.Time `json:"joined_at"`
}

type circlesHandler struct {
	application CommunityApplication
	cursors     *CursorCodec
}

func (handler circlesHandler) list(writer http.ResponseWriter, request *http.Request) {
	values, failure := parseKnownQuery(request, map[string]struct{}{"limit": {}, "cursor": {}})
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	cursor, failure := singleQueryValue(values, "cursor")
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	limit, failure := parsePageLimit(values)
	if failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	if _, present := values["cursor"]; present && cursor == "" {
		WriteError(writer, request, http.StatusBadRequest, "invalid_cursor", "分页游标无效", nil)
		return
	}
	user, ok := CurrentUserFromContext(request.Context())
	if !ok {
		writeUnauthenticated(writer, request)
		return
	}
	if handler.application == nil || handler.cursors == nil {
		writeInternalError(writer, request)
		return
	}

	input := usecase.CircleListInput{UserID: user.ID, Limit: limit}
	if cursor != "" {
		position, err := handler.cursors.DecodeCircle(cursor, circleCursorBinding{Limit: limit})
		if err != nil {
			WriteError(writer, request, http.StatusBadRequest, "invalid_cursor", "分页游标无效", nil)
			return
		}
		input.After = &position
	}
	page, err := handler.application.ListCircles(request.Context(), input)
	if err != nil {
		writeCommunityApplicationError(writer, request, err)
		return
	}
	response := circleListResponse{
		Items: make([]circleResponse, 0, len(page.Items)),
		Page:  cursorPageResponse{HasMore: page.Next != nil},
	}
	for _, circle := range page.Items {
		response.Items = append(response.Items, circleResponse{
			ID: circle.ID, Slug: circle.Slug, Name: circle.Name, Description: circle.Description,
			MemberCount: circle.MemberCount, IsMember: circle.IsMember, CreatedAt: circle.CreatedAt.UTC(),
		})
	}
	if page.Next != nil {
		token, err := handler.cursors.EncodeCircle(circleCursorBinding{Limit: limit}, *page.Next)
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		response.Page.NextCursor = &token
	}
	WriteJSON(writer, http.StatusOK, response)
}

func (handler circlesHandler) setMembership(writer http.ResponseWriter, request *http.Request) {
	circleID, err := strconv.ParseInt(request.PathValue("circleId"), 10, 64)
	if err != nil || circleID <= 0 {
		WriteError(writer, request, http.StatusBadRequest, "invalid_request", "circleId 必须是正 int64", nil)
		return
	}
	var input setCircleMembershipRequest
	if failure := decodeJSON(writer, request, &input); failure != nil {
		WriteError(writer, request, failure.status, failure.code, failure.message, nil)
		return
	}
	if input.Joined == nil {
		WriteError(writer, request, http.StatusBadRequest, "invalid_request", "joined 必须是布尔值", nil)
		return
	}
	user, ok := CurrentUserFromContext(request.Context())
	if !ok {
		writeUnauthenticated(writer, request)
		return
	}
	if handler.application == nil {
		writeInternalError(writer, request)
		return
	}
	membership, err := handler.application.SetCircleMembership(request.Context(), usecase.SetCircleMembershipInput{
		UserID: user.ID, CircleID: circleID, Joined: *input.Joined,
	})
	if err != nil {
		writeCommunityApplicationError(writer, request, err)
		return
	}
	if membership.CircleID <= 0 || membership.UserID <= 0 || membership.Joined != *input.Joined ||
		(membership.Joined && membership.JoinedAt == nil) {
		writeInternalError(writer, request)
		return
	}
	joinedAt := membership.JoinedAt
	if !membership.Joined {
		// 退出与重复退出都没有持久化成员行，因此协议必须稳定返回 null。
		joinedAt = nil
	} else {
		utc := membership.JoinedAt.UTC()
		joinedAt = &utc
	}
	WriteJSON(writer, http.StatusOK, circleMembershipResponse{
		CircleID: membership.CircleID, UserID: membership.UserID,
		Joined: membership.Joined, JoinedAt: joinedAt,
	})
}
