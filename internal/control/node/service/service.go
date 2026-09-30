package service

import (
	"context"
	"errors"
	"net"
	"time"

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

	created, err := node.AddHysteria2Inbound(ctx, port)
	if err != nil {
		return model.Inbound{}, err
	}

	return s.saveInbound(ctx, node, created, nodeID, port, model.InboundTypeHysteria2)
}

func (s Service) AddVlessInbound(ctx context.Context, nodeID string, port int32, serverName string, transport model.VlessTransportOptions) (model.Inbound, error) {
	node, err := s.inboundNode(nodeID, port)
	if err != nil {
		return model.Inbound{}, err
	}

	created, err := node.AddVlessInbound(ctx, port, serverName, transport)
	if err != nil {
		return model.Inbound{}, err
	}

	return s.saveInbound(ctx, node, created, nodeID, port, model.InboundTypeVless)
}

func (s Service) inboundNode(nodeID string, port int32) (*provider.Node, error) {
	if port < 1 || port > 65535 {
		return nil, errors.New("invalid inbound port")
	}

	return s.Get(nodeID)
}

func (s Service) saveInbound(ctx context.Context, node *provider.Node, created model.Inbound, nodeID string, port int32, kind model.InboundType) (model.Inbound, error) {
	if created.Tag == "" {
		return model.Inbound{}, errors.New("node returned empty inbound tag")
	}

	err := s.Repo.CreateInbound(ctx, model.InboundConfig{
		Tag:    created.Tag,
		NodeID: nodeID,
		Type:   kind,
		Port:   port,
	})
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		cleanupErr := node.RemoveInbound(cleanupCtx, created.Tag)
		if cleanupErr != nil {
			return model.Inbound{}, errors.Join(err, cleanupErr)
		}

		return model.Inbound{}, err
	}

	return created, nil
}
