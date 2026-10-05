package service

import (
	"context"
	"sync"

	api "vpn/gen/node"
	"vpn/internal/control/node/model"
)

type Repository interface {
	List(context.Context, string) ([]*api.RouteRule, error)
	Create(context.Context, string, model.RouteRule) error
	Delete(context.Context, string, uint32) error
}

type RouteRuleService struct {
	Repo Repository
	Mu   *sync.Mutex
}

func (s RouteRuleService) RouteRules(ctx context.Context, nodeID string) ([]*api.RouteRule, error) {
	return s.Repo.List(ctx, nodeID)
}

func (s RouteRuleService) AddRouteRule(ctx context.Context, nodeID string, rule model.RouteRule) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	return s.Repo.Create(ctx, nodeID, rule)
}

func (s RouteRuleService) DeleteRouteRule(ctx context.Context, nodeID string, index uint32) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	return s.Repo.Delete(ctx, nodeID, index)
}
