// Package mysql 实现 reference 用例所需的 MySQL 仓储。
package mysql

import (
	"database/sql"
	"fmt"
)

type Store struct {
	db *sql.DB
}

func New(database *sql.DB) (*Store, error) {
	if database == nil {
		return nil, fmt.Errorf("mysql store: database is required")
	}
	return &Store{db: database}, nil
}
