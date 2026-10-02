package service

import (
	"fmt"
	"sync"

	"vpn/internal/node/sing-box/config"
	"vpn/internal/node/sing-box/model/inbound"
	portcheck "vpn/pkg/port"
)

type SingBoxService struct {
	mu  sync.Mutex
	cfg config.Config
}

func NewSingBoxService() SingBoxService {
	return SingBoxService{
		cfg: config.New(),
	}
}

func (s *SingBoxService) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cfg.Sync()
}

func (s *SingBoxService) AddRemoteRule(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cfg.AddRuleSet(name, fmt.Sprintf("https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-%s.srs", name))
}

func (s *SingBoxService) GetRuleSets() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cfg.GetRuleSets()
}

func (s *SingBoxService) AddRoute(user *string, rule *string, action config.RouteAction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cfg.AddRoute(user, rule, action)
}

func (s *SingBoxService) GetInbounds() ([]inbound.Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cfg.GetInbounds()
}

func (s *SingBoxService) AddHysteria2Inbound(tag string, port int32) (*inbound.Hysteria2, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := portcheck.Check(port)
	if err != nil {
		return nil, err
	}

	return s.cfg.AddHysteria2Inbound(tag, port)
}

func (s *SingBoxService) AddVlessInbound(tag string, port int32, serverName string, t inbound.VlessTransportOptions, privateKey, shortID, serviceName string) (*inbound.Vless, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := portcheck.Check(port)
	if err != nil {
		return nil, err
	}

	return s.cfg.AddVlessInbound(tag, port, serverName, t, privateKey, shortID, serviceName)
}

func (s *SingBoxService) RemoveInbound(tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cfg.RemoveInbound(tag)
}

func (s *SingBoxService) AddUser(inboundTag string, user config.User) (inbound.Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cfg.AddUser(inboundTag, user)
}

func (s *SingBoxService) RemoveUser(user, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cfg.RemoveUser(user, tag)
}

func (s *SingBoxService) ModifyInbound(tag string, options []config.InboundOption) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cfg.ModifyInbound(tag, options)
}

func (s *SingBoxService) ModifyUser(inboundTag, tag string, options []config.UserOption) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cfg.ModifyUser(inboundTag, tag, options)
}
