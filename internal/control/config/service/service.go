package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/google/uuid"

	"vpn/internal/control/config/model"
	nodemodel "vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
	usermodel "vpn/internal/control/user/model"
)

type Repository interface {
	Create(context.Context, model.Profile) error
	Get(context.Context, string, string) (model.Profile, error)
	List(context.Context, string) ([]model.Profile, error)
	ListByInbound(context.Context, string) ([]model.Profile, error)
	SavePassword(context.Context, string, string) error
	Delete(context.Context, string, string) error
}

type Nodes interface {
	Get(string) (*provider.Node, error)
	List(context.Context) ([]nodemodel.Node, error)
}

type Inbounds interface {
	Inbounds(context.Context, string) ([]nodemodel.Inbound, error)
}

type Users interface {
	Get(context.Context, string) (usermodel.User, error)
}

type Service struct {
	Repo     Repository
	Nodes    Nodes
	Inbounds Inbounds
	Users    Users
	Mu       *sync.Mutex
}

func (s Service) Create(ctx context.Context, userID, nodeID, inbound string) (string, string, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	_, err := s.Users.Get(ctx, userID)
	if err != nil {
		return "", "", err
	}

	node, err := s.Nodes.Get(nodeID)
	if err != nil {
		return "", "", err
	}

	inbounds, err := s.Inbounds.Inbounds(ctx, nodeID)
	if err != nil {
		return "", "", err
	}

	var selected nodemodel.Inbound

	for _, item := range inbounds {
		if item.Tag == inbound && item.Network == nodemodel.InboundNetworkUsers {
			selected = item
			break
		}
	}

	if selected.Tag == "" {
		return "", "", fmt.Errorf("inbound %s not found", inbound)
	}

	id := uuid.NewString()

	password := ""
	if selected.Hysteria2 != nil {
		password = rand.Text()
	}

	profile := model.Profile{
		ID:       id,
		UserID:   userID,
		NodeID:   nodeID,
		Inbound:  inbound,
		Password: password,
	}

	err = s.Repo.Create(ctx, profile)
	if err != nil {
		return "", "", err
	}

	var created nodemodel.Inbound
	if selected.Hysteria2 != nil {
		created, err = node.AddHysteria2User(ctx, inbound, nodemodel.InboundUser{
			Name:     id,
			Password: password,
		})
	} else {
		created, err = node.AddVlessUser(ctx, inbound, nodemodel.VlessUser{
			Tag:  id,
			UUID: id,
		})
	}

	if err != nil {
		return "", "", err
	}

	connection, err := node.Link(created, id)
	if err != nil {
		return "", "", err
	}

	return id, connection, nil
}

func (s Service) Delete(ctx context.Context, userID, id string) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	profile, err := s.Repo.Get(ctx, userID, id)
	if err != nil {
		return err
	}

	node, err := s.Nodes.Get(profile.NodeID)
	if err != nil {
		return err
	}

	inbounds, err := node.Inbounds(ctx)
	if err != nil {
		return err
	}

	for _, inbound := range inbounds {
		if inbound.Tag == profile.Inbound {
			exists := false

			if h := inbound.Hysteria2; h != nil {
				for _, user := range h.Users {
					if user.Name == id {
						exists = true
						break
					}
				}
			} else if v := inbound.Vless; v != nil {
				for _, user := range v.UserDetails {
					if user.Tag == id {
						exists = true
						break
					}
				}

				if len(v.UserDetails) == 0 {
					exists = slices.Contains(v.Users, id)
				}
			}

			if exists {
				err = node.RemoveUser(ctx, id, profile.Inbound)
				if err != nil {
					return err
				}
			}

			break
		}
	}

	return s.Repo.Delete(ctx, userID, id)
}

func (s Service) SyncUsers(ctx context.Context) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return err
	}

	var failures []error

	for _, n := range nodes {
		err = s.syncNodeUsers(ctx, n.ID)
		if err != nil {
			failures = append(failures, fmt.Errorf("node %s: %w", n.Name, err))
		}
	}

	return errors.Join(failures...)
}

func (s Service) Get(ctx context.Context, userID string) ([]string, error) {
	profiles, err := s.GetProfiles(ctx, userID)
	if err != nil {
		return nil, err
	}

	connections := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		connections = append(connections, profile.Link)
	}

	return connections, nil
}

func (s Service) GetProfiles(ctx context.Context, userID string) ([]model.Connection, error) {
	profiles, err := s.Repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	connections := make([]model.Connection, 0, len(profiles))
	for _, p := range profiles {
		node, err := s.Nodes.Get(p.NodeID)
		if err != nil {
			return nil, err
		}

		link, err := node.Config(ctx, p.ID, p.Inbound)
		if err != nil {
			return nil, err
		}

		connections = append(connections, model.Connection{ID: p.ID, Link: link})
	}

	return connections, nil
}

func (s Service) GetProfile(ctx context.Context, userID, id string) (string, error) {
	p, err := s.Repo.Get(ctx, userID, id)
	if err != nil {
		return "", err
	}

	node, err := s.Nodes.Get(p.NodeID)
	if err != nil {
		return "", err
	}

	return node.Config(ctx, p.ID, p.Inbound)
}

func (s Service) syncNodeUsers(ctx context.Context, nodeID string) error {
	node, err := s.Nodes.Get(nodeID)
	if err != nil {
		return err
	}

	inbounds, err := node.Inbounds(ctx)
	if err != nil {
		return err
	}

	var failures []error

	for _, inbound := range inbounds {
		profiles, listErr := s.Repo.ListByInbound(ctx, inbound.Tag)
		if listErr != nil {
			failures = append(failures, listErr)
		} else {
			for _, profile := range profiles {
				if profile.NodeID == nodeID {
					err = s.syncProfile(ctx, node, inbound, profile)
					if err != nil {
						failures = append(failures, fmt.Errorf("user %s: %w", profile.ID, err))
					}
				}
			}
		}
	}

	return errors.Join(failures...)
}

func (s Service) syncProfile(ctx context.Context, node *provider.Node, inbound nodemodel.Inbound, profile model.Profile) error {
	if v := inbound.Vless; v != nil {
		for _, user := range v.UserDetails {
			if user.Tag == profile.ID {
				if user.UUID != profile.ID {
					return node.ModifyVlessUser(ctx, inbound.Tag, profile.ID, profile.ID)
				}

				return nil
			}
		}

		if len(v.UserDetails) == 0 && slices.Contains(v.Users, profile.ID) {
			return nil
		}

		_, err := node.AddVlessUser(ctx, inbound.Tag, nodemodel.VlessUser{
			Tag:  profile.ID,
			UUID: profile.ID,
		})

		return err
	}

	if h := inbound.Hysteria2; h != nil {
		for _, user := range h.Users {
			if user.Name == profile.ID {
				if profile.Password == "" {
					return s.Repo.SavePassword(ctx, profile.ID, user.Password)
				}

				if user.Password != profile.Password {
					return node.ModifyHysteria2User(ctx, inbound.Tag, profile.ID, profile.Password)
				}

				return nil
			}
		}

		if profile.Password == "" {
			return errors.New("no password stored")
		}

		_, err := node.AddHysteria2User(ctx, inbound.Tag, nodemodel.InboundUser{
			Name:     profile.ID,
			Password: profile.Password,
		})

		return err
	}

	return errors.New("unsupported inbound type")
}
