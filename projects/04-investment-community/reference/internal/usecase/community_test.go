package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

func TestListSecuritiesNormalizesFiltersAndBuildsStablePage(t *testing.T) {
	repository := &fakeCommunityRepository{listSecurities: func(_ context.Context, query domain.SecurityListQuery) ([]domain.Security, error) {
		if query.Query != "NO" || query.Exchange != "XSEA" {
			t.Fatalf("ListSecurities() filters = %q/%q, want normalized filters", query.Query, query.Exchange)
		}
		if query.Limit != 3 {
			t.Fatalf("ListSecurities() limit = %d, want requested limit + 1", query.Limit)
		}
		if query.After == nil || query.After.Code != "AURR" || query.After.ID != 4 {
			t.Fatalf("ListSecurities() after = %#v", query.After)
		}
		return []domain.Security{
			{ID: 10, Code: "NOVA", Name: "新星科技", Exchange: "XSEA"},
			{ID: 11, Code: "NOVB", Name: "新湾材料", Exchange: "XSEA"},
			{ID: 12, Code: "NOVC", Name: "新潮工业", Exchange: "XSEA"},
		}, nil
	}}
	service := mustCommunityService(t, repository)

	page, err := service.ListSecurities(context.Background(), SecurityListInput{
		Query: " NO ", Exchange: " XSEA ", Limit: 2,
		After: &domain.SecurityCursor{Code: "AURR", ID: 4},
	})
	if err != nil {
		t.Fatalf("ListSecurities() error = %v", err)
	}
	if len(page.Items) != 2 || page.Items[0].Code != "NOVA" || page.Items[1].Code != "NOVB" {
		t.Fatalf("ListSecurities() items = %#v", page.Items)
	}
	if page.Next == nil || page.Next.Code != "NOVB" || page.Next.ID != 11 {
		t.Fatalf("ListSecurities() next = %#v", page.Next)
	}
}

func TestListSecuritiesRejectsInvalidBoundaryValues(t *testing.T) {
	repository := &fakeCommunityRepository{listSecurities: func(context.Context, domain.SecurityListQuery) ([]domain.Security, error) {
		t.Fatal("invalid input must not reach the repository")
		return nil, nil
	}}
	service := mustCommunityService(t, repository)

	tests := []struct {
		name  string
		input SecurityListInput
		field string
	}{
		{name: "query too long", input: SecurityListInput{Query: strings.Repeat("界", 51), Limit: 20}, field: "q"},
		{name: "exchange too long", input: SecurityListInput{Exchange: strings.Repeat("X", 17), Limit: 20}, field: "exchange"},
		{name: "limit too large", input: SecurityListInput{Limit: 101}, field: "limit"},
		{name: "invalid cursor id", input: SecurityListInput{Limit: 20, After: &domain.SecurityCursor{Code: "NOVA", ID: 0}}, field: "cursor"},
		{name: "invalid cursor code", input: SecurityListInput{Limit: 20, After: &domain.SecurityCursor{ID: 1}}, field: "cursor"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.ListSecurities(context.Background(), test.input)
			var validation *domain.ValidationError
			if !errors.As(err, &validation) || validation.Field != test.field {
				t.Fatalf("ListSecurities() error = %v, want %s ValidationError", err, test.field)
			}
		})
	}
}

func TestListCirclesUsesCurrentUserAndBuildsDescendingCursorPage(t *testing.T) {
	created := time.Date(2026, 8, 19, 7, 0, 0, 123000000, time.UTC)
	repository := &fakeCommunityRepository{listCircles: func(_ context.Context, query domain.CircleListQuery) ([]domain.Circle, error) {
		if query.UserID != 42 || query.Limit != 3 {
			t.Fatalf("ListCircles() query = %#v", query)
		}
		if query.After == nil || !query.After.CreatedAt.Equal(created.Add(time.Hour)) || query.After.ID != 9 {
			t.Fatalf("ListCircles() after = %#v", query.After)
		}
		return []domain.Circle{
			{ID: 8, Slug: "risk-lab", Name: "风险实验室", CreatedAt: created, IsMember: true},
			{ID: 7, Slug: "long-horizon", Name: "长期观察", CreatedAt: created, IsMember: false},
			{ID: 6, Slug: "market-notes", Name: "市场笔记", CreatedAt: created.Add(-time.Hour)},
		}, nil
	}}
	service := mustCommunityService(t, repository)

	page, err := service.ListCircles(context.Background(), CircleListInput{
		UserID: 42, Limit: 2, After: &domain.CircleCursor{CreatedAt: created.Add(time.Hour), ID: 9},
	})
	if err != nil {
		t.Fatalf("ListCircles() error = %v", err)
	}
	if len(page.Items) != 2 || !page.Items[0].IsMember || page.Items[1].IsMember {
		t.Fatalf("ListCircles() items = %#v", page.Items)
	}
	if page.Next == nil || !page.Next.CreatedAt.Equal(created) || page.Next.ID != 7 {
		t.Fatalf("ListCircles() next = %#v", page.Next)
	}
}

