package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/xtls/xray-core/common/geodata"

	nodemodel "vpn/internal/control/node/model"
	"vpn/internal/control/routing/model"
	rulefilesprovider "vpn/internal/control/rulefiles/provider"
)

// ClientRules translates the active, ordered rule-file policies for a device
// region into Xray rules. Only policies valid for EVERY foreign entry node
// and targeting ONLY local nodes may bypass VPN. Ambiguity stays proxied.
func (s Service) ClientRules(ctx context.Context, region string) (model.ClientRouting, error) {
	policies, err := s.Repo.List(ctx)
	if err != nil {
		return model.ClientRouting{}, err
	}
	nodes, err := s.Nodes.List(ctx)
	if err != nil {
		return model.ClientRouting{}, err
	}
	var result model.ClientRouting
	for _, policy := range policies {
		if !policy.Active || !policy.ApplyToClients || policy.RuleFile == "" {
			continue
		}
		use, local, err := clientPolicyTarget(policy, nodes, region)
		if err != nil {
			return model.ClientRouting{}, err
		}
		if !use {
			continue
		}
		item, err := s.Sets.Get(ctx, policy.RuleFile)
		if err != nil {
			return model.ClientRouting{}, err
		}
		if !item.Active {
			return model.ClientRouting{}, fmt.Errorf("rule file %s is inactive", policy.RuleFile)
		}
		if item.GeoIPURL == "" || item.GeoSiteURL == "" ||
			!rulefilesprovider.ValidURL(item.GeoIPURL) || !rulefilesprovider.ValidURL(item.GeoSiteURL) {
			return model.ClientRouting{}, fmt.Errorf("client rule file %s needs HTTPS geoip.dat and geosite.dat URLs", item.Tag)
		}
		if result.GeoIPURL != "" && (result.GeoIPURL != item.GeoIPURL || result.GeoSiteURL != item.GeoSiteURL) {
			return model.ClientRouting{}, fmt.Errorf("client policies require the same geoip.dat and geosite.dat files")
		}
		result.GeoIPURL, result.GeoSiteURL = item.GeoIPURL, item.GeoSiteURL
		data, err := s.Data.Load(ctx, item, policy.RuleCode)
		if err != nil {
			return model.ClientRouting{}, err
		}
		result.Rules = appendGeoRules(result.Rules, policy.RuleCode, local, data.Domains, data.IPs)
	}
	return result, nil
}

func appendGeoRules(rules []map[string]any, code string, direct bool, domains []*geodata.DomainRule, ips []*geodata.IPRule) []map[string]any {
	target := map[string]any{"balancerTag": "foreign"}
	if direct {
		target = map[string]any{"outboundTag": "direct"}
	}
	if len(domains) > 0 {
		rule := map[string]any{"type": "field", "domain": []string{"geosite:" + code}}
		for key, value := range target {
			rule[key] = value
		}
		rules = append(rules, rule)
	}
	if len(ips) > 0 {
		rule := map[string]any{"type": "field", "ip": []string{"geoip:" + code}}
		for key, value := range target {
			rule[key] = value
		}
		rules = append(rules, rule)
	}
	return rules
}

func clientPolicyTarget(policy model.Policy, nodes []nodemodel.Node, region string) (use, direct bool, err error) {
	foreignCount, matchedCount := 0, 0
	for _, node := range nodes {
		if !strings.EqualFold(node.Country, region) {
			foreignCount++
			if matches(policy.Source, node.Country) {
				matchedCount++
			}
		}
	}
	if matchedCount == 0 {
		return false, false, nil
	}
	local, found := false, false
	for _, node := range nodes {
		if matches(policy.Target, node.Country) {
			found = true
			if !strings.EqualFold(node.Country, region) {
				local = false
				break
			}
			local = true
		}
	}
	if !found {
		return false, false, fmt.Errorf("policy %s has no target nodes", policy.Tag)
	}
	return true, local && matchedCount == foreignCount, nil
}
