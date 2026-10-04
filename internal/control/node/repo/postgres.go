package repo

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"vpn/internal/control/node/model"
)

type Postgres struct{ DB *sql.DB }

var builder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func (r Postgres) Create(ctx context.Context, n model.Node) error {
	_, err := builder.
		Insert("nodes").
		Columns("id", "name", "address", "description", "country").
		Values(n.ID, n.Name, n.Address, n.Description, n.Country).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}

func (r Postgres) Update(ctx context.Context, n model.Node) error {
	result, err := builder.
		Update("nodes").
		Set("name", n.Name).
		Set("address", n.Address).
		Set("description", n.Description).
		Where(sq.Eq{"id": n.ID}).
		RunWith(r.DB).
		ExecContext(ctx)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r Postgres) Delete(ctx context.Context, id string) error {
	_, err := builder.
		Delete("nodes").
		Where(sq.Eq{"id": id}).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}

func (r Postgres) List(ctx context.Context) ([]model.Node, error) {
	rows, err := builder.
		Select("id", "name", "address", "description", "country").
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

		err = rows.Scan(&n.ID, &n.Name, &n.Address, &n.Description, &n.Country)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, n)
	}

	return nodes, rows.Err()
}
