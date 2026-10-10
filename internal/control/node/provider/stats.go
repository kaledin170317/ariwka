package provider

import (
	"context"

	"github.com/xtls/xray-core/app/stats/command"
)

// XrayTraffic drains user counters for this node. Xray resets them atomically
// when returning each value; the collector persists those deltas.
func (n *Node) XrayTraffic(ctx context.Context) ([]*command.Stat, error) {
	response, err := n.xrayStats.QueryStats(ctx, &command.QueryStatsRequest{Pattern: "user>>>", Reset_: true})
	if err != nil {
		return nil, err
	}

	return response.Stat, nil
}
