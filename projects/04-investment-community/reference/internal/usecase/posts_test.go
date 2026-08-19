package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

func TestCreatePostNormalizesCanonicalRequestAndPreservesRepositoryErrors(t *testing.T) {
	var first domain.CreatePostParams
	repository := &fakePostRepository{create: func(_ context.Context, input domain.CreatePostParams) (domain.Post, error) {
		first = input
		return domain.Post{ID: 9}, nil
	}}
	service := mustPostService(t, repository)

	created, err := service.CreatePost(context.Background(), CreatePostInput{
		UserID: 42, IdempotencyKey: "post-key-1", CircleID: 7,
		Title: "  长期观察  ", Body: "  正文  ", SecurityIDs: []int64{3, 1, 2},
	})
	if err != nil || created.ID != 9 {
		t.Fatalf("CreatePost() = %#v, %v", created, err)
	}
	if first.AuthorID != 42 || first.Title != "长期观察" || first.Body != "正文" ||
		len(first.SecurityIDs) != 3 || first.SecurityIDs[0] != 1 || first.SecurityIDs[2] != 3 || len(first.RequestHash) != 64 {
		t.Fatalf("CreatePost() repository input = %#v", first)
	}

	var reordered domain.CreatePostParams
	repository.create = func(_ context.Context, input domain.CreatePostParams) (domain.Post, error) {
		reordered = input
		return domain.Post{}, domain.ErrIdempotencyConflict
	}
	_, err = service.CreatePost(context.Background(), CreatePostInput{
		UserID: 42, IdempotencyKey: "post-key-1", CircleID: 7,
		Title: "长期观察", Body: "正文", SecurityIDs: []int64{2, 3, 1},
	})
	if !errors.Is(err, domain.ErrIdempotencyConflict) || reordered.RequestHash != first.RequestHash {
		t.Fatalf("reordered request hash/error = %q, %v; first = %q", reordered.RequestHash, err, first.RequestHash)
	}
}

func TestCreatePostRejectsInvalidKeyTextAndSecuritySetsBeforeRepository(t *testing.T) {
	calls := 0
	repository := &fakePostRepository{create: func(context.Context, domain.CreatePostParams) (domain.Post, error) {
		calls++
		return domain.Post{}, nil
	}}
	service := mustPostService(t, repository)
	tests := []CreatePostInput{
		{UserID: 1, CircleID: 1, IdempotencyKey: "", Title: "标题", Body: "正文", SecurityIDs: []int64{1}},
		{UserID: 1, CircleID: 1, IdempotencyKey: "bad key", Title: "标题", Body: "正文", SecurityIDs: []int64{1}},
		{UserID: 1, CircleID: 1, IdempotencyKey: "key", Title: " ", Body: "正文", SecurityIDs: []int64{1}},
		{UserID: 1, CircleID: 1, IdempotencyKey: "key", Title: "标题", Body: "正文", SecurityIDs: nil},
		{UserID: 1, CircleID: 1, IdempotencyKey: "key", Title: "标题", Body: "正文", SecurityIDs: []int64{1, 1}},
		{UserID: 1, CircleID: 1, IdempotencyKey: "key", Title: "标题", Body: "正文", SecurityIDs: []int64{1, 2, 3, 4, 5, 6}},
	}
	for _, input := range tests {
		if _, err := service.CreatePost(context.Background(), input); err == nil {
			t.Fatalf("CreatePost(%#v) error = nil", input)
		}
	}
	if calls != 0 {
		t.Fatalf("repository calls = %d, want 0", calls)
	}
}

func TestListGetUpdateAndDeletePostsKeepPermissionsAndStablePage(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	repository := &fakePostRepository{
		list: func(_ context.Context, query domain.PostListQuery) ([]domain.Post, error) {
			if query.CircleID != 7 || query.SecurityID != 3 || query.Limit != 3 || query.After == nil || query.After.ID != 11 {
				t.Fatalf("ListPosts() query = %#v", query)
			}
			return []domain.Post{{ID: 10, CreatedAt: now}, {ID: 9, CreatedAt: now}, {ID: 8, CreatedAt: now.Add(-time.Second)}}, nil
		},
		find: func(_ context.Context, id int64) (domain.Post, error) {
			if id != 10 {
				t.Fatalf("FindVisiblePost() id = %d", id)
			}
			return domain.Post{ID: id}, nil
		},
		update: func(_ context.Context, input domain.UpdatePostParams) (domain.Post, error) {
			if input.ActorID != 42 || input.PostID != 10 || input.ExpectedVersion != 4 || input.Title == nil || *input.Title != "新标题" || !input.ReplaceSecurities {
				t.Fatalf("UpdatePost() input = %#v", input)
			}
			return domain.Post{}, domain.ErrVersionConflict
		},
		remove: func(_ context.Context, actorID, postID int64) error {
			if actorID != 42 || postID != 10 {
				t.Fatalf("DeletePost() ids = %d/%d", actorID, postID)
			}
			return domain.ErrForbidden
		},
	}
	service := mustPostService(t, repository)
	page, err := service.ListPosts(context.Background(), PostListInput{UserID: 42, CircleID: 7, SecurityID: 3, Limit: 2, After: &domain.PostCursor{CreatedAt: now.Add(time.Second), ID: 11}})
	if err != nil || len(page.Items) != 2 || page.Next == nil || page.Next.ID != 9 {
		t.Fatalf("ListPosts() = %#v, %v", page, err)
	}
	if _, err := service.GetPost(context.Background(), 42, 10); err != nil {
		t.Fatalf("GetPost() error = %v", err)
	}
	title := "  新标题  "
	securityIDs := []int64{5, 3}
	_, err = service.UpdatePost(context.Background(), UpdatePostInput{UserID: 42, PostID: 10, Version: 4, Title: &title, SecurityIDs: &securityIDs})
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("UpdatePost() error = %v", err)
	}
	if err := service.DeletePost(context.Background(), 42, 10); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("DeletePost() error = %v", err)
	}
}

type fakePostRepository struct {
	create func(context.Context, domain.CreatePostParams) (domain.Post, error)
	list   func(context.Context, domain.PostListQuery) ([]domain.Post, error)
	find   func(context.Context, int64) (domain.Post, error)
	update func(context.Context, domain.UpdatePostParams) (domain.Post, error)
	remove func(context.Context, int64, int64) error
}

func (repository *fakePostRepository) CreatePost(ctx context.Context, input domain.CreatePostParams) (domain.Post, error) {
	return repository.create(ctx, input)
}
func (repository *fakePostRepository) ListPosts(ctx context.Context, query domain.PostListQuery) ([]domain.Post, error) {
	return repository.list(ctx, query)
}
func (repository *fakePostRepository) FindVisiblePost(ctx context.Context, id int64) (domain.Post, error) {
	return repository.find(ctx, id)
}
func (repository *fakePostRepository) UpdatePost(ctx context.Context, input domain.UpdatePostParams) (domain.Post, error) {
	return repository.update(ctx, input)
}
func (repository *fakePostRepository) DeletePost(ctx context.Context, actorID, postID int64) error {
	return repository.remove(ctx, actorID, postID)
}

func mustPostService(t *testing.T, repository PostRepository) *PostService {
	t.Helper()
	service, err := NewPostService(repository)
	if err != nil {
		t.Fatalf("NewPostService() error = %v", err)
	}
	return service
}
