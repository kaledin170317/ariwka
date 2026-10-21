package happ

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

// xraySubscription builds a single full Xray configuration, not a list of
// independent servers. Happ passes full JSON configurations to Xray as-is.
func xraySubscription(links []string) ([]byte, error) {
	outbounds := make([]any, 0, len(links))
	selectors := make([]string, 0, len(links))
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
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 || u.Hostname() == "" || u.User.Username() == "" {
			return nil, fmt.Errorf("invalid VLESS endpoint: %s", u.Host)
		}
		q := u.Query()
		if q.Get("security") != "reality" || q.Get("pbk") == "" || q.Get("sni") == "" {
			return nil, errors.New("failover JSON requires VLESS/REALITY")
		}
		tag := fmt.Sprintf("vpn-%d", len(selectors)+1)
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
			stream["grpcSettings"] = map[string]any{"serviceName": q.Get("serviceName")}
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
		selectors = append(selectors, tag)
	}
	if len(selectors) == 0 {
		return nil, errors.New("no VLESS servers for failover")
	}

	config := map[string]any{
		"remarks": "🌐 ARIWKA VPN failover",
		"dns":     map[string]any{"servers": []string{"1.1.1.1"}},
		"inbounds": []any{
			map[string]any{"tag": "socks-in", "listen": "127.0.0.1", "port": 10808, "protocol": "socks", "settings": map[string]any{"udp": true}},
			map[string]any{"tag": "http-in", "listen": "127.0.0.1", "port": 10809, "protocol": "http"},
		},
		"outbounds":   outbounds,
		"observatory": map[string]any{"subjectSelector": []string{"vpn-"}, "probeURL": "https://www.gstatic.com/generate_204", "probeInterval": "10s"},
		"routing": map[string]any{
			"balancers": []any{map[string]any{"tag": "failover", "selector": []string{"vpn-"},
				"strategy": map[string]any{"type": "leastPing"}, "fallbackTag": selectors[0]}},
			"rules": []any{map[string]any{"type": "field", "inboundTag": []string{"socks-in", "http-in"}, "balancerTag": "failover"}},
		},
	}
	return json.Marshal(config)
}
