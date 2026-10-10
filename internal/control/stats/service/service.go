package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/xtls/xray-core/app/stats/command"

	nodemodel "vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
	"vpn/internal/control/stats/model"
)

type Nodes interface {
	List(context.Context) ([]nodemodel.Node, error)
	Get(string) (*provider.Node, error)
}

type Repository interface {
	Add(context.Context, map[string]model.Usage) error
	Device(context.Context, string) (model.Usage, error)
	User(context.Context, string, bool) (model.Usage, error)
}

type Service struct {
	Nodes Nodes
	Repo  Repository
	mu    sync.Mutex
	// Counters already drained from Xray but not yet committed to Postgres.
	pending map[string]map[string]model.Usage
}

func (s *Service) Sync(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return err
	}

	if s.pending == nil {
		s.pending = make(map[string]map[string]model.Usage)
	}

	var failures []error

	for _, entry := range nodes {
		if s.pending[entry.ID] == nil {
			s.pending[entry.ID] = make(map[string]model.Usage)
		}

		node, getErr := s.Nodes.Get(entry.ID)
		if getErr != nil {
			failures = append(failures, fmt.Errorf("stats %s: %w", entry.Name, getErr))
			continue
		}

		counters, getErr := node.XrayTraffic(ctx)
		if getErr != nil {
			failures = append(failures, fmt.Errorf("stats %s: %w", entry.Name, getErr))
		} else {
			for _, counter := range counters {
				id, direction, ok := parseCounter(counter)
				if !ok {
					continue
				}

				usage := s.pending[entry.ID][id]
				if direction == "uplink" {
					usage.Uploaded += counter.Value
				} else {
					usage.Downloaded += counter.Value
				}

				s.pending[entry.ID][id] = usage
			}
		}

		if len(s.pending[entry.ID]) == 0 {
			continue
		}

		saveErr := s.Repo.Add(ctx, s.pending[entry.ID])
		if saveErr != nil {
			failures = append(failures, fmt.Errorf("save stats %s: %w", entry.Name, saveErr))
			continue
		}

		delete(s.pending, entry.ID)
	}

	return errors.Join(failures...)
}

func (s *Service) Device(ctx context.Context, id string) (model.Usage, error) {
	return s.Repo.Device(ctx, id)
}

func (s *Service) User(ctx context.Context, id string, defaultNetwork bool) (model.Usage, error) {
	return s.Repo.User(ctx, id, defaultNetwork)
}

func parseCounter(stat *command.Stat) (string, string, bool) {
	if stat == nil || stat.Value <= 0 {
		return "", "", false
	}

	rest, ok := strings.CutPrefix(stat.Name, "user>>>")
	if !ok {
		return "", "", false
	}

	id, direction, ok := strings.Cut(rest, ">>>traffic>>>")
	if !ok || id == "" || direction != "uplink" && direction != "downlink" {
		return "", "", false
	}

	return id, direction, true
}
