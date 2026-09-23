package repo

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"vpn/internal/control/user/model"
)

type SQLite struct{ DB *sql.DB }

func (r SQLite) Create(ctx context.Context, u model.User) (model.User, error) {
	_, err := sq.
		Insert("users").
		Columns("id", "username").
		Values(u.ID, u.Username).
		Suffix("ON CONFLICT(username) DO NOTHING").
		RunWith(r.DB).
		ExecContext(ctx)
	if err != nil {
		return model.User{}, err
	}
	err = sq.
		Select("id", "username", "role").
		From("users").
		Where(sq.Eq{"username": u.Username}).
		RunWith(r.DB).
		QueryRowContext(ctx).
		Scan(&u.ID, &u.Username, &u.Role)
	return u, err
}

func (r SQLite) List(ctx context.Context) ([]model.User, error) {
	rows, err := sq.
		Select("id", "username", "role").
		From("users").
		OrderBy("username").
		RunWith(r.DB).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]model.User, 0)
	for rows.Next() {
		var u model.User
		err = rows.Scan(&u.ID, &u.Username, &u.Role)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r SQLite) Get(ctx context.Context, id string) (model.User, error) {
	var u model.User
	err := sq.
		Select("id", "username", "role").
		From("users").
		Where(sq.Eq{"id": id}).
		RunWith(r.DB).
		QueryRowContext(ctx).
		Scan(&u.ID, &u.Username, &u.Role)
	return u, err
}
