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

	"github.com/google/uuid"

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

	return json.Unmarshal(data, &c.file)
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

func (c *Config) AddHysteria2Inbound(portNumber int32) (*inbound.Hysteria2, error) {
	connection, err := newHysteria2(portNumber)
	if err != nil {
		return nil, err
	}

	created, err := c.addInbound(connection)
	if err != nil {
		return nil, err
	}

	return created.(*inbound.Hysteria2), nil
}

func (c *Config) AddVlessInbound(portNumber int32, serverName string, options inbound.VlessTransportOptions) (*inbound.Vless, error) {
	connection, err := newVless(portNumber, serverName, options)
	if err != nil {
		return nil, err
	}

	created, err := c.addInbound(connection)
	if err != nil {
		return nil, err
	}

	return created.(*inbound.Vless), nil
}

func (c *Config) AddUser(inboundTag string) (inbound.Connection, string, string, error) {
	for i := range c.file.Inbounds {
		if c.file.Inbounds[i].Tag != inboundTag {
			continue
		}

		tag := uuid.NewString()

		var user userConfig

		var credential string

		switch c.file.Inbounds[i].Type {
		case InboundTypeHysteria2:
			user = newHysteria2User(tag)
			credential = user.Password
		case InboundTypeVless:
			user = newVlessUser(tag)
			credential = user.UUID
		default:
			return nil, "", "", fmt.Errorf("unsupported inbound type %q", c.file.Inbounds[i].Type)
		}

		updated := c.file.Inbounds[i]
		updated.Users = append(slices.Clone(updated.Users), user)

		model, err := updated.model()
		if err != nil {
			return nil, "", "", err
		}

		c.file.Inbounds[i] = updated

		err = c.Apply()
		if err != nil {
			return nil, "", "", err
		}

		return model, tag, credential, nil
	}

	return nil, "", "", fmt.Errorf("failed to find inbound %q", inboundTag)
}

func (c *Config) RemoveInbound(tag string) error {
	for i, connection := range c.file.Inbounds {
		if connection.Tag != tag {
			continue
		}

		c.file.Inbounds = append(c.file.Inbounds[:i], c.file.Inbounds[i+1:]...)

		return c.Apply()
	}

	return fmt.Errorf("failed to find inbound %q", tag)
}

func (c *Config) RemoveUser(user, tag string) error {
	for i := range c.file.Inbounds {
		if c.file.Inbounds[i].Tag != tag {
			continue
		}

		for j, existing := range c.file.Inbounds[i].Users {
			if existing.Name == user {
				users := c.file.Inbounds[i].Users
				c.file.Inbounds[i].Users = append(users[:j], users[j+1:]...)

				return c.Apply()
			}
		}

		return fmt.Errorf("user %q not found", user)
	}

	return fmt.Errorf("failed to find inbound %q", tag)
}

func (c *Config) AddOutbound(outbound outbound) {
	c.file.Outbounds = append(c.file.Outbounds, outbound)
}

func (c *Config) GetInbounds() ([]inbound.Connection, error) {
	result := make([]inbound.Connection, 0, len(c.file.Inbounds))
	for i := range c.file.Inbounds {
		model, err := c.file.Inbounds[i].model()
		if err != nil {
			return nil, err
		}

		result = append(result, model)
	}

	return result, nil
}

func (c *Config) AddRuleSet(name, url string) error {
	c.file.Route.RuleSet = append(c.file.Route.RuleSet, ruleSet{
		Type:   RuleSetTypeRemote,
		Tag:    name,
		Format: RuleSetFormatBinary,
		URL:    url,
	})

	return c.Apply()
}

func (c *Config) GetRuleSets() []string {
	result := make([]string, 0, len(c.file.Route.RuleSet))
	for _, rule := range c.file.Route.RuleSet {
		result = append(result, rule.Tag)
	}

	return result
}

func (c *Config) AddRoute(user, rule *string, action RouteAction) error {
	for i := range c.file.Route.Rules {
		r := &c.file.Route.Rules[i]

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

	r := routeRule{
		Action: action,
	}

	if user != nil {
		r.AuthUser = []string{*user}
	}

	if rule != nil {
		r.RuleSet = []string{*rule}
	}

	c.file.Route.Rules = append(c.file.Route.Rules, r)

	c.sortRoutes()

	return c.Apply()
}

func (c *Config) apply() error {
	f, err := os.Create(tmpConfigPath)
	if err != nil {
		return err
	}

	err = json.MarshalWrite(f, &c.file, jsontext.WithIndent("  "))
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

func (c *Config) addInbound(connection inboundConfig) (inbound.Connection, error) {
	model, err := connection.model()
	if err != nil {
		return nil, err
	}

	c.file.Inbounds = append(c.file.Inbounds, connection)

	err = c.Apply()
	if err != nil {
		return nil, err
	}

	return model, nil
}

func (c *Config) sortRoutes() {
	sort.SliceStable(c.file.Route.Rules, func(i, j int) bool {
		return routePriority(c.file.Route.Rules[i].Action) < routePriority(c.file.Route.Rules[j].Action)
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
