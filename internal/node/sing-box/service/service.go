package service

import (
	"errors"
	"fmt"

	common "vpn/gen/vpn/common/v1"
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

func (s *SingBoxService) GetInbounds() ([]inbound.Inbound, error) {
	inbounds := s.cfg.GetInbouns()
	for i := range inbounds {
		if inbounds[i].Type == inbound.InboundTypeHysteria2 {
			if err := inbounds[i].LoadCertificatePin(); err != nil {
				return nil, err
			}
		}
	}
	return inbounds, nil
}

func (s *SingBoxService) AddInbound(tag string, kind common.InboundType, port int32) error {
	if err := portcheck.CheckUDP(port); err != nil {
		return err
	}

	var (
		connection inbound.Inbound
		err        error
	)

	switch kind {
	case common.InboundType_INBOUND_TYPE_HYSTERIA2:
		connection, err = inbound.NewHysteria2(tag, port)
	default:
		return errors.New("unsupported inbound type")
	}

	if err != nil {
		return err
	}
	return s.cfg.AddInbound(connection)
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

func (s *SingBoxService) RemoveUser(user, tag string) error {
	inbounds := s.cfg.GetInbouns()
	for i := range inbounds {
		if inbounds[i].Tag == tag {
			err := inbounds[i].RemoveUser(user)
			if err != nil {
				return err
			}
			return s.cfg.Apply()
		}
	}
	return fmt.Errorf("failed to find inbound %q", tag)
}
