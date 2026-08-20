package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

const commentSelectColumns = `c.id,c.post_id,c.parent_id,u.id,u.display_name,c.body,c.moderation_version,c.created_at,c.updated_at`

func (store *Store) CreateComment(ctx context.Context, input domain.CreateCommentParams) (domain.Comment, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Comment{}, fmt.Errorf("begin create comment transaction: %w", err)
	}
	defer tx.Rollback()

	// 先锁帖子，再锁用户+幂等键；所有评论创建使用同一顺序，避免多个 gap lock
	// 与帖子行锁形成死锁。帖子状态先作为快照读取，既有重放仍可在目标后来隐藏时返回原结果。
	var postAuthorID, circleID int64
	var postVisibility string
	var postDeletedAt sql.NullTime
	err = tx.QueryRowContext(ctx, `
SELECT author_id,circle_id,visibility,deleted_at FROM posts
WHERE id=? FOR UPDATE`, input.PostID).Scan(&postAuthorID, &circleID, &postVisibility, &postDeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Comment{}, domain.ErrPostNotFound
	}
	if err != nil {
		return domain.Comment{}, fmt.Errorf("lock comment post: %w", err)
	}

	commentID, storedHash, found, err := findIdempotentComment(ctx, tx, input.AuthorID, input.IdempotencyKey, true)
	if err != nil {
		return domain.Comment{}, err
	}
	if found {
		if storedHash != input.RequestHash {
			return domain.Comment{}, domain.ErrIdempotencyConflict
		}
		comment, err := findComment(ctx, tx, commentID)
		if err != nil {
			return domain.Comment{}, err
		}
		if err := tx.Commit(); err != nil {
			return domain.Comment{}, fmt.Errorf("commit comment replay: %w", err)
		}
		return comment, nil
	}
	if postVisibility != "visible" || postDeletedAt.Valid {
		return domain.Comment{}, domain.ErrPostNotFound
	}
	var membership int
	if err := tx.QueryRowContext(ctx,
		"SELECT 1 FROM circle_memberships WHERE circle_id=? AND user_id=?", circleID, input.AuthorID,
	).Scan(&membership); errors.Is(err, sql.ErrNoRows) {
		return domain.Comment{}, domain.ErrMembershipRequired
	} else if err != nil {
		return domain.Comment{}, fmt.Errorf("check comment membership: %w", err)
	}

	recipientID := postAuthorID
	notificationType := domain.NotificationTypeComment
	if input.ParentCommentID != nil {
		var parentPostID, parentAuthorID int64
		var parentParentID sql.NullInt64
		var parentVisibility string
		var parentDeletedAt sql.NullTime
		err := tx.QueryRowContext(ctx, `
SELECT post_id,author_id,parent_id,visibility,deleted_at
FROM comments WHERE id=? FOR UPDATE`, *input.ParentCommentID).
			Scan(&parentPostID, &parentAuthorID, &parentParentID, &parentVisibility, &parentDeletedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Comment{}, domain.ErrParentCommentInvalid
		}
		if err != nil {
			return domain.Comment{}, fmt.Errorf("lock parent comment: %w", err)
		}
		if parentPostID != input.PostID || parentParentID.Valid || parentVisibility != "visible" || parentDeletedAt.Valid {
			return domain.Comment{}, domain.ErrParentCommentInvalid
		}
		recipientID = parentAuthorID
		notificationType = domain.NotificationTypeReply
	}

	result, err := tx.ExecContext(ctx, `
INSERT INTO comments (post_id,author_id,parent_id,body,idempotency_key,request_hash)
VALUES (?,?,?,?,?,?)`, input.PostID, input.AuthorID, input.ParentCommentID, input.Body, input.IdempotencyKey, input.RequestHash)
	if err != nil {
		if duplicateKey(err) {
			return store.replayConcurrentComment(ctx, input.AuthorID, input.IdempotencyKey, input.RequestHash)
		}
		return domain.Comment{}, fmt.Errorf("insert comment: %w", err)
	}
	commentID, err = result.LastInsertId()
	if err != nil {
		return domain.Comment{}, fmt.Errorf("read inserted comment id: %w", err)
	}
	if recipientID != input.AuthorID {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO notifications (user_id,type,actor_user_id,post_id,comment_id)
VALUES (?,?,?,?,?)`, recipientID, notificationType, input.AuthorID, input.PostID, commentID); err != nil {
			return domain.Comment{}, fmt.Errorf("insert comment notification: %w", err)
		}
	}
	comment, err := findComment(ctx, tx, commentID)
	if err != nil {
		return domain.Comment{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Comment{}, fmt.Errorf("commit create comment transaction: %w", err)
	}
	return comment, nil
}

func (store *Store) replayConcurrentComment(ctx context.Context, authorID int64, key, hash string) (domain.Comment, error) {
	id, storedHash, found, err := findIdempotentComment(ctx, store.db, authorID, key, false)
	if err != nil {
		return domain.Comment{}, err
	}
	if !found {
		return domain.Comment{}, fmt.Errorf("idempotent comment winner disappeared")
	}
	if storedHash != hash {
		return domain.Comment{}, domain.ErrIdempotencyConflict
	}
	return findComment(ctx, store.db, id)
}

func findIdempotentComment(ctx context.Context, queryer postQueryer, authorID int64, key string, lock bool) (int64, string, bool, error) {
	statement := "SELECT id,request_hash FROM comments WHERE author_id=? AND idempotency_key=?"
	if lock {
		statement += " FOR UPDATE"
	}
	var id int64
	var hash string
	err := queryer.QueryRowContext(ctx, statement, authorID, key).Scan(&id, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, fmt.Errorf("find idempotent comment: %w", err)
	}
	return id, hash, true, nil
}

func (store *Store) ListComments(ctx context.Context, query domain.CommentListQuery) ([]domain.Comment, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin list comments transaction: %w", err)
	}
	defer tx.Rollback()
	var postExists int
	if err := tx.QueryRowContext(ctx,
		"SELECT 1 FROM posts WHERE id=? AND visibility='visible' AND deleted_at IS NULL", query.PostID,
	).Scan(&postExists); errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPostNotFound
	} else if err != nil {
		return nil, fmt.Errorf("check comment post: %w", err)
	}
	afterID := int64(0)
	afterTime := time.Unix(0, 0).UTC()
	if query.After != nil {
		afterID = query.After.ID
		afterTime = query.After.CreatedAt.UTC()
	}
	rows, err := tx.QueryContext(ctx, `
SELECT `+commentSelectColumns+`
FROM comments c
JOIN users u ON u.id=c.author_id
LEFT JOIN comments parent ON parent.id=c.parent_id
WHERE c.post_id=? AND c.visibility='visible' AND c.deleted_at IS NULL
  AND (c.parent_id IS NULL OR (parent.visibility='visible' AND parent.deleted_at IS NULL))
  AND (?=0 OR c.created_at>? OR (c.created_at=? AND c.id>?))
ORDER BY c.created_at ASC,c.id ASC LIMIT ?`, query.PostID, afterID, afterTime, afterTime, afterID, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("query comments: %w", err)
	}
	defer rows.Close()
	comments := make([]domain.Comment, 0)
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comments: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close comments: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit list comments transaction: %w", err)
	}
	return comments, nil
}

func (store *Store) DeleteComment(ctx context.Context, actorID, commentID int64) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete comment transaction: %w", err)
	}
	defer tx.Rollback()
	var authorID int64
	var visibility string
	var deletedAt sql.NullTime
	err = tx.QueryRowContext(ctx,
		"SELECT author_id,visibility,deleted_at FROM comments WHERE id=? FOR UPDATE", commentID,
	).Scan(&authorID, &visibility, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) || deletedAt.Valid {
		return domain.ErrCommentNotFound
	}
	if err != nil {
		return fmt.Errorf("lock comment for delete: %w", err)
	}
	if authorID != actorID {
		return domain.ErrForbidden
	}
	if visibility != "visible" {
		return domain.ErrContentNotEditable
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE comments SET deleted_at=UTC_TIMESTAMP(6),updated_at=UTC_TIMESTAMP(6) WHERE id=?", commentID,
	); err != nil {
		return fmt.Errorf("soft delete comment: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE reports SET status='resolved',resolution_action='author_deleted',handled_by=NULL,
handled_at=UTC_TIMESTAMP(6),resolution_note=NULL,updated_at=UTC_TIMESTAMP(6)
WHERE comment_id=? AND status='pending'`, commentID); err != nil {
		return fmt.Errorf("close comment reports after author delete: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete comment transaction: %w", err)
	}
	return nil
}

func findComment(ctx context.Context, queryer postQueryer, id int64) (domain.Comment, error) {
	comment, err := scanComment(queryer.QueryRowContext(ctx, `SELECT `+commentSelectColumns+`
FROM comments c JOIN users u ON u.id=c.author_id WHERE c.id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Comment{}, domain.ErrCommentNotFound
	}
	return comment, err
}

func scanComment(scanner postScanner) (domain.Comment, error) {
	var comment domain.Comment
	var parentID sql.NullInt64
	if err := scanner.Scan(&comment.ID, &comment.PostID, &parentID, &comment.Author.ID, &comment.Author.DisplayName,
		&comment.Body, &comment.ModerationVersion, &comment.CreatedAt, &comment.UpdatedAt); err != nil {
		return domain.Comment{}, err
	}
	if parentID.Valid {
		comment.ParentCommentID = &parentID.Int64
	}
	comment.CreatedAt = comment.CreatedAt.UTC()
	comment.UpdatedAt = comment.UpdatedAt.UTC()
	return comment, nil
}