func TestSetCircleMembershipUsesAuthenticatedUserForJoinAndLeave(t *testing.T) {
	joinedAt := time.Date(2026, 8, 19, 8, 0, 0, 0, time.UTC)
	var calls []bool
	repository := &fakeCommunityRepository{setMembership: func(_ context.Context, circleID, userID int64, joined bool) (domain.CircleMembership, error) {
		if circleID != 7 || userID != 42 {
			t.Fatalf("SetCircleMembership() ids = %d/%d, want 7/42", circleID, userID)
		}
		calls = append(calls, joined)
		membership := domain.CircleMembership{CircleID: circleID, UserID: userID, Joined: joined}
		if joined {
			membership.JoinedAt = &joinedAt
		}
		return membership, nil
	}}
	service := mustCommunityService(t, repository)

	joined, err := service.SetCircleMembership(context.Background(), SetCircleMembershipInput{
		UserID: 42, CircleID: 7, Joined: true,
	})
	if err != nil || !joined.Joined || joined.JoinedAt == nil {
		t.Fatalf("join result = %#v, error = %v", joined, err)
	}
	left, err := service.SetCircleMembership(context.Background(), SetCircleMembershipInput{
		UserID: 42, CircleID: 7, Joined: false,
	})
	if err != nil || left.Joined || left.JoinedAt != nil {
		t.Fatalf("leave result = %#v, error = %v", left, err)
	}
	if len(calls) != 2 || !calls[0] || calls[1] {
		t.Fatalf("SetCircleMembership() joined calls = %#v", calls)
	}
}

func TestSetCircleMembershipRejectsInvalidIDsAndPreservesNotFound(t *testing.T) {
	repository := &fakeCommunityRepository{setMembership: func(context.Context, int64, int64, bool) (domain.CircleMembership, error) {
		return domain.CircleMembership{}, domain.ErrCircleNotFound
	}}
	service := mustCommunityService(t, repository)

	for _, input := range []SetCircleMembershipInput{
		{UserID: 0, CircleID: 1, Joined: true},
		{UserID: 1, CircleID: 0, Joined: true},
	} {
		if _, err := service.SetCircleMembership(context.Background(), input); err == nil {
			t.Fatalf("SetCircleMembership(%#v) error = nil", input)
		}
	}
	_, err := service.SetCircleMembership(context.Background(), SetCircleMembershipInput{UserID: 1, CircleID: 999, Joined: true})
	if !errors.Is(err, domain.ErrCircleNotFound) {
		t.Fatalf("SetCircleMembership() error = %v, want ErrCircleNotFound", err)
	}
}

type fakeCommunityRepository struct {
	listSecurities func(context.Context, domain.SecurityListQuery) ([]domain.Security, error)
	listCircles    func(context.Context, domain.CircleListQuery) ([]domain.Circle, error)
	setMembership  func(context.Context, int64, int64, bool) (domain.CircleMembership, error)
}

func (repository *fakeCommunityRepository) ListSecurities(ctx context.Context, query domain.SecurityListQuery) ([]domain.Security, error) {
	if repository.listSecurities == nil {
		return nil, errors.New("unexpected ListSecurities call")
	}
	return repository.listSecurities(ctx, query)
}

func (repository *fakeCommunityRepository) ListCircles(ctx context.Context, query domain.CircleListQuery) ([]domain.Circle, error) {
	if repository.listCircles == nil {
		return nil, errors.New("unexpected ListCircles call")
	}
	return repository.listCircles(ctx, query)
}

func (repository *fakeCommunityRepository) SetCircleMembership(ctx context.Context, circleID, userID int64, joined bool) (domain.CircleMembership, error) {
	if repository.setMembership == nil {
		return domain.CircleMembership{}, errors.New("unexpected SetCircleMembership call")
	}
	return repository.setMembership(ctx, circleID, userID, joined)
}

func mustCommunityService(t *testing.T, repository CommunityRepository) *CommunityService {
	t.Helper()
	service, err := NewCommunityService(repository)
	if err != nil {
		t.Fatalf("NewCommunityService() error = %v", err)
	}
	return service
}
