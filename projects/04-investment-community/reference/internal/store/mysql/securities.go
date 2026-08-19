package mysql

import (
	"context"
	"fmt"

	"go-own/projects/04-investment-community/reference/internal/domain"
)

func (store *Store) ListSecurities(ctx context.Context, query domain.SecurityListQuery) ([]domain.Security, error) {
	if query.Limit < 1 {
		return nil, fmt.Errorf("list securities: limit must be positive")
	}
	afterCode := ""
	var afterID int64
	if query.After != nil {
		afterCode = query.After.Code
		afterID = query.After.ID
	}

	rows, err := store.db.QueryContext(ctx, `
SELECT id, code, name, market, status
FROM securities
WHERE status = 'active'
  AND (
      ? = '' OR
      LEFT(code, CHAR_LENGTH(?)) = ? OR
      LEFT(name, CHAR_LENGTH(?)) = ?
  )
  AND (? = '' OR market = ?)
  AND (? = '' OR code > ? OR (code = ? AND id > ?))
ORDER BY code ASC, id ASC
LIMIT ?`,
		query.Query, query.Query, query.Query, query.Query, query.Query,
		query.Exchange, query.Exchange,
		afterCode, afterCode, afterCode, afterID,
		query.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query active securities: %w", err)
	}
	defer rows.Close()

	securities := make([]domain.Security, 0)
	for rows.Next() {
		var security domain.Security
		var status string
		if err := rows.Scan(&security.ID, &security.Code, &security.Name, &security.Exchange, &status); err != nil {
			return nil, fmt.Errorf("scan security: %w", err)
		}
		security.Status = domain.SecurityStatus(status)
		securities = append(securities, security)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate securities: %w", err)
	}
	return securities, nil
}
