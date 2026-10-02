package repo

import (
	"context"

	sq "github.com/Masterminds/squirrel"

	"vpn/internal/control/node/model"
)

func (r SQLite) CreateInbound(ctx context.Context, config model.InboundConfig) error {
	_, err := sq.
		Insert("node_inbounds").
		Columns("tag", "node_id", "type", "port", "server_name", "transport_type", "transport_host", "transport_path", "service_name", "private_key", "short_id").
		Values(config.Tag, config.NodeID, config.Type, config.Port, config.ServerName, config.Transport.Type, config.Transport.Host, config.Transport.Path, config.ServiceName, config.PrivateKey, config.ShortID).
		RunWith(r.DB).
		ExecContext(ctx)

	return err
}

func (r SQLite) ListInbounds(ctx context.Context, nodeID string) ([]model.InboundConfig, error) {
	rows, err := sq.
		Select("tag", "node_id", "type", "port", "server_name", "transport_type", "transport_host", "transport_path", "service_name", "private_key", "short_id").
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

		err := rows.Scan(&config.Tag, &config.NodeID, &kind, &config.Port, &config.ServerName, &config.Transport.Type, &config.Transport.Host, &config.Transport.Path, &config.ServiceName, &config.PrivateKey, &config.ShortID)
		if err != nil {
			return nil, err
		}

		config.Type = model.InboundType(kind)
		configs = append(configs, config)
	}

	return configs, rows.Err()
}

func (r SQLite) DeleteInbound(ctx context.Context, nodeID, tag string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = sq.Delete("profiles").
		Where(sq.Eq{"node_id": nodeID, "inbound": tag}).
		RunWith(tx).
		ExecContext(ctx)
	if err != nil {
		return err
	}

	_, err = sq.Delete("node_inbounds").
		Where(sq.Eq{"node_id": nodeID, "tag": tag}).
		RunWith(tx).
		ExecContext(ctx)
	if err != nil {
		return err
	}

	return tx.Commit()
}
