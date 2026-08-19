package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

func (store *Store) ListCircles(ctx context.Context, query domain.CircleListQuery) ([]domain.Circle, error) {
	if query.UserID <= 0 || query.Limit < 1 {
		return nil, fmt.Errorf("list circles: user id and limit must be positive")
	}
	afterID := int64(0)
	// MySQL DATETIME 从 1000 年开始；第一页用布尔哨兵关闭比较，但仍传一个可表示时间。
	afterCreatedAt := time.Unix(0, 0).UTC()
	if query.After != nil {
		afterID = query.After.ID
		afterCreatedAt = query.After.CreatedAt.UTC()
	}

	rows, err := store.db.QueryContext(ctx, `
SELECT
    c.id,
    c.slug,
    c.name,
    c.description,
    (SELECT COUNT(*) FROM circle_memberships all_members WHERE all_members.circle_id = c.id) AS member_count,
    EXISTS(
        SELECT 1 FROM circle_memberships current_member
        WHERE current_member.circle_id = c.id AND current_member.user_id = ?
    ) AS is_member,
    c.created_at
FROM circles c
WHERE c.status = 'active'
  AND (? = 0 OR c.created_at < ? OR (c.created_at = ? AND c.id < ?))
ORDER BY c.created_at DESC, c.id DESC
LIMIT ?`, query.UserID, afterID, afterCreatedAt, afterCreatedAt, afterID, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("query active circles: %w", err)
	}
	defer rows.Close()

	circles := make([]domain.Circle, 0)
	for rows.Next() {
		var circle domain.Circle
		if err := rows.Scan(
			&circle.ID, &circle.Slug, &circle.Name, &circle.Description,
			&circle.MemberCount, &circle.IsMember, &circle.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan circle: %w", err)
		}
		circle.CreatedAt = circle.CreatedAt.UTC()
		circles = append(circles, circle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate circles: %w", err)
	}
	return circles, nil
}

func (store *Store) SetCircleMembership(
	ctx context.Context,
	circleID int64,
	userID int64,
	joined bool,
) (domain.CircleMembership, error) {
	if circleID <= 0 || userID <= 0 {
		return domain.CircleMembership{}, fmt.Errorf("set circle membership: ids must be positive")
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.CircleMembership{}, fmt.Errorf("begin membership transaction: %w", err)
	}
	defer tx.Rollback()

	// 加入与退出先锁同一圈子行。这样同一圈子的状态变化按事务线性化，
	// 不会出现“加入已提交、读取 joined_at 前被另一个退出删除”的夹缝。
	var lockedCircleID int64
	if err := tx.QueryRowContext(ctx,
		"SELECT id FROM circles WHERE id = ? AND status = 'active' FOR UPDATE", circleID,
	).Scan(&lockedCircleID); errors.Is(err, sql.ErrNoRows) {
		return domain.CircleMembership{}, domain.ErrCircleNotFound
	} else if err != nil {
		return domain.CircleMembership{}, fmt.Errorf("lock active circle: %w", err)
	}

	var membership domain.CircleMembership
	if joined {
		membership, err = joinCircle(ctx, tx, circleID, userID)
	} else {
		membership, err = leaveCircle(ctx, tx, circleID, userID)
	}
	if err != nil {
		return domain.CircleMembership{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.CircleMembership{}, fmt.Errorf("commit membership transaction: %w", err)
	}
	return membership, nil
}

func joinCircle(ctx context.Context, tx *sql.Tx, circleID, userID int64) (domain.CircleMembership, error) {
	_, err := tx.ExecContext(ctx, `
INSERT INTO circle_memberships (circle_id, user_id) VALUES (?, ?)
ON DUPLICATE KEY UPDATE joined_at = circle_memberships.joined_at`, circleID, userID)
	if err != nil {
		return domain.CircleMembership{}, fmt.Errorf("join circle: %w", err)
	}

	var joinedAt time.Time
	err = tx.QueryRowContext(ctx, `
SELECT joined_at FROM circle_memberships WHERE circle_id = ? AND user_id = ?`, circleID, userID).Scan(&joinedAt)
	if err != nil {
		return domain.CircleMembership{}, fmt.Errorf("read joined membership: %w", err)
	}
	joinedAt = joinedAt.UTC()
	return domain.CircleMembership{
		CircleID: circleID, UserID: userID, Joined: true, JoinedAt: &joinedAt,
	}, nil
}

func leaveCircle(ctx context.Context, tx *sql.Tx, circleID, userID int64) (domain.CircleMembership, error) {
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM circle_memberships WHERE circle_id = ? AND user_id = ?", circleID, userID,
	); err != nil {
		return domain.CircleMembership{}, fmt.Errorf("leave circle: %w", err)
	}
	// DELETE 对不存在的行影响 0 行，这正是重复退出所需的幂等最终状态。
	return domain.CircleMembership{CircleID: circleID, UserID: userID, Joined: false}, nil
}
