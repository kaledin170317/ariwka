package service

import (
	"context"

	"github.com/google/uuid"

	"vpn/internal/control/config/model"
	"vpn/internal/control/node/provider"
	usermodel "vpn/internal/control/user/model"
)

type Repository interface {
	Create(context.Context, model.Profile) error
	List(context.Context, string) ([]model.Profile, error)
}

type Nodes interface {
	Get(string) (*provider.Node, error)
}

type Users interface {
	Get(context.Context, string) (usermodel.User, error)
}

type Service struct {
	Repo  Repository
	Nodes Nodes
	Users Users
}

func (s Service) Create(ctx context.Context, userID, nodeID, inbound string) (string, error) {
	_, err := s.Users.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	node, err := s.Nodes.Get(nodeID)
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	err = node.AddUser(ctx, id, inbound)
	if err != nil {
		return "", err
	}
	err = s.Repo.Create(ctx, model.Profile{ID: id, UserID: userID, NodeID: nodeID, Inbound: inbound})
	if err != nil {
		_ = node.RemoveUser(ctx, id, inbound)
		return "", err
	}
	return id, nil
}

func (s Service) Get(ctx context.Context, userID string) ([]string, error) {
	profiles, err := s.Repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	connections := make([]string, 0, len(profiles))
	for _, p := range profiles {
		node, err := s.Nodes.Get(p.NodeID)
		if err != nil {
			return nil, err
		}
		link, err := node.Config(ctx, p.ID, p.Inbound)
		if err != nil {
			return nil, err
		}
		connections = append(connections, link)
	}
	return connections, nil
}
