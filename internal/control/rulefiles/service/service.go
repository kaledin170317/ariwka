package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"

	"vpn/internal/control/rulefiles/model"
	"vpn/internal/control/rulefiles/provider"
)

type Repository interface {
	Create(context.Context, model.Item) error
	Update(context.Context, model.Item) error
	Get(context.Context, string) (model.Item, error)
	Delete(context.Context, string) error
	List(context.Context) ([]model.Item, error)
	Used(context.Context, string) (bool, error)
}

type Service struct {
	Repo Repository
	Mu   *sync.Mutex
}

func (s Service) Create(ctx context.Context, item model.Item) error {
	err := validate(item)
	if err != nil {
		return err
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	return s.Repo.Create(ctx, item)
}

func (s Service) Update(ctx context.Context, item model.Item) error {
	err := validate(item)
	if err != nil {
		return err
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	return s.Repo.Update(ctx, item)
}

func (s Service) Delete(ctx context.Context, tag string) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	used, err := s.Repo.Used(ctx, tag)
	if err != nil {
		return err
	}

	if used {
		return errors.New("rule files are used by a route policy")
	}

	return s.Repo.Delete(ctx, tag)
}

func (s Service) List(ctx context.Context) ([]model.Item, error) {
	items, err := s.Repo.List(ctx)
	if err != nil {
		return nil, err
	}

	return slices.DeleteFunc(items, func(item model.Item) bool { return !item.Active }), nil
}

func validate(item model.Item) error {
	if item.Tag == "" || len(item.Tag) > 120 || strings.ContainsAny(item.Tag, " \t\n") {
		return errors.New("rule files tag required without whitespace (max 120 characters)")
	}

	if item.GeoIPURL == "" && item.GeoSiteURL == "" ||
		!provider.ValidURL(item.GeoIPURL) || !provider.ValidURL(item.GeoSiteURL) {
		return errors.New("xray rule files need a geoip.dat or geosite.dat HTTPS URL")
	}

	return nil
}
