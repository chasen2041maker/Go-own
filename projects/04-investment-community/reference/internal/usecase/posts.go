package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

type PostRepository interface {
	CreatePost(context.Context, domain.CreatePostParams) (domain.Post, error)
	ListPosts(context.Context, domain.PostListQuery) ([]domain.Post, error)
	FindVisiblePost(context.Context, int64) (domain.Post, error)
	UpdatePost(context.Context, domain.UpdatePostParams) (domain.Post, error)
	DeletePost(context.Context, int64, int64) error
}

type CreatePostInput struct {
	UserID         int64
	IdempotencyKey string
	CircleID       int64
	Title          string
	Body           string
	SecurityIDs    []int64
}

type PostListInput struct {
	UserID     int64
	CircleID   int64
	SecurityID int64
	After      *domain.PostCursor
	Limit      int
}

type PostPage struct {
	Items []domain.Post
	Next  *domain.PostCursor
}

type UpdatePostInput struct {
	UserID      int64
	PostID      int64
	Version     int64
	Title       *string
	Body        *string
	SecurityIDs *[]int64
}

type PostService struct{ repository PostRepository }

func NewPostService(repository PostRepository) (*PostService, error) {
	if repository == nil {
		return nil, errors.New("post service repository is required")
	}
	return &PostService{repository: repository}, nil
}

func (service *PostService) CreatePost(ctx context.Context, input CreatePostInput) (domain.Post, error) {
	if err := validatePositiveID("user_id", input.UserID); err != nil {
		return domain.Post{}, err
	}
	if err := validatePositiveID("circle_id", input.CircleID); err != nil {
		return domain.Post{}, err
	}
	if err := validateIdempotencyKey(input.IdempotencyKey); err != nil {
		return domain.Post{}, err
	}
	title, err := normalizePostText("title", input.Title, 120)
	if err != nil {
		return domain.Post{}, err
	}
	body, err := normalizePostText("body", input.Body, 10000)
	if err != nil {
		return domain.Post{}, err
	}
	securityIDs, err := normalizeSecurityIDs(input.SecurityIDs)
	if err != nil {
		return domain.Post{}, err
	}
	hash, err := postRequestHash(input.UserID, input.CircleID, title, body, securityIDs)
	if err != nil {
		return domain.Post{}, fmt.Errorf("create post: fingerprint request: %w", err)
	}
	post, err := service.repository.CreatePost(ctx, domain.CreatePostParams{
		AuthorID: input.UserID, CircleID: input.CircleID, Title: title, Body: body,
		SecurityIDs: securityIDs, IdempotencyKey: input.IdempotencyKey, RequestHash: hash,
	})
	if err != nil {
		return domain.Post{}, fmt.Errorf("create post: %w", err)
	}
	return post, nil
}

func (service *PostService) ListPosts(ctx context.Context, input PostListInput) (PostPage, error) {
	if err := validatePositiveID("user_id", input.UserID); err != nil {
		return PostPage{}, err
	}
	if input.CircleID < 0 {
		return PostPage{}, &domain.ValidationError{Field: "circle_id", Reason: "必须是正整数"}
	}
	if input.SecurityID < 0 {
		return PostPage{}, &domain.ValidationError{Field: "security_id", Reason: "必须是正整数"}
	}
	limit, err := validatePageLimit(input.Limit)
	if err != nil {
		return PostPage{}, err
	}
	if input.After != nil && (input.After.ID <= 0 || input.After.CreatedAt.IsZero()) {
		return PostPage{}, &domain.ValidationError{Field: "cursor", Reason: "分页位置无效"}
	}
	items, err := service.repository.ListPosts(ctx, domain.PostListQuery{CircleID: input.CircleID, SecurityID: input.SecurityID, After: input.After, Limit: limit + 1})
	if err != nil {
		return PostPage{}, fmt.Errorf("list posts: %w", err)
	}
	page := PostPage{Items: items}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1]
		page.Next = &domain.PostCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	return page, nil
}

func (service *PostService) GetPost(ctx context.Context, userID, postID int64) (domain.Post, error) {
	if err := validatePositiveID("user_id", userID); err != nil {
		return domain.Post{}, err
	}
	if err := validatePositiveID("post_id", postID); err != nil {
		return domain.Post{}, err
	}
	post, err := service.repository.FindVisiblePost(ctx, postID)
	if err != nil {
		return domain.Post{}, fmt.Errorf("get post: %w", err)
	}
	return post, nil
}

