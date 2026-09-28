package service

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/google/uuid"

	common "vpn/gen/vpn/common/v1"
	"vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
)

type Repository interface {
	Create(context.Context, model.Node) error
	Delete(context.Context, string) error
	List(context.Context) ([]model.Node, error)
	CreateInbound(context.Context, model.InboundConfig) error
	DeleteInbound(context.Context, string) error
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
	n := model.Node{ID: uuid.NewString(), Name: name, Address: address}
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

func (s Service) AddInbound(ctx context.Context, nodeID string, kind common.InboundType, port int32) (string, error) {
	node, err := s.Get(nodeID)
	if err != nil {
		return "", err
	}

	config := model.InboundConfig{
		Tag:    uuid.NewString(),
		NodeID: nodeID,
		Type:   kind,
		Port:   port,
	}

	err = s.Repo.CreateInbound(ctx, config)
	if err != nil {
		return "", err
	}

	err = node.AddInbound(ctx, config.Tag, kind, port)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if cleanupErr := s.Repo.DeleteInbound(cleanupCtx, config.Tag); cleanupErr != nil {
			return "", errors.Join(err, cleanupErr)
		}
		return "", err
	}
	return config.Tag, nil
}
