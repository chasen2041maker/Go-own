package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

type postQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

const postSelectColumns = `
p.id, p.circle_id, c.slug, c.name, p.author_id, u.display_name,
p.title, p.body,
(SELECT COUNT(*) FROM comments visible_comments
 WHERE visible_comments.post_id = p.id AND visible_comments.visibility = 'visible' AND visible_comments.deleted_at IS NULL) AS comment_count,
p.visibility, p.version, p.moderation_version, p.created_at, p.updated_at`

func (store *Store) CreatePost(ctx context.Context, input domain.CreatePostParams) (domain.Post, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Post{}, fmt.Errorf("begin create post transaction: %w", err)
	}
	defer tx.Rollback()

	if postID, hash, found, err := findIdempotentPost(ctx, tx, input.AuthorID, input.IdempotencyKey); err != nil {
		return domain.Post{}, err
	} else if found {
		if hash != input.RequestHash {
			return domain.Post{}, domain.ErrIdempotencyConflict
		}
		post, err := findPost(ctx, tx, postID, false)
		if err != nil {
			return domain.Post{}, err
		}
		if err := tx.Commit(); err != nil {
			return domain.Post{}, fmt.Errorf("commit replayed post transaction: %w", err)
		}
		return post, nil
	}
	if err := lockMembership(ctx, tx, input.CircleID, input.AuthorID); err != nil {
		return domain.Post{}, err
	}

	result, err := tx.ExecContext(ctx, `
INSERT INTO posts (circle_id, author_id, title, body, idempotency_key, request_hash)
VALUES (?, ?, ?, ?, ?, ?)`, input.CircleID, input.AuthorID, input.Title, input.Body, input.IdempotencyKey, input.RequestHash)
	if err != nil {
		if duplicateKey(err) {
			// 唯一键是并发幂等的最终裁判；先退出本事务，再读取已经提交的赢家资源。
			_ = tx.Rollback()
			return store.replayConcurrentPost(ctx, input.AuthorID, input.IdempotencyKey, input.RequestHash)
		}
		return domain.Post{}, fmt.Errorf("insert post: %w", err)
	}
	postID, err := result.LastInsertId()
	if err != nil {
		return domain.Post{}, fmt.Errorf("read inserted post id: %w", err)
	}
	// active 校验故意仍在同一事务中；任一标签失败会连刚插入的帖子一起回滚，不留下半成品。
	if err := validateActiveSecurities(ctx, tx, input.SecurityIDs); err != nil {
		return domain.Post{}, err
	}
	if err := replacePostSecurities(ctx, tx, postID, input.SecurityIDs, false); err != nil {
		return domain.Post{}, err
	}
	post, err := findPost(ctx, tx, postID, false)
	if err != nil {
		return domain.Post{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Post{}, fmt.Errorf("commit create post transaction: %w", err)
	}
	return post, nil
}

func (store *Store) ListPosts(ctx context.Context, query domain.PostListQuery) ([]domain.Post, error) {
	if query.Limit < 1 {
		return nil, fmt.Errorf("list posts: limit must be positive")
	}
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin list posts transaction: %w", err)
	}
	defer tx.Rollback()
	afterID := int64(0)
	afterCreatedAt := time.Unix(0, 0).UTC()
	if query.After != nil {
		afterID, afterCreatedAt = query.After.ID, query.After.CreatedAt.UTC()
	}
	rows, err := tx.QueryContext(ctx, `
SELECT `+postSelectColumns+`
FROM posts p
JOIN circles c ON c.id = p.circle_id
JOIN users u ON u.id = p.author_id
WHERE p.visibility = 'visible' AND p.deleted_at IS NULL
  AND (? = 0 OR p.circle_id = ?)
  AND (? = 0 OR EXISTS (SELECT 1 FROM post_securities filter_tags WHERE filter_tags.post_id = p.id AND filter_tags.security_id = ?))
  AND (? = 0 OR p.created_at < ? OR (p.created_at = ? AND p.id < ?))
ORDER BY p.created_at DESC, p.id DESC
LIMIT ?`, query.CircleID, query.CircleID, query.SecurityID, query.SecurityID,
		afterID, afterCreatedAt, afterCreatedAt, afterID, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("query visible posts: %w", err)
	}
	posts := make([]domain.Post, 0)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate posts: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close posts rows: %w", err)
	}
	if err := attachPostSecurities(ctx, tx, posts); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit list posts transaction: %w", err)
	}
	return posts, nil
}

func (store *Store) FindVisiblePost(ctx context.Context, postID int64) (domain.Post, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return domain.Post{}, fmt.Errorf("begin find post transaction: %w", err)
	}
	defer tx.Rollback()
	// 帖子正文与证券标签来自两条查询；同一只读事务确保它们属于同一个数据库快照。
	post, err := findPost(ctx, tx, postID, true)
	if err != nil {
		return domain.Post{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Post{}, fmt.Errorf("commit find post transaction: %w", err)
	}
	return post, nil
}

