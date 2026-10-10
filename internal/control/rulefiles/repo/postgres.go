package repo

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"vpn/internal/control/rulefiles/model"
)

type Postgres struct{ DB *sql.DB }

var builder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func (r Postgres) Create(ctx context.Context, item model.Item) error {
	result, err := builder.
		Insert("rule_files").
		Columns("tag", "geoip_url", "geosite_url", "active").
		Values(item.Tag, item.GeoIPURL, item.GeoSiteURL, true).
		Suffix(`ON CONFLICT (tag) DO UPDATE SET
			geoip_url = EXCLUDED.geoip_url, geosite_url = EXCLUDED.geosite_url,
			active = TRUE
			WHERE NOT rule_files.active`).
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
		return fmt.Errorf("rule files %s already exist", item.Tag)
	}

	return nil
}

func (r Postgres) Update(ctx context.Context, item model.Item) error {
	result, err := builder.
		Update("rule_files").
		Set("geoip_url", item.GeoIPURL).
		Set("geosite_url", item.GeoSiteURL).
		Where(sq.Eq{"tag": item.Tag, "active": true}).
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

func (r Postgres) Delete(ctx context.Context, tag string) error {
	result, err := builder.
		Update("rule_files").
		Set("active", false).
		Where(sq.Eq{"tag": tag, "active": true}).
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

func (r Postgres) Get(ctx context.Context, tag string) (model.Item, error) {
	var item model.Item

	err := builder.
		Select("tag", "geoip_url", "geosite_url", "active").
		From("rule_files").
		Where(sq.Eq{"tag": tag}).
		RunWith(r.DB).
		QueryRowContext(ctx).
		Scan(&item.Tag, &item.GeoIPURL, &item.GeoSiteURL, &item.Active)

	return item, err
}

func (r Postgres) List(ctx context.Context) ([]model.Item, error) {
	rows, err := builder.
		Select("tag", "geoip_url", "geosite_url", "active").
		From("rule_files").
		OrderBy("tag").
		RunWith(r.DB).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Item, 0)

	for rows.Next() {
		var item model.Item

		err = rows.Scan(&item.Tag, &item.GeoIPURL, &item.GeoSiteURL, &item.Active)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

func (r Postgres) Used(ctx context.Context, tag string) (bool, error) {
	var found string

	err := builder.
		Select("tag").
		From("routing_policies").
		Where(sq.Eq{"rule_file": tag, "active": true}).
		Limit(1).
		RunWith(r.DB).
		QueryRowContext(ctx).
		Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}

	return err == nil, err
}
