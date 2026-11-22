package happ

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestXrayRoutedDevice(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	links := []string{
		"hysteria2://secret@somewhere:443",
		"vless://11111111-1111-4111-8111-111111111111@fi.example:443?security=reality&pbk=" + key + "&sni=example.com&fp=chrome&type=grpc&serviceName=vpn",
		"vless://22222222-2222-4222-8222-222222222222@ru.example:443?security=reality&pbk=" + key + "&sni=example.com&fp=chrome&type=grpc&serviceName=vpn",
		"vless://33333333-3333-4333-8333-333333333333@ru2.example:443?security=reality&pbk=" + key + "&sni=example.com&fp=chrome&type=grpc&serviceName=vpn",
	}
	policyRules := []map[string]any{{"type": "field", "domain": []string{"domain:example.ru"}, "outboundTag": "direct"}}
	data, err := xraySubscriptionMode(links, "RU", map[string]string{
		"11111111-1111-4111-8111-111111111111": "FI",
		"22222222-2222-4222-8222-222222222222": "RU",
		"33333333-3333-4333-8333-333333333333": "RU",
	}, policyRules, true)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		DNS    struct{ Servers []string } `json:"dns"`
		Policy struct {
			Levels map[string]struct{ BufferSize int }
		} `json:"policy"`
		Outbounds []struct {
			Tag            string `json:"tag"`
			StreamSettings struct {
				GRPCSettings struct {
					InitialWindowSize int `json:"initial_windows_size"`
				} `json:"grpcSettings"`
			} `json:"streamSettings"`
		} `json:"outbounds"`
		Routing struct {
			Balancers []struct {
				Tag, FallbackTag string
				Selector         []string
			} `json:"balancers"`
			Rules []struct {
				InboundTag  []string `json:"inboundTag"`
				OutboundTag string   `json:"outboundTag"`
				BalancerTag string   `json:"balancerTag"`
				Network     string   `json:"network"`
			} `json:"rules"`
		} `json:"routing"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.DNS.Servers) != 1 || config.DNS.Servers[0] != "https://8.8.8.8/dns-query" || config.Policy.Levels["0"].BufferSize != 4 {
		t.Fatalf("missing mobile DNS/buffer settings: %s", data)
	}
	for _, outbound := range config.Outbounds {
		if outbound.Tag == "foreign-1" || outbound.Tag == "domestic-1" || outbound.Tag == "domestic-2" {
			if outbound.StreamSettings.GRPCSettings.InitialWindowSize != 65536 {
				t.Fatalf("gRPC window must be fixed for %s", outbound.Tag)
			}
		}
	}
	if len(config.Outbounds) != 5 || config.Outbounds[0].Tag != "block" || config.Outbounds[1].Tag != "foreign-1" || config.Outbounds[2].Tag != "domestic-1" || config.Outbounds[3].Tag != "domestic-2" {
		t.Fatalf("wrong node classification: %s", data)
	}
	if len(config.Routing.Balancers) != 1 || config.Routing.Balancers[0].FallbackTag != "direct" {
		t.Fatalf("wrong failover chain: %s", data)
	}
	if len(config.Routing.Rules) != 2 || config.Routing.Rules[0].OutboundTag != "direct" || config.Routing.Rules[1].BalancerTag != "foreign" {
		t.Fatalf("wrong routing order: %s", data)
	}
	if len(config.Routing.Rules[0].InboundTag) != 0 || len(config.Routing.Rules[1].InboundTag) != 0 || config.Routing.Rules[1].Network != "tcp,udp" {
		t.Fatalf("tunnel traffic must match regardless of Happ inbound tag: %s", data)
	}
}

func TestXrayRoutedDeviceFailsClosed(t *testing.T) {
	link := "vless://11111111-1111-4111-8111-111111111111@fi.example:443?security=reality&pbk=key&sni=example.com&fp=chrome&type=grpc"
	if _, err := xraySubscriptionMode([]string{link}, "RU", map[string]string{"11111111-1111-4111-8111-111111111111": "FI"}, nil, true); err == nil {
		t.Fatal("missing domestic reserve accepted")
	}
	if _, err := xraySubscription([]string{link}, "RU", nil, nil); err == nil {
		t.Fatal("unknown node country accepted")
	}
}

func TestXrayDiagnosticMode(t *testing.T) {
	link := "vless://11111111-1111-4111-8111-111111111111@fi.example:443?security=reality&pbk=key&sni=example.com&fp=chrome&type=grpc"
	rules := []map[string]any{
		{"type": "field", "domain": []string{"geosite:ru-standard"}, "outboundTag": "direct"},
		{"type": "field", "domain": []string{"example.com"}, "balancerTag": "foreign"},
	}
	data, err := xraySubscriptionMode([]string{link}, "RU", map[string]string{"11111111-1111-4111-8111-111111111111": "FI"}, rules, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"observatory", "balancers", "balancerTag", "domestic-loop", "domestic-in"} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatalf("diagnostic config contains %s: %s", forbidden, data)
		}
	}
	var config struct {
		Routing struct {
			Rules []struct {
				OutboundTag, Network string
				Domain               []string
			}
		}
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	r := config.Routing.Rules
	if len(r) != 3 || r[0].OutboundTag != "direct" || r[0].Domain[0] != "geosite:ru-standard" || r[1].OutboundTag != "foreign-1" || r[2].OutboundTag != "foreign-1" || r[2].Network != "tcp,udp" {
		t.Fatalf("incorrect static routing: %s", data)
	}
	if rules[1]["balancerTag"] != "foreign" {
		t.Fatal("input rules mutated")
	}
	if _, err := xraySubscription([]string{link}, "RU", map[string]string{"11111111-1111-4111-8111-111111111111": "RU"}, nil); err == nil {
		t.Fatal("missing foreign node accepted")
	}
}

func TestXraySingleDomesticReserve(t *testing.T) {
	links := []string{
		"vless://fi@fi.example:443?security=reality&pbk=key&sni=example.com&fp=chrome&type=grpc",
		"vless://ru@ru.example:443?security=reality&pbk=key&sni=example.com&fp=chrome&type=grpc",
	}
	data, err := xraySubscription(links, "RU", map[string]string{"fi": "FI", "ru": "RU"}, []map[string]any{
		{"type": "field", "domain": []string{"geosite:ru-standard"}, "outboundTag": "direct"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Remarks     string
		Observatory struct {
			SubjectSelector   []string
			ProbeInterval     string
			EnableConcurrency bool
		}
		Routing struct {
			Balancers []struct {
				Tag, FallbackTag string
				Selector         []string
				Strategy         struct{ Type string }
			}
			Rules []struct{ OutboundTag, BalancerTag string }
		}
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if config.Remarks != "🇷🇺 Авто" || config.Observatory.ProbeInterval != "60s" || config.Observatory.EnableConcurrency || len(config.Observatory.SubjectSelector) != 2 || config.Observatory.SubjectSelector[0] != "foreign-" || config.Observatory.SubjectSelector[1] != "domestic-" {
		t.Fatalf("incorrect lightweight probing: %s", data)
	}
	if len(config.Routing.Balancers) != 1 || config.Routing.Balancers[0].FallbackTag != "direct" || config.Routing.Balancers[0].Tag != "foreign" || bytes.Contains(data, []byte("loopback")) {
		t.Fatalf("random pool must fall back directly without loopback: %s", data)
	}
	b := config.Routing.Balancers[0]
	if b.Strategy.Type != "random" || len(b.Selector) != 2 || b.Selector[0] != "foreign-" || b.Selector[1] != "domestic-" {
		t.Fatalf("incorrect random pool: %s", data)
	}
	if len(config.Routing.Rules) != 2 || config.Routing.Rules[0].OutboundTag != "direct" || config.Routing.Rules[1].BalancerTag != "foreign" {
		t.Fatalf("incorrect policy/fallback routing: %s", data)
	}
}
