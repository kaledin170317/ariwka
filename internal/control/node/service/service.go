package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/google/uuid"

	controlapi "vpn/gen/control"
	api "vpn/gen/node"
	"vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
	"vpn/pkg/country"
)

type Repository interface {
	Create(context.Context, model.Node) error
	Update(context.Context, model.Node) error
	Delete(context.Context, string) error
	List(context.Context) ([]model.Node, error)
	CreateInbound(context.Context, model.InboundConfig) error
	ListInbounds(context.Context, string) ([]model.InboundConfig, error)
	DeleteInbound(context.Context, string, string) error
}

type Service struct {
	Repo     Repository
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

func (s Service) Inbounds(ctx context.Context, nodeID string) ([]model.Inbound, error) {
	node, err := s.Get(nodeID)
	if err != nil {
		return nil, err
	}

	inbounds, err := node.Inbounds(ctx)
	if err != nil {
		return nil, err
	}

	configs, err := s.Repo.ListInbounds(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	for i := range inbounds {
		for _, config := range configs {
			if config.Tag == inbounds[i].Tag {
				inbounds[i].Network = config.Network
				break
			}
		}
	}

	return inbounds, nil
}

func (s Service) RuleSets(ctx context.Context, nodeID string) ([]*api.RuleSet, error) {
	node, err := s.Get(nodeID)
	if err != nil {
		return nil, err
	}

	return node.RuleSets(ctx)
}

func (s Service) AddRuleSet(ctx context.Context, nodeID, name, url string) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	return node.AddRuleSet(ctx, name, url)
}

func (s Service) RemoveRuleSet(ctx context.Context, nodeID, name string) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	return node.RemoveRuleSet(ctx, name)
}

func (s Service) ModifyRuleSet(ctx context.Context, nodeID, name, url string) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	return node.ModifyRuleSet(ctx, name, url)
}

func (s Service) RouteRules(ctx context.Context, nodeID string) ([]*api.RouteRule, error) {
	node, err := s.Get(nodeID)
	if err != nil {
		return nil, err
	}

	return node.RouteRules(ctx)
}

func (s Service) RemoveRouteRule(ctx context.Context, nodeID string, index uint32) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	return node.RemoveRouteRule(ctx, index)
}

func (s Service) AddRouteRule(ctx context.Context, nodeID string, rule model.RouteRule) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	return node.AddRouteRule(ctx, rule)
}

func (s Service) Outbounds(ctx context.Context, nodeID string) ([]*api.Outbound, error) {
	node, err := s.Get(nodeID)
	if err != nil {
		return nil, err
	}

	return node.Outbounds(ctx)
}

