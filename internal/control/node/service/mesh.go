package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	api "vpn/gen/node"
	common "vpn/gen/vpn/common"
	"vpn/internal/control/node/model"
)

const (
	meshPrefix     = "mesh:"
	meshPort       = 6767
	meshServerName = "www.cloudflare.com"
)

// SyncMesh maintains one private connection per node and an outbound to every other node.
func (s Service) SyncMesh(ctx context.Context) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	nodes, err := s.Repo.List(ctx)
	if err != nil {
		return err
	}

	var failures []error

	connections := make(map[string]model.Inbound, len(nodes))
	for _, n := range nodes {
		inbound, syncErr := s.meshInbound(ctx, n.ID)
		if syncErr != nil {
			failures = append(failures, fmt.Errorf("mesh inbound %s: %w", n.Name, syncErr))
			continue
		}

		connections[n.ID] = inbound
	}

	for _, source := range nodes {
		node, getErr := s.Get(source.ID)
		if getErr != nil {
			failures = append(failures, getErr)
			continue
		}

		current, listErr := node.Outbounds(ctx)
		if listErr != nil {
			failures = append(failures, fmt.Errorf("mesh outbounds %s: %w", source.Name, listErr))
			continue
		}

		existing := make(map[string]*api.Outbound, len(current))
		for _, outbound := range current {
			existing[outbound.GetTag()] = outbound
		}

		if direct, ok := existing["direct"]; ok {
			if direct.GetDirect() == nil {
				failures = append(failures, fmt.Errorf("node %s: outbound direct has wrong type", source.Name))
				continue
			}
		} else {
			err = node.AddOutbound(ctx, &api.Outbound{
				Tag:     "direct",
				Details: &api.Outbound_Direct{Direct: &api.DirectOutbound{}},
			})
			if err != nil {
				failures = append(failures, fmt.Errorf("node %s: add direct outbound: %w", source.Name, err))
				continue
			}
		}

		for _, target := range nodes {
			if source.ID == target.ID {
				continue
			}

			inbound, ok := connections[target.ID]
			if !ok {
				continue
			}

			config, configErr := s.meshConfig(ctx, target.ID)
			if configErr != nil {
				failures = append(failures, configErr)
				continue
			}

			outbound := meshOutbound(target.Address, inbound, config.InternalUUID)

			old, found := existing[outbound.Tag]
			if found && proto.Equal(old, outbound) {
				continue
			}

			if found {
				v := outbound.GetVless()

				updateErr := node.ModifyOutbound(ctx, outbound.Tag, []*api.OutboundOption{
					{Value: &api.OutboundOption_Server{Server: v.Server}},
					{Value: &api.OutboundOption_Port{Port: v.Port}},
					{Value: &api.OutboundOption_Uuid{Uuid: v.Uuid}},
					{Value: &api.OutboundOption_Vless{Vless: v.Inbound}},
				})
				if updateErr != nil {
					failures = append(failures, updateErr)
				}

				continue
			}

			addErr := node.AddOutbound(ctx, outbound)
			if addErr != nil {
				failures = append(failures, fmt.Errorf("mesh %s -> %s: %w", source.Name, target.Name, addErr))
			}
		}

		for _, outbound := range current {
			targetID, ok := strings.CutPrefix(outbound.GetTag(), meshPrefix)
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
				removeErr := node.RemoveOutbound(ctx, outbound.GetTag())
				if removeErr != nil {
					failures = append(failures, removeErr)
				}
			}
		}
	}

	return errors.Join(failures...)
}

func (s Service) meshConfig(ctx context.Context, nodeID string) (model.InboundConfig, error) {
	configs, err := s.Repo.ListInbounds(ctx, nodeID)
	if err != nil {
		return model.InboundConfig{}, err
	}

	for _, config := range configs {
		if config.Tag == meshPrefix+nodeID {
			return config, nil
		}
	}

	return model.InboundConfig{}, fmt.Errorf("mesh inbound %s not found", nodeID)
}