func (store *Store) UpdatePost(ctx context.Context, input domain.UpdatePostParams) (domain.Post, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Post{}, fmt.Errorf("begin update post transaction: %w", err)
	}
	defer tx.Rollback()
	var circleID, authorID, version int64
	var visibility string
	var deletedAt sql.NullTime
	err = tx.QueryRowContext(ctx, "SELECT circle_id, author_id, visibility, version, deleted_at FROM posts WHERE id=? FOR UPDATE", input.PostID).
		Scan(&circleID, &authorID, &visibility, &version, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) || deletedAt.Valid {
		return domain.Post{}, domain.ErrPostNotFound
	}
	if err != nil {
		return domain.Post{}, fmt.Errorf("lock post for update: %w", err)
	}
	if authorID != input.ActorID {
		return domain.Post{}, domain.ErrForbidden
	}
	if domain.Visibility(visibility) != domain.VisibilityVisible {
		return domain.Post{}, domain.ErrContentNotEditable
	}
	if err := lockMembership(ctx, tx, circleID, input.ActorID); err != nil {
		return domain.Post{}, err
	}
	if version != input.ExpectedVersion {
		return domain.Post{}, domain.ErrVersionConflict
	}
	if input.ReplaceSecurities {
		if err := validateActiveSecurities(ctx, tx, input.SecurityIDs); err != nil {
			return domain.Post{}, err
		}
	}
	assignments := []string{"version = version + 1", "updated_at = UTC_TIMESTAMP(6)"}
	arguments := make([]any, 0, 6)
	if input.Title != nil {
		assignments = append(assignments, "title = ?")
		arguments = append(arguments, *input.Title)
	}
	if input.Body != nil {
		assignments = append(assignments, "body = ?")
		arguments = append(arguments, *input.Body)
	}
	arguments = append(arguments, input.PostID, input.ExpectedVersion)
	result, err := tx.ExecContext(ctx, "UPDATE posts SET "+strings.Join(assignments, ", ")+" WHERE id=? AND version=? AND visibility='visible' AND deleted_at IS NULL", arguments...)
	if err != nil {
		return domain.Post{}, fmt.Errorf("update post content: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return domain.Post{}, fmt.Errorf("read post update result: %w", err)
	}
	if affected != 1 {
		return domain.Post{}, domain.ErrVersionConflict
	}
	if input.ReplaceSecurities {
		if err := replacePostSecurities(ctx, tx, input.PostID, input.SecurityIDs, true); err != nil {
			return domain.Post{}, err
		}
	}
	post, err := findPost(ctx, tx, input.PostID, false)
	if err != nil {
		return domain.Post{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Post{}, fmt.Errorf("commit update post transaction: %w", err)
	}
	return post, nil
}

func (store *Store) DeletePost(ctx context.Context, actorID, postID int64) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete post transaction: %w", err)
	}
	defer tx.Rollback()
	var authorID int64
	var visibility string
	var deletedAt sql.NullTime
	err = tx.QueryRowContext(ctx, "SELECT author_id, visibility, deleted_at FROM posts WHERE id=? FOR UPDATE", postID).Scan(&authorID, &visibility, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) || deletedAt.Valid {
		return domain.ErrPostNotFound
	}
	if err != nil {
		return fmt.Errorf("lock post for delete: %w", err)
	}
	if authorID != actorID {
		return domain.ErrForbidden
	}
	if domain.Visibility(visibility) != domain.VisibilityVisible {
		return domain.ErrContentNotEditable
	}
	// 作者删除只写 deleted_at，不篡改治理 visibility。与该目标相关的待办举报必须在同一事务
	// 收口，否则管理员队列会留下永远无法再处理的 visible 前提举报。
	result, err := tx.ExecContext(ctx, "UPDATE posts SET deleted_at=UTC_TIMESTAMP(6), updated_at=UTC_TIMESTAMP(6) WHERE id=? AND deleted_at IS NULL", postID)
	if err != nil {
		return fmt.Errorf("soft delete post: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read delete result: %w", err)
	}
	if affected != 1 {
		return domain.ErrPostNotFound
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE reports
SET status='resolved', resolution_action='author_deleted', handled_by=NULL,
    handled_at=UTC_TIMESTAMP(6), resolution_note=NULL, updated_at=UTC_TIMESTAMP(6)
WHERE post_id=? AND status='pending'`, postID); err != nil {
		return fmt.Errorf("close post reports after author delete: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete post transaction: %w", err)
	}
	return nil
}

func lockMembership(ctx context.Context, tx *sql.Tx, circleID, userID int64) error {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT cm.circle_id FROM circles c JOIN circle_memberships cm ON cm.circle_id=c.id
WHERE c.id=? AND c.status='active' AND cm.user_id=? FOR UPDATE`, circleID, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrMembershipRequired
	}
	if err != nil {
		return fmt.Errorf("lock circle membership: %w", err)
	}
	return nil
}

