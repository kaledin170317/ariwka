package service

import (
	"fmt"

	"vpn/internal/node/sing-box/config"
	"vpn/internal/node/sing-box/model/inbound"
	portcheck "vpn/pkg/port"
)

type SingBoxService struct {
	cfg config.Config
}

func NewSingBoxService() SingBoxService {
	return SingBoxService{
		cfg: config.New(),
	}
}

func (s *SingBoxService) Sync() error {
	return s.cfg.Sync()
}

func (s *SingBoxService) AddRemoteRule(name string) error {
	return s.cfg.AddRuleSet(name, fmt.Sprintf("https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-%s.srs", name))
}

func (s *SingBoxService) GetRuleSets() []string {
	return s.cfg.GetRuleSets()
}

func (s *SingBoxService) AddRoute(user *string, rule *string, action config.RouteAction) error {
	return s.cfg.AddRoute(user, rule, action)
}

func (s *SingBoxService) GetInbounds() ([]inbound.Connection, error) {
	return s.cfg.GetInbounds()
}

func (s *SingBoxService) AddHysteria2Inbound(port int32) (*inbound.Hysteria2, error) {
	err := portcheck.Check(port)
	if err != nil {
		return nil, err
	}

	return s.cfg.AddHysteria2Inbound(port)
}

func (s *SingBoxService) AddVlessInbound(port int32, serverName string, t inbound.VlessTransportOptions) (*inbound.Vless, error) {
	err := portcheck.Check(port)
	if err != nil {
		return nil, err
	}

	return s.cfg.AddVlessInbound(port, serverName, t)
}

func (s *SingBoxService) RemoveInbound(tag string) error {
	return s.cfg.RemoveInbound(tag)
}

func (s *SingBoxService) AddUser(inboundTag string) (inbound.Connection, string, string, error) {
	return s.cfg.AddUser(inboundTag)
}

func (s *SingBoxService) RemoveUser(user, tag string) error {
	return s.cfg.RemoveUser(user, tag)
}
