package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

const (
	DefaultPageLimit = 20
	MaximumPageLimit = 100
)

type CommunityRepository interface {
	ListSecurities(context.Context, domain.SecurityListQuery) ([]domain.Security, error)
	ListCircles(context.Context, domain.CircleListQuery) ([]domain.Circle, error)
	SetCircleMembership(context.Context, int64, int64, bool) (domain.CircleMembership, error)
}

type SecurityListInput struct {
	Query    string
	Exchange string
	After    *domain.SecurityCursor
	Limit    int
}

type SecurityPage struct {
	Items []domain.Security
	Next  *domain.SecurityCursor
}

type CircleListInput struct {
	UserID int64
	After  *domain.CircleCursor
	Limit  int
}

type CirclePage struct {
	Items []domain.Circle
	Next  *domain.CircleCursor
}

type SetCircleMembershipInput struct {
	UserID   int64
	CircleID int64
	Joined   bool
}

type CommunityService struct {
	repository CommunityRepository
}

func NewCommunityService(repository CommunityRepository) (*CommunityService, error) {
	if repository == nil {
		return nil, errors.New("community service repository is required")
	}
	return &CommunityService{repository: repository}, nil
}

func (service *CommunityService) ListSecurities(ctx context.Context, input SecurityListInput) (SecurityPage, error) {
	query, err := normalizeSecurityListInput(input)
	if err != nil {
		return SecurityPage{}, err
	}
	items, err := service.repository.ListSecurities(ctx, query)
	if err != nil {
		return SecurityPage{}, fmt.Errorf("list securities: %w", err)
	}
	return securityPage(items, inputLimit(input.Limit)), nil
}

func (service *CommunityService) ListCircles(ctx context.Context, input CircleListInput) (CirclePage, error) {
	if input.UserID <= 0 {
		return CirclePage{}, &domain.ValidationError{Field: "user_id", Reason: "必须是正整数"}
	}
	limit, err := validatePageLimit(input.Limit)
	if err != nil {
		return CirclePage{}, err
	}
	if input.After != nil && (input.After.ID <= 0 || input.After.CreatedAt.IsZero()) {
		return CirclePage{}, &domain.ValidationError{Field: "cursor", Reason: "分页位置无效"}
	}
	items, err := service.repository.ListCircles(ctx, domain.CircleListQuery{
		UserID: input.UserID,
		After:  input.After,
		Limit:  limit + 1,
	})
	if err != nil {
		return CirclePage{}, fmt.Errorf("list circles: %w", err)
	}
	return circlePage(items, limit), nil
}

func (service *CommunityService) SetCircleMembership(ctx context.Context, input SetCircleMembershipInput) (domain.CircleMembership, error) {
	if input.UserID <= 0 {
		return domain.CircleMembership{}, &domain.ValidationError{Field: "user_id", Reason: "必须是正整数"}
	}
	if input.CircleID <= 0 {
		return domain.CircleMembership{}, &domain.ValidationError{Field: "circle_id", Reason: "必须是正整数"}
	}
	// user_id 只能由认证中间件传入；请求 DTO 没有这个字段，因此客户端不能替别人入圈或退圈。
	membership, err := service.repository.SetCircleMembership(ctx, input.CircleID, input.UserID, input.Joined)
	if err != nil {
		return domain.CircleMembership{}, fmt.Errorf("set circle membership: %w", err)
	}
	return membership, nil
}

func normalizeSecurityListInput(input SecurityListInput) (domain.SecurityListQuery, error) {
	query := strings.TrimSpace(input.Query)
	exchange := strings.TrimSpace(input.Exchange)
	if !utf8.ValidString(query) || utf8.RuneCountInString(query) > 50 {
		return domain.SecurityListQuery{}, &domain.ValidationError{Field: "q", Reason: "最多 50 个字符"}
	}
	if !utf8.ValidString(exchange) || utf8.RuneCountInString(exchange) > 16 {
		return domain.SecurityListQuery{}, &domain.ValidationError{Field: "exchange", Reason: "最多 16 个字符"}
	}
	limit, err := validatePageLimit(input.Limit)
	if err != nil {
		return domain.SecurityListQuery{}, err
	}
	if input.After != nil && (input.After.ID <= 0 || input.After.Code == "") {
		return domain.SecurityListQuery{}, &domain.ValidationError{Field: "cursor", Reason: "分页位置无效"}
	}
	return domain.SecurityListQuery{
		Query: query, Exchange: exchange, After: input.After, Limit: limit + 1,
	}, nil
}

func validatePageLimit(limit int) (int, error) {
	if limit == 0 {
		return DefaultPageLimit, nil
	}
	if limit < 1 || limit > MaximumPageLimit {
		return 0, &domain.ValidationError{Field: "limit", Reason: "必须是 1 到 100 的整数"}
	}
	return limit, nil
}

func inputLimit(limit int) int {
	if limit == 0 {
		return DefaultPageLimit
	}
	return limit
}

func securityPage(items []domain.Security, limit int) SecurityPage {
	page := SecurityPage{Items: items}
	if len(page.Items) <= limit {
		return page
	}
	page.Items = page.Items[:limit]
	last := page.Items[len(page.Items)-1]
	page.Next = &domain.SecurityCursor{Code: last.Code, ID: last.ID}
	return page
}

func circlePage(items []domain.Circle, limit int) CirclePage {
	page := CirclePage{Items: items}
	if len(page.Items) <= limit {
		return page
	}
	page.Items = page.Items[:limit]
	last := page.Items[len(page.Items)-1]
	page.Next = &domain.CircleCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	return page
}
