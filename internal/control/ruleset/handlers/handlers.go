package handlers

import (
	"context"
	"errors"

	api "vpn/gen/control"
	"vpn/internal/control/ruleset/model"
	"vpn/internal/control/ruleset/service"
)

type Handler struct {
	api.UnimplementedRuleSetServiceServer
	Service service.Service
}

func (h Handler) Create(ctx context.Context, req *api.CreateRuleSetRequest) (*api.CreateRuleSetResponse, error) {
	if req.GetRuleSet() == nil {
		return nil, errors.New("rule set required")
	}

	err := h.Service.Create(ctx, itemFromProto(req.GetRuleSet()))

	return &api.CreateRuleSetResponse{}, err
}

func (h Handler) Update(ctx context.Context, req *api.UpdateRuleSetRequest) (*api.UpdateRuleSetResponse, error) {
	if req.GetRuleSet() == nil {
		return nil, errors.New("rule set required")
	}

	err := h.Service.Update(ctx, itemFromProto(req.GetRuleSet()))

	return &api.UpdateRuleSetResponse{}, err
}

func (h Handler) Delete(ctx context.Context, req *api.DeleteRuleSetRequest) (*api.DeleteRuleSetResponse, error) {
	err := h.Service.Delete(ctx, req.GetTag())
	return &api.DeleteRuleSetResponse{}, err
}

func (h Handler) List(ctx context.Context, _ *api.ListRuleSetsRequest) (*api.ListRuleSetsResponse, error) {
	items, err := h.Service.List(ctx)
	if err != nil {
		return nil, err
	}

	resp := &api.ListRuleSetsResponse{RuleSets: make([]*api.ManagedRuleSet, 0, len(items))}
	for _, item := range items {
		resp.RuleSets = append(resp.RuleSets, &api.ManagedRuleSet{
			Tag:    item.Tag,
			Url:    item.URL,
			Format: api.RuleSetFormat(item.Format),
		})
	}

	return resp, nil
}

func itemFromProto(set *api.ManagedRuleSet) model.Item {
	return model.Item{
		Tag:    set.GetTag(),
		URL:    set.GetUrl(),
		Format: model.Format(set.GetFormat()),
	}
}
