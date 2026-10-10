package handlers

import (
	"context"
	"errors"

	api "vpn/gen/control"
	"vpn/internal/control/rulefiles/model"
	"vpn/internal/control/rulefiles/service"
)

type Handler struct {
	api.UnimplementedRuleFilesServiceServer
	Service service.Service
}

func (h Handler) Create(ctx context.Context, req *api.CreateRuleFilesRequest) (*api.CreateRuleFilesResponse, error) {
	if req.GetFiles() == nil {
		return nil, errors.New("rule files required")
	}

	err := h.Service.Create(ctx, itemFromProto(req.GetFiles()))

	return &api.CreateRuleFilesResponse{}, err
}

func (h Handler) Update(ctx context.Context, req *api.UpdateRuleFilesRequest) (*api.UpdateRuleFilesResponse, error) {
	if req.GetFiles() == nil {
		return nil, errors.New("rule files required")
	}

	err := h.Service.Update(ctx, itemFromProto(req.GetFiles()))

	return &api.UpdateRuleFilesResponse{}, err
}

func (h Handler) Delete(ctx context.Context, req *api.DeleteRuleFilesRequest) (*api.DeleteRuleFilesResponse, error) {
	err := h.Service.Delete(ctx, req.GetTag())
	return &api.DeleteRuleFilesResponse{}, err
}

func (h Handler) List(ctx context.Context, _ *api.ListRuleFilesRequest) (*api.ListRuleFilesResponse, error) {
	items, err := h.Service.List(ctx)
	if err != nil {
		return nil, err
	}

	resp := &api.ListRuleFilesResponse{Files: make([]*api.RuleFiles, 0, len(items))}
	for _, item := range items {
		resp.Files = append(resp.Files, &api.RuleFiles{
			Tag:        item.Tag,
			GeoipUrl:   item.GeoIPURL,
			GeositeUrl: item.GeoSiteURL,
		})
	}

	return resp, nil
}

func itemFromProto(set *api.RuleFiles) model.Item {
	return model.Item{
		Tag:        set.GetTag(),
		GeoIPURL:   set.GetGeoipUrl(),
		GeoSiteURL: set.GetGeositeUrl(),
	}
}
