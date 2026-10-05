package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	nodemodel "vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
	"vpn/internal/control/ruleset/model"
)

type Repository interface {
	Create(context.Context, model.Item) error
	Update(context.Context, model.Item) error
	Get(context.Context, string) (model.Item, error)
	Delete(context.Context, string) error
	List(context.Context) ([]model.Item, error)
	Used(context.Context, string) (bool, error)
}

type Nodes interface {
	List(context.Context) ([]nodemodel.Node, error)
	Get(string) (*provider.Node, error)
}

type Service struct {
	Repo  Repository
	Nodes Nodes
	Mu    *sync.Mutex
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
		return errors.New("rule set is used by a route policy")
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

// Sync ensures all nodes have the same remote rule sets.
func (s Service) Sync(ctx context.Context) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	items, err := s.Repo.List(ctx)
	if err != nil {
		return err
	}

	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return err
	}

	var failures []error

	for _, entry := range nodes {
		node, getErr := s.Nodes.Get(entry.ID)
		if getErr != nil {
			failures = append(failures, getErr)
			continue
		}

		current, listErr := node.RuleSets(ctx)
		if listErr != nil {
			failures = append(failures, fmt.Errorf("rule sets on %s: %w", entry.Name, listErr))
			continue
		}

		for _, item := range items {
			found := false

			for _, existing := range current {
				if existing.GetName() != item.Tag {
					continue
				}

				found = true

				if item.Active && (existing.GetUrl() != item.URL || existing.GetFormat() != item.Format.String() || existing.GetUpdateIntervalSeconds() != model.UpdateIntervalSeconds) {
					getErr = node.ModifyRuleSetConfig(ctx, item.Tag, item.URL, item.Format.String(), model.UpdateIntervalSeconds)
				}

				break
			}

			if item.Active && !found {
				getErr = node.AddRuleSetConfig(ctx, item.Tag, item.URL, item.Format.String(), model.UpdateIntervalSeconds)
			} else if !item.Active && found {
				getErr = node.RemoveRuleSet(ctx, item.Tag)
			}

			if getErr != nil {
				failures = append(failures, fmt.Errorf("rule set %s on %s: %w", item.Tag, entry.Name, getErr))
				getErr = nil
			}
		}

		for _, existing := range current {
			known := false

			for _, item := range items {
				if item.Tag == existing.GetName() {
					known = true
					break
				}
			}

			if !known {
				removeErr := node.RemoveRuleSet(ctx, existing.GetName())
				if removeErr != nil {
					failures = append(failures, fmt.Errorf("extra rule set %s on %s: %w", existing.GetName(), entry.Name, removeErr))
				}
			}
		}
	}

	return errors.Join(failures...)
}

func validate(item model.Item) error {
	if item.Tag == "" || len(item.Tag) > 120 || strings.ContainsAny(item.Tag, " \t\n") {
		return errors.New("rule set tag required without whitespace (max 120 characters)")
	}

	if !strings.HasPrefix(item.URL, "https://") || !item.Format.IsValid() {
		return errors.New("rule set HTTPS URL and binary or source format required")
	}

	return nil
}
