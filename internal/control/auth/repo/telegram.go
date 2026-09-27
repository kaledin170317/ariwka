package repo

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
)

func (r SQLite) TelegramUser(ctx context.Context, subject, newID string) (string, error) {
	var id string
	err := sq.
		Select("user_id").
		From("identities").
		Where(sq.Eq{"provider": "telegram", "subject": subject}).
		RunWith(r.DB).
		QueryRowContext(ctx).
		Scan(&id)
	if err == nil || !errors.Is(err, sql.ErrNoRows) {
		return id, err
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	_, err = sq.
		Insert("users").
		Columns("id", "username", "role").
		Values(newID, "telegram:"+subject, "user").
		RunWith(tx).
		ExecContext(ctx)
	if err != nil {
		return "", err
	}
	_, err = sq.
		Insert("identities").
		Columns("user_id", "provider", "subject").
		Values(newID, "telegram", subject).
		RunWith(tx).
		ExecContext(ctx)
	if err != nil {
		return "", err
	}
	return newID, tx.Commit()
}