func (service *PostService) UpdatePost(ctx context.Context, input UpdatePostInput) (domain.Post, error) {
	if err := validatePositiveID("user_id", input.UserID); err != nil {
		return domain.Post{}, err
	}
	if err := validatePositiveID("post_id", input.PostID); err != nil {
		return domain.Post{}, err
	}
	if err := validatePositiveID("version", input.Version); err != nil {
		return domain.Post{}, err
	}
	if input.Title == nil && input.Body == nil && input.SecurityIDs == nil {
		return domain.Post{}, &domain.ValidationError{Field: "request", Reason: "至少提交一个要修改的字段"}
	}
	params := domain.UpdatePostParams{ActorID: input.UserID, PostID: input.PostID, ExpectedVersion: input.Version}
	if input.Title != nil {
		value, err := normalizePostText("title", *input.Title, 120)
		if err != nil {
			return domain.Post{}, err
		}
		params.Title = &value
	}
	if input.Body != nil {
		value, err := normalizePostText("body", *input.Body, 10000)
		if err != nil {
			return domain.Post{}, err
		}
		params.Body = &value
	}
	if input.SecurityIDs != nil {
		values, err := normalizeSecurityIDs(*input.SecurityIDs)
		if err != nil {
			return domain.Post{}, err
		}
		params.SecurityIDs, params.ReplaceSecurities = values, true
	}
	post, err := service.repository.UpdatePost(ctx, params)
	if err != nil {
		return domain.Post{}, fmt.Errorf("update post: %w", err)
	}
	return post, nil
}

func (service *PostService) DeletePost(ctx context.Context, userID, postID int64) error {
	if err := validatePositiveID("user_id", userID); err != nil {
		return err
	}
	if err := validatePositiveID("post_id", postID); err != nil {
		return err
	}
	if err := service.repository.DeletePost(ctx, userID, postID); err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	return nil
}

func normalizePostText(field, raw string, maximum int) (string, error) {
	value := strings.TrimSpace(raw)
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < 1 || utf8.RuneCountInString(value) > maximum {
		return "", &domain.ValidationError{Field: field, Reason: fmt.Sprintf("长度必须为 1 到 %d 个字符", maximum)}
	}
	return value, nil
}

func normalizeSecurityIDs(values []int64) ([]int64, error) {
	if len(values) < 1 || len(values) > 5 {
		return nil, &domain.ValidationError{Field: "security_ids", Reason: "必须包含 1 到 5 个证券"}
	}
	result := append([]int64(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	for index, id := range result {
		if id <= 0 {
			return nil, &domain.ValidationError{Field: "security_ids", Reason: "每个证券 ID 都必须是正整数"}
		}
		if index > 0 && id == result[index-1] {
			return nil, &domain.ValidationError{Field: "security_ids", Reason: "证券 ID 不能重复"}
		}
	}
	return result, nil
}

func validateIdempotencyKey(value string) error {
	if len(value) < 1 || len(value) > 128 {
		return &domain.ValidationError{Field: "Idempotency-Key", Reason: "长度必须为 1 到 128 个可见 ASCII 字符"}
	}
	for index := range len(value) {
		if value[index] < '!' || value[index] > '~' {
			return &domain.ValidationError{Field: "Idempotency-Key", Reason: "只能包含可见 ASCII 字符"}
		}
	}
	return nil
}

func validatePositiveID(field string, value int64) error {
	if value <= 0 {
		return &domain.ValidationError{Field: field, Reason: "必须是正整数"}
	}
	return nil
}

func postRequestHash(userID, circleID int64, title, body string, securityIDs []int64) (string, error) {
	// 标签顺序不改变业务资源，因此指纹使用排序后的 ID；JSON 字段顺序和网络重试也不会制造假冲突。
	payload := struct {
		Operation   string  `json:"operation"`
		UserID      int64   `json:"user_id"`
		Path        string  `json:"path"`
		CircleID    int64   `json:"circle_id"`
		Title       string  `json:"title"`
		Body        string  `json:"body"`
		SecurityIDs []int64 `json:"security_ids"`
	}{
		Operation: "createPost", UserID: userID, Path: "/api/v1/posts", CircleID: circleID, Title: title, Body: body, SecurityIDs: securityIDs,
	}
	contents, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:]), nil
}
