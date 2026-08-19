package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	drivermysql "github.com/go-sql-driver/mysql"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

const selectUserColumns = "id, email, password_hash, display_name, role, status"

func (store *Store) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	result, err := store.db.ExecContext(ctx, `
INSERT INTO users (email, password_hash, display_name, role, status)
VALUES (?, ?, ?, ?, ?)`, user.Email, user.PasswordHash, user.DisplayName, user.Role, user.Status)
	if err != nil {
		var mysqlError *drivermysql.MySQLError
		if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
			// users 目前唯一业务键只有 email；在仓储边界把驱动错误翻译为稳定领域错误。
			return domain.User{}, domain.ErrEmailTaken
		}
		return domain.User{}, fmt.Errorf("insert user: %w", err)
	}
	userID, err := result.LastInsertId()
	if err != nil {
		return domain.User{}, fmt.Errorf("read inserted user id: %w", err)
	}
	user.ID = userID
	return user, nil
}

func (store *Store) FindUserByEmail(ctx context.Context, email string) (domain.User, error) {
	return scanUser(store.db.QueryRowContext(ctx,
		"SELECT "+selectUserColumns+" FROM users WHERE email = ?", email,
	))
}

func (store *Store) FindUserByID(ctx context.Context, userID int64) (domain.User, error) {
	return scanUser(store.db.QueryRowContext(ctx,
		"SELECT "+selectUserColumns+" FROM users WHERE id = ?", userID,
	))
}

func scanUser(row *sql.Row) (domain.User, error) {
	var user domain.User
	var role string
	var status string
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &role, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("scan user: %w", err)
	}
	user.Role = domain.Role(role)
	user.Status = domain.UserStatus(status)
	return user, nil
}