func (s Service) AddOutbound(ctx context.Context, nodeID string, outbound *api.Outbound) error {
	if outbound == nil || strings.HasPrefix(outbound.GetTag(), meshPrefix) {
		return errors.New("reserved outbound tag")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	return node.AddOutbound(ctx, outbound)
}

func (s Service) RemoveOutbound(ctx context.Context, nodeID, tag string) error {
	if strings.HasPrefix(tag, meshPrefix) {
		return errors.New("reserved outbound tag")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	return node.RemoveOutbound(ctx, tag)
}

func (s Service) ModifyOutbound(ctx context.Context, nodeID, tag string, options []*api.OutboundOption) error {
	if strings.HasPrefix(tag, meshPrefix) {
		return errors.New("reserved outbound tag")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	return node.ModifyOutbound(ctx, tag, options)
}

func (s Service) AddHysteria2Inbound(ctx context.Context, nodeID string, port int32) (model.Inbound, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	node, err := s.inboundNode(nodeID, port)
	if err != nil {
		return model.Inbound{}, err
	}

	config := model.InboundConfig{
		Tag:     uuid.NewString(),
		NodeID:  nodeID,
		Type:    model.InboundTypeHysteria2,
		Port:    port,
		Network: model.InboundNetworkUsers,
	}

	err = s.Repo.CreateInbound(ctx, config)
	if err != nil {
		return model.Inbound{}, err
	}

	return node.AddHysteria2Inbound(ctx, config.Tag, port)
}

func (s Service) AddVlessInbound(ctx context.Context, nodeID string, port int32, serverName string, transport model.VlessTransportOptions) (model.Inbound, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	node, err := s.inboundNode(nodeID, port)
	if err != nil {
		return model.Inbound{}, err
	}

	if serverName == "" || !transport.Type.IsValid() {
		return model.Inbound{}, errors.New("server name and valid transport required")
	}

	key := make([]byte, 32)

	_, err = rand.Read(key)
	if err != nil {
		return model.Inbound{}, err
	}

	short := make([]byte, 8)

	_, err = rand.Read(short)
	if err != nil {
		return model.Inbound{}, err
	}

	config := model.InboundConfig{
		Tag:        uuid.NewString(),
		NodeID:     nodeID,
		Type:       model.InboundTypeVless,
		Port:       port,
		ServerName: serverName,
		Transport:  transport,
		PrivateKey: base64.RawURLEncoding.EncodeToString(key),
		ShortID:    hex.EncodeToString(short),
		Network:    model.InboundNetworkUsers,
	}
	if transport.Type == model.VlessTransportTypeGrpc {
		config.ServiceName = rand.Text()
	}

	err = s.Repo.CreateInbound(ctx, config)
	if err != nil {
		return model.Inbound{}, err
	}

	return node.AddVlessInbound(ctx, config)
}

func (s Service) DeleteInbound(ctx context.Context, nodeID, tag string) error {
	if strings.HasPrefix(tag, meshPrefix) {
		return errors.New("reserved inbound tag")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	configs, err := s.Repo.ListInbounds(ctx, nodeID)
	if err != nil {
		return err
	}

	found := false

	for _, config := range configs {
		if config.Tag == tag {
			found = true
			break
		}
	}

	if !found {
		return sql.ErrNoRows
	}

	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	inbounds, err := node.Inbounds(ctx)
	if err != nil {
		return err
	}

	for _, inbound := range inbounds {
		if inbound.Tag == tag {
			err = node.RemoveInbound(ctx, tag)
			if err != nil {
				return err
			}

			break
		}
	}

	return s.Repo.DeleteInbound(ctx, nodeID, tag)
}

func (s Service) SyncInbounds(ctx context.Context) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	nodes, err := s.Repo.List(ctx)
	if err != nil {
		return err
	}

	var failures []error

	for _, n := range nodes {
		err = s.syncNode(ctx, n.ID)
		if err != nil {
			failures = append(failures, fmt.Errorf("node %s: %w", n.Name, err))
		}
	}

	return errors.Join(failures...)
}

func (s Service) inboundNode(nodeID string, port int32) (*provider.Node, error) {
	if port < 1 || port > 65535 {
		return nil, errors.New("invalid inbound port")
	}

	return s.Get(nodeID)
}

func (s Service) syncNode(ctx context.Context, nodeID string) error {
	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	desired, err := s.Repo.ListInbounds(ctx, nodeID)
	if err != nil {
		return err
	}

	current, err := node.Inbounds(ctx)
	if err != nil {
		return err
	}

	desiredTags := make(map[string]bool, len(desired))
	for _, config := range desired {
		desiredTags[config.Tag] = true
	}

	tags := make(map[string]model.Inbound, len(current))
	for _, inbound := range current {
		if desiredTags[inbound.Tag] {
			tags[inbound.Tag] = inbound
		} else {
			err = node.RemoveInbound(ctx, inbound.Tag)
			if err != nil {
				return fmt.Errorf("remove extra inbound %s: %w", inbound.Tag, err)
			}
		}
	}

	var failures []error

	for _, config := range desired {
		if existing, ok := tags[config.Tag]; ok {
			err = node.ModifyInbound(ctx, config, existing)
		} else {
			switch config.Type {
			case model.InboundTypeHysteria2:
				_, err = node.AddHysteria2Inbound(ctx, config.Tag, config.Port)
			case model.InboundTypeVless:
				if config.PrivateKey != "" {
					_, err = node.AddVlessInbound(ctx, config)
				} else {
					err = errors.New("no private key stored")
				}
			default:
				err = fmt.Errorf("unsupported inbound type %d", config.Type)
			}
		}

		if err != nil {
			failures = append(failures, fmt.Errorf("inbound %s: %w", config.Tag, err))
		}
	}

	return errors.Join(failures...)
}

func nodeHost(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		return "", errors.New("address must be host:port")
	}

	return host, nil
}
