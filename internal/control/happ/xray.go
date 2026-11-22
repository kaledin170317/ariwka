package happ

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// xraySubscription builds a single full Xray configuration, not a list of
// independent servers. Happ passes full JSON configurations to Xray as-is.
func xraySubscription(links []string, region string, countries map[string]string, policyRules []map[string]any) ([]byte, error) {
	return xraySubscriptionMode(links, region, countries, policyRules, true)
}

func xraySubscriptionMode(links []string, region string, countries map[string]string, policyRules []map[string]any, failover bool) ([]byte, error) {
	outbounds := make([]any, 0, len(links))
	foreign, domestic := 0, 0
	if region == "" {
		return nil, errors.New("device region required")
	}
	for _, link := range links {
		u, err := url.Parse(link)
		if err != nil {
			return nil, fmt.Errorf("invalid subscription link: %w", err)
		}
		// The routed device can also have Hysteria2 profiles. Only VLESS/REALITY
		// has an outbound representation here; never mix in a default-network
		// profile or turn an unsupported transport into a broken Xray config.
		if u.Scheme == "hysteria2" {
			continue
		}
		if u.Scheme != "vless" || u.User == nil {
			return nil, errors.New("failover JSON supports only VLESS links")
		}
		country := countries[u.User.Username()]
		if country == "" {
			return nil, errors.New("node country required for VLESS profile")
		}
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 || u.Hostname() == "" || u.User.Username() == "" {
			return nil, fmt.Errorf("invalid VLESS endpoint: %s", u.Host)
		}
		q := u.Query()
		if q.Get("security") != "reality" || q.Get("pbk") == "" || q.Get("sni") == "" {
			return nil, errors.New("failover JSON requires VLESS/REALITY")
		}
		tag := "foreign-"
		if strings.EqualFold(region, country) {
			domestic++
			tag = fmt.Sprintf("domestic-%d", domestic)
		} else {
			foreign++
			tag = fmt.Sprintf("foreign-%d", foreign)
		}
		stream := map[string]any{
			"security": "reality",
			"realitySettings": map[string]any{
				"serverName": q.Get("sni"), "fingerprint": q.Get("fp"),
				"publicKey": q.Get("pbk"), "shortId": q.Get("sid"),
			},
		}
		switch q.Get("type") {
		case "grpc":
			stream["network"] = "grpc"
			// A fixed HTTP/2 stream window disables gRPC's dynamic window
			// growth, limiting queued receive data on memory-constrained clients.
			stream["grpcSettings"] = map[string]any{"serviceName": q.Get("serviceName"), "initial_windows_size": 65536}
		case "xhttp":
			stream["network"] = "xhttp"
			stream["xhttpSettings"] = map[string]any{"path": q.Get("path"), "host": q.Get("host")}
		default:
			return nil, errors.New("unsupported VLESS transport in failover JSON")
		}
		outbounds = append(outbounds, map[string]any{
			"tag": tag, "protocol": "vless",
			"settings": map[string]any{"vnext": []any{map[string]any{
				"address": u.Hostname(), "port": port,
				"users": []any{map[string]any{"id": u.User.Username(), "encryption": "none"}},
			}}},
			"streamSettings": stream,
		})
	}
	if foreign == 0 || (failover && domestic == 0) {
		return nil, errors.New("failover needs VLESS servers both outside and inside device region")
	}
	// Unmatched traffic is blocked; direct is an explicit balancer fallback.
	outbounds = append([]any{map[string]any{"tag": "block", "protocol": "blackhole"}}, outbounds...)
	outbounds = append(outbounds,
		map[string]any{"tag": "direct", "protocol": "freedom"},
	)
	rules := make([]any, 0, len(policyRules)+1)
	for _, rule := range policyRules {
		// Match Happ's tunnel without depending on its platform-specific tag.
		rules = append(rules, rule)
	}
	rules = append(rules, map[string]any{"type": "field", "network": "tcp,udp", "balancerTag": "foreign"})

	config := map[string]any{
		"remarks": "🇷🇺 Авто",
		// Use an IP-literal DoH endpoint so resolving the DNS server itself
		// does not require another DNS connection. HTTPS uses normal routing.
		"dns": map[string]any{"servers": []string{"https://8.8.8.8/dns-query"}},
		"policy": map[string]any{"levels": map[string]any{
			"0": map[string]any{"bufferSize": 4},
		}},
		"inbounds": []any{
			map[string]any{"tag": "socks-in", "listen": "127.0.0.1", "port": 10808, "protocol": "socks", "settings": map[string]any{"udp": true}},
			map[string]any{"tag": "http-in", "listen": "127.0.0.1", "port": 10809, "protocol": "http"},
		},
		"outbounds":   outbounds,
		"observatory": map[string]any{"subjectSelector": []string{"foreign-", "domestic-"}, "probeURL": "https://www.gstatic.com/generate_204", "probeInterval": "60s", "enableConcurrency": false},
		"routing": map[string]any{
			"domainStrategy": "IPIfNonMatch",
			"balancers": []any{
				// Keep the tag used by client policies; candidates now include both regions.
				map[string]any{"tag": "foreign", "selector": []string{"foreign-", "domestic-"}, "strategy": map[string]any{"type": "random"}, "fallbackTag": "direct"},
			},
			"rules": rules,
		},
	}
	if !failover {
		config["remarks"] = "🌐 Тест без резерва"
		delete(config, "observatory")
		staticRules := make([]any, 0, len(policyRules)+1)
		for _, rule := range policyRules {
			copy := make(map[string]any, len(rule))
			for key, value := range rule {
				copy[key] = value
			}
			if _, ok := copy["balancerTag"]; ok {
				delete(copy, "balancerTag")
				copy["outboundTag"] = "foreign-1"
			}
			staticRules = append(staticRules, copy)
		}
		staticRules = append(staticRules, map[string]any{"type": "field", "network": "tcp,udp", "outboundTag": "foreign-1"})
		config["routing"] = map[string]any{"domainStrategy": "IPIfNonMatch", "rules": staticRules}
	}
	return json.Marshal(config)
}
