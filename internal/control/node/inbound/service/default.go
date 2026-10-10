package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"vpn/internal/control/node/model"
)

// SyncDefault maintains a public inbound on every node, including newly added nodes.
func (s InboundService) SyncDefault(ctx context.Context) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return err
	}

	var failures []error

	for _, node := range nodes {
		err = s.ensureDefault(ctx, node.ID)
		if err != nil {
			failures = append(failures, fmt.Errorf("node %s: %w", node.Name, err))
		}
	}

	return errors.Join(failures...)
}

func (s InboundService) ensureDefault(ctx context.Context, nodeID string) error {
	configs, err := s.Repo.ListInbounds(ctx, nodeID)
	if err != nil {
		return err
	}

	tag := DefaultPrefix + nodeID
	for _, config := range configs {
		if config.Tag == tag {
			if config.Network != model.InboundNetworkDefault || config.Type != model.InboundTypeVless {
				return fmt.Errorf("invalid default inbound %s", tag)
			}

			return nil
		}

		if config.Port == defaultPort {
			return fmt.Errorf("default port %d occupied by %s", defaultPort, config.Tag)
		}
	}

	key := make([]byte, 32)

	_, err = rand.Read(key)
	if err != nil {
		return err
	}

	short := make([]byte, 8)

	_, err = rand.Read(short)
	if err != nil {
		return err
	}

	return s.Repo.CreateInbound(ctx, model.InboundConfig{
		Tag:         tag,
		NodeID:      nodeID,
		Type:        model.InboundTypeVless,
		Port:        defaultPort,
		ServerName:  "www.cloudflare.com",
		Transport:   model.VlessTransportOptions{Type: model.VlessTransportTypeGrpc},
		ServiceName: rand.Text(),
		PrivateKey:  base64.RawURLEncoding.EncodeToString(key),
		ShortID:     hex.EncodeToString(short),
		Network:     model.InboundNetworkDefault,
	})
}
