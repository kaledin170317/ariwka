package repo

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"vpn/internal/control/node/model"
)

type SQLite struct{ DB *sql.DB }

func (r SQLite) Create(ctx context.Context, n model.Node) error {
	_, err := sq.
		Insert("nodes").
		Columns("id", "name", "address").
		Values(n.ID, n.Name, n.Address).
		RunWith(r.DB).
		ExecContext(ctx)
	return err
}

func (r SQLite) Delete(ctx context.Context, id string) error {
	_, err := sq.
		Delete("nodes").
		Where(sq.Eq{"id": id}).
		RunWith(r.DB).
		ExecContext(ctx)
	return err
}

func (r SQLite) List(ctx context.Context) ([]model.Node, error) {
	rows, err := sq.
		Select("id", "name", "address").
		From("nodes").
		OrderBy("name").
		RunWith(r.DB).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make([]model.Node, 0)
	for rows.Next() {
		var n model.Node
		err = rows.Scan(&n.ID, &n.Name, &n.Address)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}
