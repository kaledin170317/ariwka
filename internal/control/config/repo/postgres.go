package repo

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"vpn/internal/control/config/model"
)

type Postgres struct{ DB *sql.DB }

var builder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func (r Postgres) Create(ctx context.Context, p model.Profile) error {
	_, err := builder.
		Insert("profiles").
		Columns("id", "user_id", "node_id", "inbound", "password").
		Values(p.ID, p.UserID, p.NodeID, p.Inbound, p.Password).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}

func (r Postgres) Get(ctx context.Context, userID, id string) (model.Profile, error) {
	var p model.Profile

	err := builder.
		Select("id", "user_id", "node_id", "inbound", "password").
		From("profiles").
		Where(sq.Eq{"id": id, "user_id": userID}).
		RunWith(r.DB).
		QueryRowContext(ctx).
		Scan(&p.ID, &p.UserID, &p.NodeID, &p.Inbound, &p.Password)

	return p, err
}

func (r Postgres) List(ctx context.Context, userID string) ([]model.Profile, error) {
	rows, err := builder.
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

func (r Postgres) ListByInbound(ctx context.Context, tag string) ([]model.Profile, error) {
	rows, err := builder.
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

func (r Postgres) SavePassword(ctx context.Context, id, password string) error {
	_, err := builder.
		Update("profiles").
		Set("password", password).
		Where(sq.Eq{"id": id, "password": ""}).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}

func (r Postgres) Delete(ctx context.Context, userID, id string) error {
	_, err := builder.
		Delete("profiles").
		Where(sq.Eq{"id": id, "user_id": userID}).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}
