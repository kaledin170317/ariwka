package repo

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"vpn/internal/control/user/model"
)

type SQLite struct{ DB *sql.DB }

func (r SQLite) CreatePassword(ctx context.Context, u model.User, hash string) error {
	_, err := sq.
		Insert("identities").
		Columns("user_id", "provider", "subject", "secret_hash").
		Values(u.ID, "password", u.Username, hash).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}

func (r SQLite) Password(ctx context.Context, username string) (string, string, error) {
	var id, hash string

	err := sq.
		Select("u.id", "i.secret_hash").
		From("identities i").
		Join("users u ON u.id = i.user_id").
		Where(sq.Eq{"i.provider": "password", "i.subject": username}).
		RunWith(r.DB).
		QueryRowContext(ctx).
		Scan(&id, &hash)

	return id, hash, err
}
