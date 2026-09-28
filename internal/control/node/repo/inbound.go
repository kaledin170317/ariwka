package repo

import (
	"context"

	sq "github.com/Masterminds/squirrel"

	common "vpn/gen/vpn/common/v1"
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

func (r SQLite) DeleteInbound(ctx context.Context, tag string) error {
	_, err := sq.
		Delete("node_inbounds").
		Where(sq.Eq{"tag": tag}).
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
		var config model.InboundConfig
		var kind int32
		if err := rows.Scan(&config.Tag, &config.NodeID, &kind, &config.Port); err != nil {
			return nil, err
		}
		config.Type = common.InboundType(kind)
		configs = append(configs, config)
	}
	return configs, rows.Err()
}
