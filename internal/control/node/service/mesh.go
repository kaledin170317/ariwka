package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
)

const (
	meshPrefix     = "mesh:"
	meshPort       = 6767
	meshServerName = "www.cloudflare.com"
)

// SyncMesh connects every node to every other node through VLESS/REALITY.
func (s Service) SyncMesh(ctx context.Context) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	nodes, err := s.Repo.List(ctx)
	if err != nil {
		return err
	}

	connections := make(map[string]model.Inbound, len(nodes))

	var failures []error

	for _, entry := range nodes {
		inbound, err := s.meshInbound(ctx, entry.ID)
		if err != nil {
			failures = append(failures, fmt.Errorf("mesh inbound %s: %w", entry.Name, err))
			continue
		}

		connections[entry.ID] = inbound
	}

	for _, source := range nodes {
		node, err := s.Get(source.ID)
		if err != nil {
			failures = append(failures, err)
			continue
		}

		current, err := node.XrayOutbounds(ctx)
		if err != nil {
			failures = append(failures, fmt.Errorf("mesh outbounds %s: %w", source.Name, err))
			continue
		}

		existing := make(map[string]proto.Message, len(current))
		for _, outbound := range current {
			existing[outbound.Tag] = outbound
		}

		for _, target := range nodes {
			if target.ID == source.ID {
				continue
			}

			inbound, ok := connections[target.ID]
			if !ok {
				continue
			}

			config, err := s.meshConfig(ctx, target.ID)
			if err != nil {
				failures = append(failures, err)
				continue
			}

			outbound, err := provider.MeshOutbound(target.Address, inbound, config.InternalUUID)
			if err != nil {
				failures = append(failures, err)
				continue
			}

			if old, ok := existing[outbound.Tag]; ok {
				if proto.Equal(old, outbound) {
					continue
				}

				err = node.XrayRemoveOutbound(ctx, outbound.Tag)
				if err != nil {
					failures = append(failures, err)
					continue
				}
			}

			err = node.XrayAddOutbound(ctx, outbound)
			if err != nil {
				failures = append(failures, fmt.Errorf("mesh %s -> %s: %w", source.Name, target.Name, err))
			}
		}

		for _, outbound := range current {
			targetID, ok := strings.CutPrefix(outbound.Tag, meshPrefix)
			if !ok || targetID == source.ID {
				continue
			}

			alive := false

			for _, target := range nodes {
				if target.ID == targetID {
					alive = true
					break
				}
			}

			if !alive {
				err = node.XrayRemoveOutbound(ctx, outbound.Tag)
				if err != nil {
					failures = append(failures, err)
				}
			}
		}
	}

	return errors.Join(failures...)
}

func (s Service) meshConfig(ctx context.Context, nodeID string) (model.InboundConfig, error) {
	configs, err := s.Inbounds.ListInbounds(ctx, nodeID)
	if err != nil {
		return model.InboundConfig{}, err
	}

	for _, config := range configs {
		if config.Tag == meshPrefix+nodeID {
			return config, nil
		}
	}

	return model.InboundConfig{}, sql.ErrNoRows
}

func (s Service) meshInbound(ctx context.Context, nodeID string) (model.Inbound, error) {
	node, err := s.Get(nodeID)
	if err != nil {
		return model.Inbound{}, err
	}

	config, err := s.meshConfig(ctx, nodeID)
	if errors.Is(err, sql.ErrNoRows) {
		configs, listErr := s.Inbounds.ListInbounds(ctx, nodeID)
		if listErr != nil {
			return model.Inbound{}, listErr
		}

		for _, existing := range configs {
			if existing.Port == meshPort {
				return model.Inbound{}, fmt.Errorf("port %d occupied by %s", meshPort, existing.Tag)
			}
		}

		key := make([]byte, 32)
		short := make([]byte, 8)

		_, err = rand.Read(key)
		if err != nil {
			return model.Inbound{}, err
		}

		_, err = rand.Read(short)
		if err != nil {
			return model.Inbound{}, err
		}

		config = model.InboundConfig{Tag: meshPrefix + nodeID, NodeID: nodeID, Type: model.InboundTypeVless,
			Port: meshPort, ServerName: meshServerName, Transport: model.VlessTransportOptions{Type: model.VlessTransportTypeGrpc},
			ServiceName: rand.Text(), PrivateKey: base64.RawURLEncoding.EncodeToString(key), ShortID: hex.EncodeToString(short),
			Network: model.InboundNetworkInternal, InternalUUID: uuid.NewString()}

		err = s.Inbounds.CreateInbound(ctx, config)
		if err != nil {
			return model.Inbound{}, err
		}
	} else if err != nil {
		return model.Inbound{}, err
	}

	if config.Network != model.InboundNetworkInternal || config.Type != model.InboundTypeVless || config.InternalUUID == "" || config.PrivateKey == "" {
		return model.Inbound{}, fmt.Errorf("invalid mesh inbound %s", config.Tag)
	}

	inbounds, err := node.XrayInbounds(ctx, []model.InboundConfig{config})
	if err != nil {
		return model.Inbound{}, err
	}

	var inbound model.Inbound
	if len(inbounds) == 0 {
		inbound, err = node.XrayAddInbound(ctx, config)
		if err != nil {
			return model.Inbound{}, err
		}
	} else {
		inbound = inbounds[0]
	}

	for _, user := range inbound.Vless.UserDetails {
		if user.Tag == config.Tag {
			if user.UUID == config.InternalUUID {
				return inbound, nil
			}

			err = node.XrayRemoveUser(ctx, config.Tag, config.Tag)
			if err != nil {
				return model.Inbound{}, err
			}

			break
		}
	}

	err = node.XrayAddUser(ctx, config.Tag, model.VlessUser{Tag: config.Tag, UUID: config.InternalUUID}, "")

	return inbound, err
}
