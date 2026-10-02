package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"

	"github.com/google/uuid"

	"vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
)

type Repository interface {
	Create(context.Context, model.Node) error
	Delete(context.Context, string) error
	List(context.Context) ([]model.Node, error)
	CreateInbound(context.Context, model.InboundConfig) error
	ListInbounds(context.Context, string) ([]model.InboundConfig, error)
}

type Service struct {
	Repo     Repository
	Provider *provider.Provider
}

func (s Service) Create(ctx context.Context, name, address string) (model.Node, error) {
	if name == "" || address == "" {
		return model.Node{}, errors.New("name and address required")
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		return model.Node{}, errors.New("address must be host:port")
	}

	n := model.Node{
		ID:      uuid.NewString(),
		Name:    name,
		Address: address,
	}

	err = s.Repo.Create(ctx, n)
	if err != nil {
		return model.Node{}, err
	}

	err = s.Provider.Add(n)
	if err != nil {
		_ = s.Repo.Delete(ctx, n.ID)
		return model.Node{}, err
	}

	return n, nil
}

func (s Service) Delete(ctx context.Context, id string) error {
	err := s.Repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	s.Provider.Remove(id)

	return nil
}
func (s Service) List(ctx context.Context) ([]model.Node, error) {
	return s.Repo.List(ctx)
}

func (s Service) Get(id string) (*provider.Node, error) {
	return s.Provider.Get(id)
}

func (s Service) Inbounds(ctx context.Context, nodeID string) ([]model.Inbound, error) {
	node, err := s.Get(nodeID)
	if err != nil {
		return nil, err
	}

	return node.Inbounds(ctx)
}

func (s Service) AddHysteria2Inbound(ctx context.Context, nodeID string, port int32) (model.Inbound, error) {
	node, err := s.inboundNode(nodeID, port)
	if err != nil {
		return model.Inbound{}, err
	}

	config := model.InboundConfig{
		Tag:    uuid.NewString(),
		NodeID: nodeID,
		Type:   model.InboundTypeHysteria2,
		Port:   port,
	}

	err = s.Repo.CreateInbound(ctx, config)
	if err != nil {
		return model.Inbound{}, err
	}

	return node.AddHysteria2Inbound(ctx, config.Tag, port)
}

func (s Service) AddVlessInbound(ctx context.Context, nodeID string, port int32, serverName string, transport model.VlessTransportOptions) (model.Inbound, error) {
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
	}
	if transport.Type == model.VlessTransportTypeGrpc {
		config.ServiceName = rand.Text()
	}

	err = s.Repo.CreateInbound(ctx, config)
	if err != nil {
		return model.Inbound{}, err
	}

	return node.AddVlessInbound(ctx, config)
}

func (s Service) SyncInbounds(ctx context.Context) error {
	nodes, err := s.Repo.List(ctx)
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

func (s Service) inboundNode(nodeID string, port int32) (*provider.Node, error) {
	if port < 1 || port > 65535 {
		return nil, errors.New("invalid inbound port")
	}

	return s.Get(nodeID)
}

func (s Service) syncNode(ctx context.Context, nodeID string) error {
	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	desired, err := s.Repo.ListInbounds(ctx, nodeID)
	if err != nil {
		return err
	}

	current, err := node.Inbounds(ctx)
	if err != nil {
		return err
	}

	tags := make(map[string]model.Inbound, len(current))
	for _, inbound := range current {
		tags[inbound.Tag] = inbound
	}

	var failures []error

	for _, config := range desired {
		if existing, ok := tags[config.Tag]; ok {
			err = node.ModifyInbound(ctx, config, existing)
		} else {
			switch config.Type {
			case model.InboundTypeHysteria2:
				_, err = node.AddHysteria2Inbound(ctx, config.Tag, config.Port)
			case model.InboundTypeVless:
				if config.PrivateKey != "" {
					_, err = node.AddVlessInbound(ctx, config)
				} else {
					err = errors.New("no private key stored")
				}
			default:
				err = fmt.Errorf("unsupported inbound type %d", config.Type)
			}
		}

		if err != nil {
			failures = append(failures, fmt.Errorf("inbound %s: %w", config.Tag, err))
		}
	}

	return errors.Join(failures...)
}
