package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	api "vpn/gen/node"
)

type Repository interface {
	List(context.Context, string) ([]*api.Outbound, error)
	Create(context.Context, string, *api.Outbound) error
	Delete(context.Context, string, string) error
	Update(context.Context, string, string, []*api.OutboundOption) error
}

type OutboundService struct {
	Repo Repository
	Mu   *sync.Mutex
}

func (s OutboundService) Outbounds(ctx context.Context, nodeID string) ([]*api.Outbound, error) {
	return s.Repo.List(ctx, nodeID)
}

func (s OutboundService) AddOutbound(ctx context.Context, nodeID string, outbound *api.Outbound) error {
	if outbound == nil || strings.HasPrefix(outbound.GetTag(), "mesh:") {
		return errors.New("reserved outbound tag")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	return s.Repo.Create(ctx, nodeID, outbound)
}

func (s OutboundService) DeleteOutbound(ctx context.Context, nodeID, tag string) error {
	if strings.HasPrefix(tag, "mesh:") {
		return errors.New("reserved outbound tag")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	return s.Repo.Delete(ctx, nodeID, tag)
}

func (s OutboundService) ModifyOutbound(ctx context.Context, nodeID, tag string, options []*api.OutboundOption) error {
	if strings.HasPrefix(tag, "mesh:") {
		return errors.New("reserved outbound tag")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	return s.Repo.Update(ctx, nodeID, tag, options)
}
