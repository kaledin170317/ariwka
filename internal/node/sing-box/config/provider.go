package config

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	configPath    = "/etc/sing-box/config.json"
	tmpConfigPath = "/etc/sing-box/config.tmp"
)

func New() Config {
	return Config{}
}

func (c *Config) Sync() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var loaded Config

	err = json.Unmarshal(data, &loaded)
	if err != nil {
		return err
	}

	*c = loaded

	return nil
}

func (c *Config) Apply() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	applyErr := c.apply()
	if applyErr != nil {
		err = os.WriteFile(configPath, data, 0644)
		if err != nil {
			return err
		}

		return errors.Join(applyErr, err, c.Sync())
	}

	return nil
}

func (c *Config) AddRuleSet(name, url, format string, interval uint32) error {
	err := validateRuleSet(name, url, format)
	if err != nil {
		return err
	}

	for _, item := range c.Route.RuleSet {
		if item.Tag == name {
			return fmt.Errorf("rule set %q already exists", name)
		}
	}

	c.Route.RuleSet = append(c.Route.RuleSet, ruleSet{
		Type:           RuleSetTypeRemote,
		Tag:            name,
		Format:         ruleSetFormat(format),
		URL:            url,
		UpdateInterval: ruleSetInterval(interval),
	})
	c.ensureDomainResolver()

	return c.Apply()
}

func (c *Config) GetRuleSets() []RuleSet {
	result := make([]RuleSet, 0, len(c.Route.RuleSet))
	for _, item := range c.Route.RuleSet {
		duration, err := time.ParseDuration(item.UpdateInterval)
		if err != nil {
			duration = 0
		}

		result = append(result, RuleSet{
			Name:                  item.Tag,
			URL:                   item.URL,
			Format:                string(item.Format),
			UpdateIntervalSeconds: uint32(duration / time.Second),
		})
	}

	return result
}

func (c *Config) RemoveRuleSet(name string) error {
	for _, rule := range c.Route.Rules {
		if slices.Contains(rule.RuleSet, name) {
			return fmt.Errorf("rule set %q is used by a route rule", name)
		}
	}

	for i, item := range c.Route.RuleSet {
		if item.Tag == name {
			c.Route.RuleSet = append(c.Route.RuleSet[:i], c.Route.RuleSet[i+1:]...)

			return c.Apply()
		}
	}

	return fmt.Errorf("rule set %q not found", name)
}

func (c *Config) ModifyRuleSet(name, url, format string, interval uint32) error {
	err := validateRuleSet(name, url, format)
	if err != nil {
		return err
	}

	for i := range c.Route.RuleSet {
		if c.Route.RuleSet[i].Tag == name {
			if c.Route.RuleSet[i].Type != RuleSetTypeRemote {
				return fmt.Errorf("rule set %q is not remote", name)
			}

			c.Route.RuleSet[i].URL = url
			c.Route.RuleSet[i].Format = ruleSetFormat(format)
			c.Route.RuleSet[i].UpdateInterval = ruleSetInterval(interval)
			c.ensureDomainResolver()

			return c.Apply()
		}
	}

	return fmt.Errorf("rule set %q not found", name)
}

type RuleSet struct {
	Name                  string
	URL                   string
	Format                string
	UpdateIntervalSeconds uint32
}

func validateRuleSet(name, url, format string) error {
	if name == "" || url == "" {
		return errors.New("rule set name and URL required")
	}

	if format != "" && format != "binary" && format != "source" {
		return errors.New("invalid rule set format")
	}

	return nil
}

func ruleSetFormat(format string) RuleSetFormat {
	if format == "source" {
		return RuleSetFormatSource
	}

	return RuleSetFormatBinary
}

func ruleSetInterval(seconds uint32) string {
	if seconds == 0 {
		return ""
	}

	if seconds%3600 == 0 {
		return fmt.Sprintf("%dh", seconds/3600)
	}

	if seconds%60 == 0 {
		return fmt.Sprintf("%dm", seconds/60)
	}

	return fmt.Sprintf("%ds", seconds)
}

type RouteRule struct {
	Users    []string
	RuleSets []string
	Action   RouteAction
	Outbound string
}

func (c *Config) GetRouteRules() []RouteRule {
	rules := make([]RouteRule, 0, len(c.Route.Rules))
	for _, rule := range c.Route.Rules {
		rules = append(rules, RouteRule{
			Users:    slices.Clone(rule.AuthUser),
			RuleSets: slices.Clone(rule.RuleSet),
			Action:   rule.Action,
			Outbound: rule.Outbound,
		})
	}

	return rules
}

func (c *Config) RemoveRouteRule(index uint32) error {
	if uint64(index) >= uint64(len(c.Route.Rules)) {
		return errors.New("route rule index out of range")
	}

	c.Route.Rules = append(c.Route.Rules[:index], c.Route.Rules[index+1:]...)

	return c.Apply()
}

