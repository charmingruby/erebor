package postgrex

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const (
	// DefaultWriteTimeout limits the duration of write queries.
	DefaultWriteTimeout = 6 * time.Second
	// DefaultReadTimeout limits the duration of read queries.
	DefaultReadTimeout = 4 * time.Second
)

// One scans a named query's first row into T, returning nil when no row exists.
func One[T any](ctx context.Context, stmts *Statements, name string, args ...any) (*T, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultReadTimeout)
	defer cancel()

	stmt, err := stmts.Stmt(name)
	if err != nil {
		return nil, err
	}

	var row T
	if err := stmt.QueryRowxContext(ctx, args...).StructScan(&row); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &row, nil
}

// Many scans all rows from a named query into a slice of T.
func Many[T any](ctx context.Context, stmts *Statements, name string, args ...any) ([]T, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultReadTimeout)
	defer cancel()

	stmt, err := stmts.Stmt(name)
	if err != nil {
		return nil, err
	}

	rows := make([]T, 0)
	if err := stmt.SelectContext(ctx, &rows, args...); err != nil {
		return nil, err
	}

	return rows, nil
}

// Exec executes a named statement with the default write timeout.
func Exec(ctx context.Context, stmts *Statements, name string, args ...any) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultWriteTimeout)
	defer cancel()

	stmt, err := stmts.Stmt(name)
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx, args...)

	return err
}

// ExecReturning executes a named write query and scans its returned value into T.
func ExecReturning[T any](ctx context.Context, stmts *Statements, name string, args ...any) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultWriteTimeout)
	defer cancel()

	var value T

	stmt, err := stmts.Stmt(name)
	if err != nil {
		return value, err
	}

	if err := stmt.QueryRowxContext(ctx, args...).Scan(&value); err != nil {
		return value, err
	}

	return value, nil
}
