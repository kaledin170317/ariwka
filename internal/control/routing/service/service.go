package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash/fnv"
	"slices"
	"strings"
	"sync"

	"github.com/xtls/xray-core/app/router"
	"google.golang.org/protobuf/proto"

	nodemodel "vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
	"vpn/internal/control/routing/model"
	rulefilesmodel "vpn/internal/control/rulefiles/model"
	rulefilesprovider "vpn/internal/control/rulefiles/provider"
)

type Repository interface {
	Create(context.Context, model.Policy) error
	Update(context.Context, model.Policy) error
	Delete(context.Context, string) error
	List(context.Context) ([]model.Policy, error)
}

type Nodes interface {
	List(context.Context) ([]nodemodel.Node, error)
	Get(string) (*provider.Node, error)
}

type RuleFiles interface {
	Get(context.Context, string) (rulefilesmodel.Item, error)
	List(context.Context) ([]rulefilesmodel.Item, error)
}

type Inbounds interface {
	ListInbounds(context.Context, string) ([]nodemodel.InboundConfig, error)
}

type fileCode struct{ file, code string }

type Service struct {
	Repo     Repository
	Nodes    Nodes
	Sets     RuleFiles
	Inbounds Inbounds
	Data     *rulefilesprovider.Provider
	Mu       *sync.Mutex
	applied  map[string][32]byte
}

func (s Service) Create(ctx context.Context, policy model.Policy) error {
	err := validatePolicy(policy)
	if err != nil {
		return err
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	err = s.checkRuleFile(ctx, policy.RuleFile)
	if err != nil {
		return err
	}

	return s.Repo.Create(ctx, policy)
}

func (s Service) Update(ctx context.Context, policy model.Policy) error {
	err := validatePolicy(policy)
	if err != nil {
		return err
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	err = s.checkRuleFile(ctx, policy.RuleFile)
	if err != nil {
		return err
	}

	return s.Repo.Update(ctx, policy)
}

func (s Service) Delete(ctx context.Context, tag string) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	return s.Repo.Delete(ctx, tag)
}

func (s Service) List(ctx context.Context) ([]model.Policy, error) {
	policies, err := s.Repo.List(ctx)
	if err != nil {
		return nil, err
	}

	return slices.DeleteFunc(policies, func(p model.Policy) bool { return !p.Active }), nil
}

// Sync replaces Xray's routing rules with the rules derived from active policies.
func (s *Service) Sync(ctx context.Context) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	policies, err := s.Repo.List(ctx)
	if err != nil {
		return err
	}

	items, err := s.Sets.List(ctx)
	if err != nil {
		return err
	}

	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return err
	}

	slices.SortFunc(nodes, func(a, b nodemodel.Node) int { return strings.Compare(a.ID, b.ID) })

	sets := make(map[fileCode]rulefilesprovider.Data)

	for _, policy := range policies {
		if !policy.Active || !policy.ApplyToNodes || policy.RuleFile == "" {
			continue
		}

		key := fileCode{policy.RuleFile, policy.RuleCode}
		if _, ok := sets[key]; ok {
			continue
		}

		found := false

		for _, item := range items {
			if item.Tag != policy.RuleFile || !item.Active {
				continue
			}

			data, loadErr := s.Data.Load(ctx, item, policy.RuleCode)
			if loadErr != nil {
				return fmt.Errorf("rule files %s: %w", item.Tag, loadErr)
			}

			sets[key] = data
			found = true

			break
		}

		if !found {
			return fmt.Errorf("rule files %s not found", policy.RuleFile)
		}
	}

	if s.applied == nil {
		s.applied = make(map[string][32]byte)
	}

	var failures []error

	for _, source := range nodes {
		err = s.syncNode(ctx, source, nodes, policies, sets)
		if err != nil {
			failures = append(failures, fmt.Errorf("routes on %s: %w", source.Name, err))
		}
	}

	return errors.Join(failures...)
}