func (c *Config) AddRoute(user, rule *string, action RouteAction, outboundTag string) error {
	if !action.IsValid() {
		return errors.New("invalid route action")
	}

	if action == RouteActionRoute && outboundTag == "" {
		return errors.New("route outbound required")
	}

	if rule != nil {
		found := false

		for _, item := range c.Route.RuleSet {
			if item.Tag == *rule {
				found = true
			}
		}

		if !found {
			return fmt.Errorf("rule set %q not found", *rule)
		}
	}

	if outboundTag != "" {
		if action != RouteActionRoute {
			return errors.New("outbound requires route action")
		}

		found := false

		for _, item := range c.Outbounds {
			if item.Tag == outboundTag {
				found = true
			}
		}

		if !found {
			return fmt.Errorf("outbound %q not found", outboundTag)
		}
	}

	for i := range c.Route.Rules {
		r := &c.Route.Rules[i]

		if user == nil && rule == nil {
			if len(r.Inbound) == 0 && r.Port == 0 && len(r.IPCIDR) == 0 && len(r.AuthUser) == 0 && len(r.RuleSet) == 0 && r.Action == action && r.Outbound == outboundTag {
				return nil
			}
		} else if rule != nil && len(r.RuleSet) == 1 && r.RuleSet[0] == *rule && r.Action == action && r.Outbound == outboundTag {
			if user != nil {
				if slices.Contains(r.AuthUser, *user) {
					return nil
				}

				r.AuthUser = append(r.AuthUser, *user)

				c.sortRoutes()

				return c.Apply()
			}

			if len(r.AuthUser) == 0 {
				return nil
			}
		}
	}

	r := routeRule{
		Action:   action,
		Outbound: outboundTag,
	}

	if user != nil {
		r.AuthUser = []string{*user}
	}

	if rule != nil {
		r.RuleSet = []string{*rule}
	}

	c.Route.Rules = append(c.Route.Rules, r)

	c.sortRoutes()

	return c.Apply()
}

func (c *Config) apply() error {
	c.syncDefaultRoute()

	f, err := os.Create(tmpConfigPath)
	if err != nil {
		return err
	}

	err = json.MarshalWrite(f, c, jsontext.WithIndent("  "))
	if err != nil {
		_ = f.Close()
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	out, err := exec.Command("sing-box", "check", "-c", tmpConfigPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed validate config: %s", out)
	}

	err = os.Rename(tmpConfigPath, configPath)
	if err != nil {
		return fmt.Errorf("failed replace config: %w", err)
	}

	out, err = exec.Command("systemctl", "restart", "sing-box").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed restart sing-box: %s", out)
	}

	return nil
}

func (c *Config) ensureDomainResolver() {
	if c.Route.DefaultDomainResolver == "" {
		c.Route.DefaultDomainResolver = c.DNS.Final
	}
}

func (c *Config) sortRoutes() {
	sort.SliceStable(c.Route.Rules, func(i, j int) bool {
		return routePriority(c.Route.Rules[i]) < routePriority(c.Route.Rules[j])
	})
}

func (c *Config) syncDefaultRoute() {
	rules := slices.DeleteFunc(slices.Clone(c.Route.Rules), func(rule routeRule) bool {
		return len(rule.Inbound) == 1 &&
			len(rule.RuleSet) == 0 && len(rule.AuthUser) == 0 && rule.Port == 0 &&
			rule.Action == RouteActionRoute && rule.Outbound == "direct" &&
			len(rule.IPCIDR) == 0 && strings.HasPrefix(rule.Inbound[0], "default:")
	})

	for _, inbound := range c.Inbounds {
		if strings.HasPrefix(inbound.Tag, "default:") {
			rules = append(rules, routeRule{
				Inbound:  []string{inbound.Tag},
				Action:   RouteActionRoute,
				Outbound: "direct",
			})
		}
	}

	c.Route.Rules = rules
	c.sortRoutes()
}

func routePriority(rule routeRule) int {
	if len(rule.Inbound) == 1 && strings.HasPrefix(rule.Inbound[0], "default:") &&
		rule.Action == RouteActionRoute && rule.Outbound == "direct" {
		return -2
	}

	if len(rule.Inbound) > 0 && (rule.Port == 53 || rule.Port == 443) && rule.Action == RouteActionRoute && rule.Outbound == "direct" {
		return -1
	}

	switch rule.Action {
	case RouteActionHijackDns:
		return 0
	case RouteActionSniff:
		return 1
	case RouteActionReject:
		return 2
	case RouteActionRoute:
		if len(rule.RuleSet) > 0 {
			return 3
		}

		return 4
	default:
		return 100
	}
}
