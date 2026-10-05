package service

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"slices"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"

	api "vpn/gen/node"
	nodemodel "vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
	"vpn/internal/control/routing/model"
	rulesetmodel "vpn/internal/control/ruleset/model"
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

type RuleSets interface {
	Get(context.Context, string) (rulesetmodel.Item, error)
	List(context.Context) ([]rulesetmodel.Item, error)
}

type Service struct {
	Repo  Repository
	Nodes Nodes
	Sets  RuleSets
	Mu    *sync.Mutex
}

func (s Service) Create(ctx context.Context, policy model.Policy) error {
	err := validatePolicy(policy)
	if err != nil {
		return err
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	err = s.checkRuleSet(ctx, policy.RuleSet)
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

	err = s.checkRuleSet(ctx, policy.RuleSet)
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

// Sync reconciles remote rule sets and mesh routes after mesh outbounds exist.
func (s Service) Sync(ctx context.Context) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	policies, err := s.Repo.List(ctx)
	if err != nil {
		return err
	}

	if len(policies) == 0 {
		return nil
	}

	sets, err := s.Sets.List(ctx)
	if err != nil {
		return err
	}

	known := make(map[string]bool, len(sets))
	for _, set := range sets {
		known[set.Tag] = true
	}

	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return err
	}

	slices.SortFunc(nodes, func(a, b nodemodel.Node) int { return strings.Compare(a.ID, b.ID) })

	var failures []error

	for _, source := range nodes {
		err = s.syncNode(ctx, source, nodes, policies, known)
		if err != nil {
			failures = append(failures, fmt.Errorf("routes on %s: %w", source.Name, err))
		}
	}

	return errors.Join(failures...)
}

func (s Service) syncNode(ctx context.Context, source nodemodel.Node, nodes []nodemodel.Node, policies []model.Policy, known map[string]bool) error {
	node, err := s.Nodes.Get(source.ID)
	if err != nil {
		return err
	}

	currentRules, err := node.RouteRules(ctx)
	if err != nil {
		return err
	}

	outbounds, err := node.Outbounds(ctx)
	if err != nil {
		return err
	}

	available := make(map[string]bool, len(outbounds))
	for _, outbound := range outbounds {
		available[outbound.GetTag()] = true
	}

	var desired []*api.RouteRule

	var dnsRoutes []*api.DNSRoute

	var failures []error

	// Rule-set matches must precede the catch-all route on each source node.
	for _, hasSet := range []bool{true, false} {
		for _, policy := range policies {
			if !policy.Active || (policy.RuleSet != "") != hasSet || !matches(policy.Source, source.Country) {
				continue
			}

			target, ok := selectNode(nodes, policy, source.ID)
			if !ok {
				failures = append(failures, fmt.Errorf("%s: no target node", policy.Tag))
				continue
			}

			outbound := "direct"
			if target.ID != source.ID {
				outbound = "mesh:" + target.ID
			}

			if !available[outbound] {
				failures = append(failures, fmt.Errorf("%s: outbound %s unavailable", policy.Tag, outbound))
				continue
			}

			rule := &api.RouteRule{Action: api.RouteAction_ROUTE_ACTION_ROUTE, Outbound: outbound}
			if hasSet {
				rule.RuleSets = []string{policy.RuleSet}
			} else if slices.ContainsFunc(desired, func(existing *api.RouteRule) bool { return proto.Equal(existing, rule) }) {
				continue
			}

			desired = append(desired, rule)
			dnsRoutes = append(dnsRoutes, &api.DNSRoute{
				RuleSet:  policy.RuleSet,
				Outbound: outbound,
			})
		}
	}

	var indices []int

	var existing []*api.RouteRule

	for i, rule := range currentRules {
		if managedRule(rule, known) {
			indices = append(indices, i)
			existing = append(existing, rule)
		}
	}

	if !equalRules(existing, desired) {
		for i := len(indices) - 1; i >= 0; i-- {
			err = node.RemoveRouteRule(ctx, uint32(indices[i]))
			if err != nil {
				return err
			}
		}

		for _, rule := range desired {
			var tag *string
			if len(rule.RuleSets) > 0 {
				tag = &rule.RuleSets[0]
			}

			err = node.AddRouteRule(ctx, nodemodel.RouteRule{
				RuleSet:  tag,
				Action:   nodemodel.RouteActionRoute,
				Outbound: rule.Outbound,
			})
			if err != nil {
				return err
			}
		}
	}

	managedSets := make([]string, 0, len(known))
	for tag := range known {
		managedSets = append(managedSets, tag)
	}

	err = node.SyncDNSRoutes(ctx, dnsRoutes, managedSets)
	if err != nil {
		failures = append(failures, err)
	}

	return errors.Join(failures...)
}

func managedRule(rule *api.RouteRule, known map[string]bool) bool {
	if rule.GetAction() != api.RouteAction_ROUTE_ACTION_ROUTE || len(rule.GetUsers()) != 0 {
		return false
	}

	for _, tag := range rule.GetRuleSets() {
		if known[tag] {
			return true
		}
	}

	return len(rule.GetRuleSets()) == 0 && strings.HasPrefix(rule.GetOutbound(), "mesh:")
}

func equalRules(a, b []*api.RouteRule) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if !proto.Equal(a[i], b[i]) {
			return false
		}
	}

	return true
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

func (s Service) checkRuleSet(ctx context.Context, tag string) error {
	if tag == "" {
		return nil
	}

	set, err := s.Sets.Get(ctx, tag)
	if err != nil {
		return err
	}

	if !set.Active {
		return errors.New("rule set is deleted")
	}

	return nil
}

func validatePolicy(p model.Policy) error {
	if p.Tag == "" || len(p.Tag) > 120 || strings.ContainsAny(p.Tag, " \t\n") {
		return errors.New("route tag required without whitespace (max 120 characters)")
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