func (s *Service) syncNode(ctx context.Context, source nodemodel.Node, nodes []nodemodel.Node, policies []model.Policy, sets map[fileCode]rulefilesprovider.Data) error {
	node, err := s.Nodes.Get(source.ID)
	if err != nil {
		return err
	}

	configs, err := s.Inbounds.ListInbounds(ctx, source.ID)
	if err != nil {
		return err
	}

	var userTags []string

	for _, config := range configs {
		if config.Network == nodemodel.InboundNetworkUsers {
			userTags = append(userTags, config.Tag)
		}
	}

	outbounds, err := node.XrayOutbounds(ctx)
	if err != nil {
		return err
	}

	available := make(map[string]bool, len(outbounds))
	for _, outbound := range outbounds {
		available[outbound.Tag] = true
	}

	desired := &router.Config{}
	// Mesh arrivals and the default subscription must always exit from this node.
	desired.Rule = append(desired.Rule,
		&router.RoutingRule{RuleTag: "vpn:mesh", TargetTag: &router.RoutingRule_Tag{Tag: "direct"}, InboundTag: []string{"mesh:" + source.ID}},
		&router.RoutingRule{RuleTag: "vpn:default", TargetTag: &router.RoutingRule_Tag{Tag: "direct"}, InboundTag: []string{"default:" + source.ID}},
	)

	// Set matches precede the catch-all routes, just as in the old policy engine.
	for _, hasSet := range []bool{true, false} {
		for _, policy := range policies {
			if !policy.Active || !policy.ApplyToNodes || (policy.RuleFile != "") != hasSet || !matches(policy.Source, source.Country) {
				continue
			}

			target, ok := selectNode(nodes, policy, source.ID)
			if !ok {
				return fmt.Errorf("%s: no target node", policy.Tag)
			}

			outbound := "direct"
			if target.ID != source.ID {
				outbound = "mesh:" + target.ID
			}

			if !available[outbound] {
				return fmt.Errorf("%s: outbound %s unavailable", policy.Tag, outbound)
			}

			if hasSet {
				data := sets[fileCode{policy.RuleFile, policy.RuleCode}]

				tag := fmt.Sprintf("vpn:policy:%d:%s", len(policy.Tag), policy.Tag)
				if len(data.Domains) > 0 {
					desired.Rule = append(desired.Rule, &router.RoutingRule{RuleTag: tag + ":domain", TargetTag: &router.RoutingRule_Tag{Tag: outbound}, Domain: data.Domains})
				}

				if len(data.IPs) > 0 {
					desired.Rule = append(desired.Rule, &router.RoutingRule{RuleTag: tag + ":ip", TargetTag: &router.RoutingRule_Tag{Tag: outbound}, Ip: data.IPs})
				}
			} else {
				if len(userTags) > 0 {
					tag := fmt.Sprintf("vpn:policy:%d:%s", len(policy.Tag), policy.Tag)
					desired.Rule = append(desired.Rule, &router.RoutingRule{RuleTag: tag, TargetTag: &router.RoutingRule_Tag{Tag: outbound}, InboundTag: userTags})
				}
			}
		}
	}

	current, err := node.XrayRules(ctx)
	if err != nil {
		return err
	}

	encoded, err := proto.Marshal(desired)
	if err != nil {
		return err
	}

	sum := sha256.Sum256(encoded)
	if sum == s.applied[source.ID] && len(current) == len(desired.Rule) {
		matched := true

		for i, rule := range current {
			if rule.RuleTag != desired.Rule[i].RuleTag || rule.Tag != desired.Rule[i].GetTag() {
				matched = false
				break
			}
		}

		if matched {
			return nil
		}
	}

	err = node.XraySetRules(ctx, desired)
	if err != nil {
		return err
	}

	s.applied[source.ID] = sum

	return nil
}

func selectNode(nodes []nodemodel.Node, policy model.Policy, sourceID string) (nodemodel.Node, bool) {
	var candidates []nodemodel.Node

	for _, node := range nodes {
		if matches(policy.Target, node.Country) {
			candidates = append(candidates, node)
		}
	}

	if len(candidates) == 0 {
		return nodemodel.Node{}, false
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(policy.Tag + sourceID))

	return candidates[int(h.Sum32())%len(candidates)], true
}

func matches(f model.Filter, country string) bool {
	return (f.Country == "" || strings.EqualFold(f.Country, country)) &&
		(f.NotCountry == "" || !strings.EqualFold(f.NotCountry, country))
}

func (s Service) checkRuleFile(ctx context.Context, tag string) error {
	if tag == "" {
		return nil
	}

	set, err := s.Sets.Get(ctx, tag)
	if err != nil {
		return err
	}

	if !set.Active {
		return errors.New("rule files are deleted")
	}

	return nil
}

func validatePolicy(p model.Policy) error {
	if p.Tag == "" || len(p.Tag) > 120 || strings.ContainsAny(p.Tag, " \t\n") {
		return errors.New("route tag required without whitespace (max 120 characters)")
	}
	if p.ApplyToClients && p.RuleFile == "" {
		return errors.New("client policies require a rule file; the VPN fallback is configured separately")
	}

	if (p.RuleFile == "") != (p.RuleCode == "") || len(p.RuleCode) > 120 || strings.ContainsAny(p.RuleCode, " \t\n") {
		return errors.New("rule_file and rule_code must both be set or both be empty (max 120 characters)")
	}

	for _, f := range []model.Filter{p.Source, p.Target} {
		if f.Country != "" && f.NotCountry != "" {
			return errors.New("country and not_country cannot be combined")
		}

		for _, code := range []string{f.Country, f.NotCountry} {
			if code != "" && (len(code) != 2 || strings.ToUpper(code) != code || code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z') {
				return errors.New("country filter must be a two-letter uppercase code")
			}
		}
	}

	return nil
}
