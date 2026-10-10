package provider

import (
	"context"

	"github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/app/router/command"
	"github.com/xtls/xray-core/common/serial"
)

func (n *Node) XrayRules(ctx context.Context) ([]*command.ListRuleItem, error) {
	response, err := n.xrayRoute.ListRule(ctx, &command.ListRuleRequest{})
	if err != nil {
		return nil, err
	}

	return response.Rules, nil
}

func (n *Node) XraySetRules(ctx context.Context, rules *router.Config) error {
	_, err := n.xrayRoute.AddRule(ctx, &command.AddRuleRequest{Config: serial.ToTypedMessage(rules)})
	return err
}
