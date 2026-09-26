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
		Columns("id", "user_id", "node_id", "inbound").
		Values(p.ID, p.UserID, p.NodeID, p.Inbound).
		RunWith(r.DB).
		ExecContext(ctx)
	return err
}

func (r SQLite) List(ctx context.Context, userID string) ([]model.Profile, error) {
	rows, err := sq.
		Select("id", "user_id", "node_id", "inbound").
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
		err = rows.Scan(&p.ID, &p.UserID, &p.NodeID, &p.Inbound)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}
