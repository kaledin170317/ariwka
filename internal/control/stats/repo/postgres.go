package repo

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"vpn/internal/control/stats/model"
)

type Postgres struct{ DB *sql.DB }

var builder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func (r Postgres) Add(ctx context.Context, deltas map[string]model.Usage) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for id, usage := range deltas {
		if usage.Uploaded == 0 && usage.Downloaded == 0 {
			continue
		}

		selected := sq.Select("id", "user_id", "device_id", "(inbound LIKE 'default:%')").
			Column(sq.Expr("?", usage.Uploaded)).
			Column(sq.Expr("?", usage.Downloaded)).
			From("profiles").
			Where(sq.Eq{"id": id})

		_, err = builder.
			Insert("traffic_usage").
			Columns("profile_id", "user_id", "device_id", "default_network", "uploaded", "downloaded").
			Select(selected).
			Suffix(`ON CONFLICT (profile_id) DO UPDATE SET
				uploaded = traffic_usage.uploaded + EXCLUDED.uploaded,
				downloaded = traffic_usage.downloaded + EXCLUDED.downloaded`).
			RunWith(tx).
			ExecContext(ctx)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r Postgres) Device(ctx context.Context, id string) (model.Usage, error) {
	var usage model.Usage

	err := builder.
		Select("COALESCE(SUM(uploaded), 0)", "COALESCE(SUM(downloaded), 0)").
		From("traffic_usage").
		Where(sq.Eq{"device_id": id}).
		RunWith(r.DB).
		QueryRowContext(ctx).
		Scan(&usage.Uploaded, &usage.Downloaded)

	return usage, err
}

func (r Postgres) User(ctx context.Context, id string, defaultNetwork bool) (model.Usage, error) {
	var usage model.Usage

	err := builder.
		Select("COALESCE(SUM(uploaded), 0)", "COALESCE(SUM(downloaded), 0)").
		From("traffic_usage").
		Where(sq.Eq{"user_id": id, "device_id": nil, "default_network": defaultNetwork}).
		RunWith(r.DB).
		QueryRowContext(ctx).
		Scan(&usage.Uploaded, &usage.Downloaded)

	return usage, err
}
