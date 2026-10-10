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
	"sync"

	"github.com/google/uuid"

	"vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
)

type Repository interface {
	CreateInbound(context.Context, model.InboundConfig) error
	ListInbounds(context.Context, string) ([]model.InboundConfig, error)
	SaveInbound(context.Context, model.InboundConfig) error
	DeleteInbound(context.Context, string, string) error
}

type InboundNodes interface {
	List(context.Context) ([]model.Node, error)
	Get(string) (*provider.Node, error)
}

type InboundService struct {
	Repo  Repository
	Nodes InboundNodes
	Mu    *sync.Mutex
}

type NodeInbound struct {
	NodeID  string
	Inbound model.Inbound
}

const (
	DefaultPrefix = "default:"
	defaultPort   = 1489
)

func (s InboundService) ListInbounds(ctx context.Context) ([]NodeInbound, error) {
	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return nil, err
	}

	var result []NodeInbound

	for _, node := range nodes {
		inbounds, err := s.Inbounds(ctx, node.ID)
		if err != nil {
			return nil, err
		}

		for _, inbound := range inbounds {
			result = append(result, NodeInbound{NodeID: node.ID, Inbound: inbound})
		}
	}

	return result, nil
}

func (s InboundService) Inbounds(ctx context.Context, nodeID string) ([]model.Inbound, error) {
	node, err := s.Nodes.Get(nodeID)
	if err != nil {
		return nil, err
	}

	configs, err := s.Repo.ListInbounds(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	return node.XrayInbounds(ctx, configs)
}

func (s InboundService) AddHysteria2Inbound(ctx context.Context, nodeID string, port int32) (model.Inbound, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	node, err := s.inboundNode(nodeID, port)
	if err != nil {
		return model.Inbound{}, err
	}

	config := model.InboundConfig{
		Tag:     uuid.NewString(),
		NodeID:  nodeID,
		Type:    model.InboundTypeHysteria2,
		Port:    port,
		Network: model.InboundNetworkUsers,
	}

	err = s.Repo.CreateInbound(ctx, config)
	if err != nil {
		return model.Inbound{}, err
	}

	created, err := node.XrayAddInbound(ctx, config)
	if err != nil {
		_ = s.Repo.DeleteInbound(ctx, nodeID, config.Tag)
	}

	return created, err
}

func (s InboundService) AddVlessInbound(ctx context.Context, nodeID string, port int32, serverName string, transport model.VlessTransportOptions) (model.Inbound, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	node, err := s.inboundNode(nodeID, port)
	if err != nil {
		return model.Inbound{}, err
	}

	if serverName == "" || !transport.Type.IsValid() {
		return model.Inbound{}, errors.New("server name and valid transport required")
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

	config := model.InboundConfig{
		Tag:        uuid.NewString(),
		NodeID:     nodeID,
		Type:       model.InboundTypeVless,
		Port:       port,
		ServerName: serverName,
		Transport:  transport,
		PrivateKey: base64.RawURLEncoding.EncodeToString(key),
		ShortID:    hex.EncodeToString(short),
		Network:    model.InboundNetworkUsers,
	}
	if transport.Type == model.VlessTransportTypeGrpc {
		config.ServiceName = rand.Text()
	}

	err = s.Repo.CreateInbound(ctx, config)
	if err != nil {
		return model.Inbound{}, err
	}

	created, err := node.XrayAddInbound(ctx, config)
	if err != nil {
		_ = s.Repo.DeleteInbound(ctx, nodeID, config.Tag)
	}

	return created, err
}

func (s InboundService) DeleteInbound(ctx context.Context, nodeID, tag string) error {
	if strings.HasPrefix(tag, "mesh:") || strings.HasPrefix(tag, DefaultPrefix) {
		return errors.New("reserved inbound tag")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	configs, err := s.Repo.ListInbounds(ctx, nodeID)
	if err != nil {
		return err
	}

	found := false

	for _, config := range configs {
		if config.Tag == tag {
			found = true
			break
		}
	}

	if !found {
		return sql.ErrNoRows
	}

	node, err := s.Nodes.Get(nodeID)
	if err != nil {
		return err
	}

	inbounds, err := node.XrayInbounds(ctx, configs)
	if err != nil {
		return err
	}

	for _, inbound := range inbounds {
		if inbound.Tag == tag {
			err = node.XrayRemoveInbound(ctx, tag)
			if err != nil {
				return err
			}

			break
		}
	}

	return s.Repo.DeleteInbound(ctx, nodeID, tag)
}

func (s InboundService) SyncInbounds(ctx context.Context) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return err
	}

	var failures []error

	for _, n := range nodes {
		err = s.syncNode(ctx, n.ID)
		if err != nil {
			failures = append(failures, fmt.Errorf("node %s: %w", n.Name, err))
		}
	}

	return errors.Join(failures...)
}

func (s InboundService) inboundNode(nodeID string, port int32) (*provider.Node, error) {
	if port < 1 || port > 65535 {
		return nil, errors.New("invalid inbound port")
	}

	return s.Nodes.Get(nodeID)
}

func (s InboundService) syncNode(ctx context.Context, nodeID string) error {
	node, err := s.Nodes.Get(nodeID)
	if err != nil {
		return err
	}

	desired, err := s.Repo.ListInbounds(ctx, nodeID)
	if err != nil {
		return err
	}

	for i := range desired {
		c := &desired[i]
		if c.Network == model.InboundNetworkInternal || c.Type != model.InboundTypeVless || c.PrivateKey != "" {
			continue
		}

		key := make([]byte, 32)
		short := make([]byte, 8)

		_, err = rand.Read(key)
		if err != nil {
			return err
		}

		_, err = rand.Read(short)
		if err != nil {
			return err
		}

		c.PrivateKey = base64.RawURLEncoding.EncodeToString(key)

		c.ShortID = hex.EncodeToString(short)
		if c.ServerName == "" {
			c.ServerName = "www.cloudflare.com"
		}

		if !c.Transport.Type.IsValid() {
			c.Transport.Type = model.VlessTransportTypeGrpc
		}

		if c.Transport.Type == model.VlessTransportTypeGrpc && c.ServiceName == "" {
			c.ServiceName = rand.Text()
		}

		err = s.Repo.SaveInbound(ctx, *c)
		if err != nil {
			return err
		}
	}

	current, err := node.XrayInbounds(ctx, desired)
	if err != nil {
		return err
	}

	tags := make(map[string]model.Inbound, len(current))
	for _, inbound := range current {
		tags[inbound.Tag] = inbound
	}

	var failures []error

	for _, config := range desired {
		if config.Network == model.InboundNetworkInternal {
			continue
		}

		if _, ok := tags[config.Tag]; ok {
			if config.Network == model.InboundNetworkUsers {
				err = node.XrayEnableSniffing(ctx, config.Tag)
				if err != nil {
					failures = append(failures, fmt.Errorf("inbound %s sniffing: %w", config.Tag, err))
				}
			}

			continue
		}

		switch config.Type {
		case model.InboundTypeHysteria2, model.InboundTypeVless:
			_, err = node.XrayAddInbound(ctx, config)
		default:
			err = fmt.Errorf("unsupported inbound type %d", config.Type)
		}

		if err != nil {
			failures = append(failures, fmt.Errorf("inbound %s: %w", config.Tag, err))
		}
	}

	return errors.Join(failures...)
}