func (s Service) meshInbound(ctx context.Context, nodeID string) (model.Inbound, error) {
	node, err := s.Get(nodeID)
	if err != nil {
		return model.Inbound{}, err
	}

	config, err := s.meshConfig(ctx, nodeID)
	if err != nil {
		configs, listErr := s.Repo.ListInbounds(ctx, nodeID)
		if listErr != nil {
			return model.Inbound{}, listErr
		}

		for _, existing := range configs {
			if existing.Port == meshPort {
				return model.Inbound{}, fmt.Errorf("port %d occupied by %s", meshPort, existing.Tag)
			}
		}

		key := make([]byte, 32)

		_, err = rand.Read(key)
		if err != nil {
			return model.Inbound{}, err
		}

		short := make([]byte, 8)

		_, err = rand.Read(short)
		if err != nil {
			return model.Inbound{}, err
		}

		config = model.InboundConfig{
			Tag:          meshPrefix + nodeID,
			NodeID:       nodeID,
			Type:         model.InboundTypeVless,
			Port:         meshPort,
			ServerName:   meshServerName,
			Transport:    model.VlessTransportOptions{Type: model.VlessTransportTypeGrpc},
			ServiceName:  rand.Text(),
			PrivateKey:   base64.RawURLEncoding.EncodeToString(key),
			ShortID:      hex.EncodeToString(short),
			Network:      model.InboundNetworkInternal,
			InternalUUID: uuid.NewString(),
		}

		err = s.Repo.CreateInbound(ctx, config)
		if err != nil {
			return model.Inbound{}, err
		}
	}

	if config.Network != model.InboundNetworkInternal || config.InternalUUID == "" || config.Type != model.InboundTypeVless {
		return model.Inbound{}, fmt.Errorf("invalid mesh inbound %s", config.Tag)
	}

	inbounds, err := node.Inbounds(ctx)
	if err != nil {
		return model.Inbound{}, err
	}

	var inbound model.Inbound

	for _, current := range inbounds {
		if current.Tag == config.Tag {
			inbound = current
			break
		}
	}

	if inbound.Tag == "" {
		inbound, err = node.AddVlessInbound(ctx, config)
	} else {
		err = node.ModifyInbound(ctx, config, inbound)
	}

	if err != nil {
		return model.Inbound{}, err
	}

	// Read back the effective REALITY public key and gRPC service name after reconciliation.
	inbounds, err = node.Inbounds(ctx)
	if err != nil {
		return model.Inbound{}, err
	}

	for _, current := range inbounds {
		if current.Tag != config.Tag || current.Vless == nil {
			continue
		}

		for _, user := range current.Vless.UserDetails {
			if user.Tag == config.Tag {
				if user.UUID != config.InternalUUID {
					err = node.ModifyVlessUser(ctx, config.Tag, config.Tag, config.InternalUUID)
				}

				return current, err
			}
		}

		_, err = node.AddVlessUser(ctx, config.Tag, model.VlessUser{Tag: config.Tag, UUID: config.InternalUUID})

		return current, err
	}

	return model.Inbound{}, fmt.Errorf("mesh inbound %s missing", config.Tag)
}

func meshOutbound(address string, inbound model.Inbound, userID string) *api.Outbound {
	host, _, _ := net.SplitHostPort(address)
	v := inbound.Vless

	return &api.Outbound{
		Tag: inbound.Tag,
		Details: &api.Outbound_Vless{Vless: &api.VlessOutbound{
			Server: host,
			Port:   uint32(v.Port),
			Uuid:   userID,
			Inbound: &common.VlessInbound{
				ServerName: v.ServerName,
				PublicKey:  v.PublicKey,
				ShortId:    v.ShortID,
				Transport: &common.VlessTransport{
					Type:        common.VlessTransportType(v.Transport.Type),
					ServiceName: v.Transport.ServiceName,
					Host:        v.Transport.Host,
					Path:        v.Transport.Path,
				},
			},
		}},
	}
}
