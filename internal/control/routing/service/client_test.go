package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xtls/xray-core/common/geodata"

	nodemodel "vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
	"vpn/internal/control/routing/model"
)

type testPolicies []model.Policy

func (p testPolicies) List(context.Context) ([]model.Policy, error) { return p, nil }
func (testPolicies) Create(context.Context, model.Policy) error     { return nil }
func (testPolicies) Update(context.Context, model.Policy) error     { return nil }
func (testPolicies) Delete(context.Context, string) error           { return nil }

type testNodes []nodemodel.Node

func (n testNodes) List(context.Context) ([]nodemodel.Node, error) { return n, nil }
func (testNodes) Get(string) (*provider.Node, error)               { return nil, nil }

func TestClientRulesSkipsNodeOnlyPolicies(t *testing.T) {
	s := Service{Repo: testPolicies{{Tag: "ru-direct-full", RuleFile: "large", Active: true, ApplyToNodes: true}},
		Nodes: testNodes{{Country: "FI"}, {Country: "RU"}}}
	rules, err := s.ClientRules(context.Background(), "RU")
	if err != nil || len(rules.Rules) != 0 || rules.GeoIPURL != "" || rules.GeoSiteURL != "" {
		t.Fatalf("node-only list must not load on client: %v %v", rules, err)
	}
}

func TestClientPolicyRequiresRuleFile(t *testing.T) {
	if err := validatePolicy(model.Policy{Tag: "client-catchall", ApplyToClients: true}); err == nil {
		t.Fatal("client-only catch-all would be silently ignored")
	}
}

func TestClientGeoRulesRemainCompact(t *testing.T) {
	domains := make([]*geodata.DomainRule, 2000)
	ips := make([]*geodata.IPRule, 1000)
	rules := appendGeoRules(nil, "ru-standard", true, domains, ips)
	data, err := json.Marshal(rules)
	if err != nil || len(data) > 250 || len(rules) != 2 {
		t.Fatalf("geo rule list expanded: %s (%v)", data, err)
	}
	if string(data) != `[{"domain":["geosite:ru-standard"],"outboundTag":"direct","type":"field"},{"ip":["geoip:ru-standard"],"outboundTag":"direct","type":"field"}]` {
		t.Fatalf("wrong geo categories: %s", data)
	}
}

func TestClientPolicyTarget(t *testing.T) {
	nodes := []nodemodel.Node{{Country: "FI"}, {Country: "DE"}, {Country: "RU"}, {Country: "RU"}}
	for _, tc := range []struct {
		name        string
		policy      model.Policy
		use, direct bool
	}{
		{"local for all foreign sources", model.Policy{Source: model.Filter{NotCountry: "RU"}, Target: model.Filter{Country: "RU"}}, true, true},
		{"ambiguous source", model.Policy{Source: model.Filter{Country: "FI"}, Target: model.Filter{Country: "RU"}}, true, false},
		{"foreign target", model.Policy{Target: model.Filter{Country: "FI"}}, true, false},
		{"ambiguous target", model.Policy{}, true, false},
		{"only local source", model.Policy{Source: model.Filter{Country: "RU"}, Target: model.Filter{Country: "RU"}}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			use, direct, err := clientPolicyTarget(tc.policy, nodes, "RU")
			if err != nil || use != tc.use || direct != tc.direct {
				t.Fatalf("use=%v direct=%v err=%v", use, direct, err)
			}
		})
	}
}
