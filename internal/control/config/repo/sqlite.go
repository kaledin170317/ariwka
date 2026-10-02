package repo

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"vpn/internal/control/config/model"
)

type SQLite struct{ DB *sql.DB }

func (r SQLite) Create(ctx context.Context, p model.Profile) error {
	_, err := sq.
		Insert("profiles").
		Columns("id", "user_id", "node_id", "inbound", "password").
		Values(p.ID, p.UserID, p.NodeID, p.Inbound, p.Password).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}

func (r SQLite) Get(ctx context.Context, userID, id string) (model.Profile, error) {
	var p model.Profile

	err := sq.
		Select("id", "user_id", "node_id", "inbound", "password").
		From("profiles").
		Where(sq.Eq{"id": id, "user_id": userID}).
		RunWith(r.DB).
		QueryRowContext(ctx).
		Scan(&p.ID, &p.UserID, &p.NodeID, &p.Inbound, &p.Password)

	return p, err
}

func (r SQLite) List(ctx context.Context, userID string) ([]model.Profile, error) {
	rows, err := sq.
		Select("id", "user_id", "node_id", "inbound", "password").
		From("profiles").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("node_id", "inbound", "id").
		RunWith(r.DB).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]model.Profile, 0)

	for rows.Next() {
		var p model.Profile

		err = rows.Scan(&p.ID, &p.UserID, &p.NodeID, &p.Inbound, &p.Password)
		if err != nil {
			return nil, err
		}

		profiles = append(profiles, p)
	}

	return profiles, rows.Err()
}

func (r SQLite) ListByInbound(ctx context.Context, tag string) ([]model.Profile, error) {
	rows, err := sq.
		Select("id", "user_id", "node_id", "inbound", "password").
		From("profiles").
		Where(sq.Eq{"inbound": tag}).
		RunWith(r.DB).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]model.Profile, 0)

	for rows.Next() {
		var p model.Profile

		err = rows.Scan(&p.ID, &p.UserID, &p.NodeID, &p.Inbound, &p.Password)
		if err != nil {
			return nil, err
		}

		profiles = append(profiles, p)
	}

	return profiles, rows.Err()
}

func (r SQLite) SavePassword(ctx context.Context, id, password string) error {
	_, err := sq.
		Update("profiles").
		Set("password", password).
		Where(sq.Eq{"id": id, "password": ""}).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}

func (r SQLite) Delete(ctx context.Context, userID, id string) error {
	_, err := sq.
		Delete("profiles").
		Where(sq.Eq{"id": id, "user_id": userID}).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}
