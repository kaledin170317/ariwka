package repo

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"vpn/internal/control/routing/model"
)

type Postgres struct{ DB *sql.DB }

var builder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func (r Postgres) Create(ctx context.Context, p model.Policy) error {
	result, err := builder.
		Insert("routing_policies").
		Columns("tag", "rule_file", "rule_code", "source_country", "source_not_country", "target_country", "target_not_country", "active").
		Values(p.Tag, p.RuleFile, p.RuleCode, p.Source.Country, p.Source.NotCountry, p.Target.Country, p.Target.NotCountry, true).
		Suffix(`ON CONFLICT (tag) DO UPDATE SET
			rule_file = EXCLUDED.rule_file, rule_code = EXCLUDED.rule_code,
			source_country = EXCLUDED.source_country, source_not_country = EXCLUDED.source_not_country,
			target_country = EXCLUDED.target_country, target_not_country = EXCLUDED.target_not_country, active = TRUE
			WHERE NOT routing_policies.active`).
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
		return fmt.Errorf("route policy %s already exists", p.Tag)
	}

	return nil
}

func (r Postgres) Update(ctx context.Context, p model.Policy) error {
	result, err := builder.
		Update("routing_policies").
		Set("rule_file", p.RuleFile).
		Set("rule_code", p.RuleCode).
		Set("source_country", p.Source.Country).
		Set("source_not_country", p.Source.NotCountry).
		Set("target_country", p.Target.Country).
		Set("target_not_country", p.Target.NotCountry).
		Where(sq.Eq{"tag": p.Tag, "active": true}).
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
		Update("routing_policies").
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

func (r Postgres) List(ctx context.Context) ([]model.Policy, error) {
	rows, err := builder.
		Select("tag", "rule_file", "rule_code", "source_country", "source_not_country", "target_country", "target_not_country", "active").
		From("routing_policies").
		OrderBy("tag").
		RunWith(r.DB).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	policies := make([]model.Policy, 0)

	for rows.Next() {
		var p model.Policy

		err = rows.Scan(&p.Tag, &p.RuleFile, &p.RuleCode,
			&p.Source.Country, &p.Source.NotCountry, &p.Target.Country, &p.Target.NotCountry, &p.Active)
		if err != nil {
			return nil, err
		}

		policies = append(policies, p)
	}

	return policies, rows.Err()
}
