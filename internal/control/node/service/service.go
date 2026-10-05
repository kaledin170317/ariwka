package service

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/google/uuid"

	controlapi "vpn/gen/control"
	"vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
	"vpn/pkg/country"
)

type Repository interface {
	Create(context.Context, model.Node) error
	Update(context.Context, model.Node) error
	Delete(context.Context, string) error
	List(context.Context) ([]model.Node, error)
}

type InboundRepository interface {
	CreateInbound(context.Context, model.InboundConfig) error
	ListInbounds(context.Context, string) ([]model.InboundConfig, error)
	DeleteInbound(context.Context, string, string) error
}

type Service struct {
	Repo     Repository
	Inbounds InboundRepository
	Provider *provider.Provider
	Country  country.Service
	Mu       *sync.Mutex
}

func (s Service) Create(ctx context.Context, name, address, description string) (model.Node, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	if name == "" {
		return model.Node{}, errors.New("name required")
	}

	host, err := nodeHost(address)
	if err != nil {
		return model.Node{}, err
	}

	code, err := s.Country.Code(ctx, host)
	if err != nil {
		return model.Node{}, err
	}

	n := model.Node{
		ID:          uuid.NewString(),
		Name:        name,
		Address:     address,
		Description: description,
		Country:     code,
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

func (s Service) Modify(ctx context.Context, id string, options []*controlapi.NodeOption) (model.Node, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	if len(options) == 0 {
		return model.Node{}, errors.New("node options required")
	}

	current, err := s.Provider.Get(id)
	if err != nil {
		return model.Node{}, err
	}

	n := current.Node

	for _, option := range options {
		switch value := option.GetValue().(type) {
		case *controlapi.NodeOption_Name:
			if value.Name == "" {
				return model.Node{}, errors.New("name required")
			}

			n.Name = value.Name
		case *controlapi.NodeOption_Address:
			_, err = nodeHost(value.Address)
			if err != nil {
				return model.Node{}, err
			}

			n.Address = value.Address
		case *controlapi.NodeOption_Description:
			n.Description = value.Description
		default:
			return model.Node{}, errors.New("invalid node option")
		}
	}

	err = s.Repo.Update(ctx, n)
	if err != nil {
		return model.Node{}, err
	}

	err = s.Provider.Add(n)
	if err != nil {
		_ = s.Repo.Update(ctx, current.Node)
		return model.Node{}, err
	}

	return n, nil
}

func (s Service) Delete(ctx context.Context, id string) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

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

func nodeHost(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		return "", errors.New("address must be host:port")
	}

	return host, nil
}
