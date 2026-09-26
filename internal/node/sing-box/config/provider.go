package config

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"sort"

	"vpn/internal/node/sing-box/model/inbound"
)

const (
	configPath    = "/etc/sing-box/config.json"
	tmpConfigPath = "/etc/sing-box/config.tmp.json"
)

func New() Config {
	return Config{}
}

func (c *Config) Sync() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, c)
}

func (c *Config) Apply() error {
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

func (c *Config) AddInbound(inbound inbound.Inbound) error {
	c.Inbounds = append(c.Inbounds, inbound)
	return c.Apply()
}

func (c *Config) AddOutbound(outbound outbound) {
	c.Outbounds = append(c.Outbounds, outbound)
}

func (c *Config) GetInbouns() []inbound.Inbound {
	return c.Inbounds
}

func (c *Config) AddRuleSet(ruleSet RuleSet) error {
	c.Route.RuleSet = append(c.Route.RuleSet, ruleSet)
	return c.Apply()
}

func (c *Config) GetRuleSet() []RuleSet {
	return c.Route.RuleSet
}

func (c *Config) AddRoute(user, rule *string, action RouteAction) error {
	// Ищем уже существующее правило.
	for i := range c.Route.Rules {
		r := &c.Route.Rules[i]

		if user == nil && rule == nil {
			if len(r.AuthUser) == 0 && len(r.RuleSet) == 0 && r.Action == action {
				return nil
			}
			continue
		}

		if rule != nil && len(r.RuleSet) == 1 && r.RuleSet[0] == *rule && r.Action == action {
			if user != nil {
				if slices.Contains(r.AuthUser, *user) {
					return nil
				}
				r.AuthUser = append(r.AuthUser, *user)
				c.sortRoutes()
				return c.Apply()
			}

			return nil
		}
	}

	r := RouteRule{
		Action: action,
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

func (c *Config) sortRoutes() {
	sort.SliceStable(c.Route.Rules, func(i, j int) bool {
		return routePriority(c.Route.Rules[i].Action) < routePriority(c.Route.Rules[j].Action)
	})
}

func routePriority(action RouteAction) int {
	switch action {
	case RouteActionHijackDns:
		return 0
	case RouteActionSniff:
		return 1
	case RouteActionReject:
		return 2
	case RouteActionRoute:
		return 3
	default:
		return 100
	}
}