func validateActiveSecurities(ctx context.Context, tx *sql.Tx, securityIDs []int64) error {
	if len(securityIDs) < 1 || len(securityIDs) > 5 {
		return &domain.ValidationError{Field: "security_ids", Reason: "必须包含 1 到 5 个启用证券"}
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(securityIDs)), ",")
	args := make([]any, len(securityIDs))
	for i, id := range securityIDs {
		args[i] = id
	}
	rows, err := tx.QueryContext(ctx, "SELECT id FROM securities WHERE status='active' AND id IN ("+placeholders+") FOR UPDATE", args...)
	if err != nil {
		return fmt.Errorf("validate active securities: %w", err)
	}
	defer rows.Close()
	found := make(map[int64]struct{}, len(securityIDs))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan active security: %w", err)
		}
		found[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate active securities: %w", err)
	}
	if len(found) != len(securityIDs) {
		return &domain.ValidationError{Field: "security_ids", Reason: "包含不存在、停用或重复的证券"}
	}
	return nil
}

func replacePostSecurities(ctx context.Context, tx *sql.Tx, postID int64, securityIDs []int64, removeExisting bool) error {
	if removeExisting {
		if _, err := tx.ExecContext(ctx, "DELETE FROM post_securities WHERE post_id=?", postID); err != nil {
			return fmt.Errorf("delete existing post securities: %w", err)
		}
	}
	for _, securityID := range securityIDs {
		if _, err := tx.ExecContext(ctx, "INSERT INTO post_securities (post_id,security_id) VALUES (?,?)", postID, securityID); err != nil {
			return fmt.Errorf("insert post security: %w", err)
		}
	}
	return nil
}

func findIdempotentPost(ctx context.Context, queryer postQueryer, authorID int64, key string) (int64, string, bool, error) {
	var id int64
	var hash string
	err := queryer.QueryRowContext(ctx, "SELECT id, request_hash FROM posts WHERE author_id=? AND idempotency_key=?", authorID, key).Scan(&id, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, fmt.Errorf("find idempotent post: %w", err)
	}
	return id, hash, true, nil
}

func (store *Store) replayConcurrentPost(ctx context.Context, authorID int64, key, hash string) (domain.Post, error) {
	id, storedHash, found, err := findIdempotentPost(ctx, store.db, authorID, key)
	if err != nil {
		return domain.Post{}, err
	}
	if !found {
		return domain.Post{}, fmt.Errorf("idempotent winner disappeared")
	}
	if storedHash != hash {
		return domain.Post{}, domain.ErrIdempotencyConflict
	}
	return findPost(ctx, store.db, id, false)
}

func findPost(ctx context.Context, queryer postQueryer, postID int64, visibleOnly bool) (domain.Post, error) {
	statement := `SELECT ` + postSelectColumns + ` FROM posts p JOIN circles c ON c.id=p.circle_id JOIN users u ON u.id=p.author_id WHERE p.id=?`
	if visibleOnly {
		statement += " AND p.visibility='visible' AND p.deleted_at IS NULL"
	}
	post, err := scanPost(queryer.QueryRowContext(ctx, statement, postID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Post{}, domain.ErrPostNotFound
	}
	if err != nil {
		return domain.Post{}, err
	}
	posts := []domain.Post{post}
	if err := attachPostSecurities(ctx, queryer, posts); err != nil {
		return domain.Post{}, err
	}
	return posts[0], nil
}

type postScanner interface{ Scan(...any) error }

func scanPost(scanner postScanner) (domain.Post, error) {
	var post domain.Post
	var visibility string
	err := scanner.Scan(&post.ID, &post.Circle.ID, &post.Circle.Slug, &post.Circle.Name, &post.Author.ID, &post.Author.DisplayName, &post.Title, &post.Body, &post.CommentCount, &visibility, &post.Version, &post.ModerationVersion, &post.CreatedAt, &post.UpdatedAt)
	if err != nil {
		return domain.Post{}, err
	}
	post.Visibility = domain.Visibility(visibility)
	post.CreatedAt = post.CreatedAt.UTC()
	post.UpdatedAt = post.UpdatedAt.UTC()
	return post, nil
}

func attachPostSecurities(ctx context.Context, queryer postQueryer, posts []domain.Post) error {
	if len(posts) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(posts)), ",")
	args := make([]any, len(posts))
	byID := make(map[int64]int, len(posts))
	for i := range posts {
		args[i] = posts[i].ID
		byID[posts[i].ID] = i
		posts[i].Securities = []domain.Security{}
	}
	rows, err := queryer.QueryContext(ctx, `SELECT ps.post_id,s.id,s.code,s.name,s.market,s.status FROM post_securities ps JOIN securities s ON s.id=ps.security_id WHERE ps.post_id IN (`+placeholders+`) ORDER BY ps.post_id,s.code,s.id`, args...)
	if err != nil {
		return fmt.Errorf("query post securities: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		var security domain.Security
		var status string
		if err := rows.Scan(&postID, &security.ID, &security.Code, &security.Name, &security.Exchange, &status); err != nil {
			return fmt.Errorf("scan post security: %w", err)
		}
		security.Status = domain.SecurityStatus(status)
		index, ok := byID[postID]
		if !ok {
			return fmt.Errorf("post security references unexpected post %d", postID)
		}
		posts[index].Securities = append(posts[index].Securities, security)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate post securities: %w", err)
	}
	return nil
}

func duplicateKey(err error) bool {
	var mysqlError *drivermysql.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == 1062
}
