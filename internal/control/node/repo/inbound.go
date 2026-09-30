package repo

import (
	"context"

	sq "github.com/Masterminds/squirrel"

	"vpn/internal/control/node/model"
)

func (r SQLite) CreateInbound(ctx context.Context, config model.InboundConfig) error {
	_, err := sq.
		Insert("node_inbounds").
		Columns("tag", "node_id", "type", "port").
		Values(config.Tag, config.NodeID, config.Type, config.Port).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}

func (r SQLite) ListInbounds(ctx context.Context, nodeID string) ([]model.InboundConfig, error) {
	rows, err := sq.
		Select("tag", "node_id", "type", "port").
		From("node_inbounds").
		Where(sq.Eq{"node_id": nodeID}).
		OrderBy("tag").
		RunWith(r.DB).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make([]model.InboundConfig, 0)

	for rows.Next() {
		var (
			config model.InboundConfig
			kind   int32
		)

		err := rows.Scan(&config.Tag, &config.NodeID, &kind, &config.Port)
		if err != nil {
			return nil, err
		}

		config.Type = model.InboundType(kind)
		configs = append(configs, config)
	}

	return configs, rows.Err()
}
