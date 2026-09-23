package service

import (
	"fmt"

	"vpn/internal/node/sing-box/config"
	"vpn/internal/node/sing-box/model/inbound"
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
	return s.cfg.AddRuleSet(config.RuleSet{
		Type:   config.RuleSetTypeRemote,
		Tag:    name,
		Format: config.RuleSetFormatBinary,
		URL:    fmt.Sprintf("https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-%s.srs", name),
	})
}

func (s *SingBoxService) GetRuleSets() []string {
	ruleSets := s.cfg.GetRuleSet()

	result := make([]string, 0, len(ruleSets))

	for _, ruleSet := range ruleSets {
		result = append(result, ruleSet.Tag)
	}

	return result
}

func (s *SingBoxService) AddRoute(user *string, rule *string, action config.RouteAction) error {
	return s.cfg.AddRoute(user, rule, action)
}

func (s *SingBoxService) GetInbounds() []inbound.Inbound {
	return s.cfg.GetInbouns()
}

func (s *SingBoxService) AddInbound(tag string, inboundType inbound.InboundType) error {
	switch inboundType {
	case inbound.InboundTypeHysteria2:
		return s.cfg.AddInbound(inbound.NewHysteria2(tag, 1488))
	}
	return fmt.Errorf("Unknowm type")
}

func (s *SingBoxService) AddUser(user string, tag string) error {
	inbounds := s.cfg.GetInbouns()

	for i := range inbounds {
		if inbounds[i].Tag == tag {
			err := inbounds[i].AddUser(user)
			if err != nil {
				return err
			}

			return s.cfg.Apply()
		}
	}

	return fmt.Errorf("failed to find inbound %q", tag)
}
