package config

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

const (
	managedDNSPrefix = "route-dns:"
	meshDNSServer    = "8.8.8.8"
)

type DNSRoute struct {
	RuleSet  string
	Outbound string
}

// SyncDNSRoutes applies the same destination choices to hijacked DNS queries.
func (c *Config) SyncDNSRoutes(routes []DNSRoute, managedRuleSets []string) error {
	known := make(map[string]bool, len(managedRuleSets))
	for _, tag := range managedRuleSets {
		known[tag] = true
	}

	servers := make([]server, 0, len(c.DNS.Servers))

	var base server

	for _, item := range c.DNS.Servers {
		if item.Tag == c.DNS.Final {
			base = item
		}

		if !strings.HasPrefix(item.Tag, managedDNSPrefix) {
			servers = append(servers, item)
		}
	}

	if base.Tag == "" {
		return errors.New("default DNS server required for routes")
	}

	rules := make([]dnsRule, 0, len(c.DNS.Rules)+len(routes))
	for _, route := range routes {
		if route.Outbound == "" {
			return errors.New("DNS route outbound required")
		}

		if route.Outbound == "direct" && route.RuleSet == "" {
			continue // The default DNS server already uses the local connection.
		}

		serverTag := base.Tag
		if route.Outbound != "direct" {
			serverTag = managedDNSPrefix + route.Outbound
			found := false

			for _, item := range servers {
				if item.Tag == serverTag {
					found = true
					break
				}
			}

			if !found {
				copy := base
				copy.Type = DNSServerTypeHttps
				copy.Tag = serverTag
				copy.Server = meshDNSServer
				copy.ServerPort = 443
				copy.Headers = map[string]string{"Connection": "close"}
				copy.TLS = &serverTLS{
					Enabled:    true,
					ServerName: "dns.google",
					ALPN:       []string{"http/1.1"},
				}
				copy.Detour = route.Outbound
				servers = append(servers, copy)
			}
		}

		rule := dnsRule{Action: "route", Server: serverTag}
		if route.RuleSet != "" {
			rule.RuleSet = []string{route.RuleSet}
		}

		rules = append(rules, rule)
	}

	for _, rule := range c.DNS.Rules {
		managed := strings.HasPrefix(rule.Server, managedDNSPrefix)
		for _, tag := range rule.RuleSet {
			managed = managed || known[tag]
		}

		if !managed {
			rules = append(rules, rule)
		}
	}

	if len(rules) == 0 {
		rules = nil
	}

	meshDNSChanged := c.syncMeshDNSBypass()
	if !meshDNSChanged && reflect.DeepEqual(c.DNS.Servers, servers) && reflect.DeepEqual(c.DNS.Rules, rules) {
		return nil
	}

	c.DNS.Servers = servers
	c.DNS.Rules = rules
	c.ensureDomainResolver()

	err := c.Apply()
	if err != nil {
		return fmt.Errorf("sync DNS routes: %w", err)
	}

	return nil
}

func (c *Config) syncMeshDNSBypass() bool {
	var meshInbounds []string

	for _, inbound := range c.Inbounds {
		if strings.HasPrefix(inbound.Tag, "mesh:") {
			meshInbounds = append(meshInbounds, inbound.Tag)
		}
	}

	var desired []routeRule
	if len(meshInbounds) > 0 {
		desired = []routeRule{
			{
				Inbound:  meshInbounds,
				Port:     53,
				Action:   RouteActionRoute,
				Outbound: "direct",
			},
			{
				Inbound:  meshInbounds,
				IPCIDR:   []string{meshDNSServer + "/32"},
				Port:     443,
				Action:   RouteActionRoute,
				Outbound: "direct",
			},
		}
	}

	rules := slices.DeleteFunc(slices.Clone(c.Route.Rules), func(rule routeRule) bool {
		return len(rule.Inbound) > 0 && rule.Action == RouteActionRoute && rule.Outbound == "direct" &&
			((rule.Port == 53 && len(rule.IPCIDR) == 0) ||
				(rule.Port == 443 && slices.Equal(rule.IPCIDR, []string{meshDNSServer + "/32"})))
	})

	rules = append(desired, rules...)
	if reflect.DeepEqual(c.Route.Rules, rules) {
		return false
	}

	c.Route.Rules = rules

	return true
}
