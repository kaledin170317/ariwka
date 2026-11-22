package happ

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	routingmodel "vpn/internal/control/routing/model"
	statsmodel "vpn/internal/control/stats/model"
)

type testProfiles struct{ links []string }

func (p testProfiles) DeviceSubscription(context.Context, string, string) ([]string, bool, error) {
	return p.links, false, nil
}
func (p testProfiles) GetSubscription(context.Context, string, bool) ([]string, error) {
	return nil, nil
}
func (p testProfiles) DeviceRouting(context.Context, string) (string, map[string]string, error) {
	return "RU", map[string]string{
		"11111111-1111-4111-8111-111111111111": "FI",
		"22222222-2222-4222-8222-222222222222": "RU",
	}, nil
}

type testStats struct{}

func (testStats) Device(context.Context, string) (statsmodel.Usage, error) {
	return statsmodel.Usage{Uploaded: 1024, Downloaded: 2048}, nil
}
func (testStats) User(context.Context, string, bool) (statsmodel.Usage, error) {
	return statsmodel.Usage{}, nil
}

type testRoutes struct{}

func (testRoutes) ClientRules(context.Context, string) (routingmodel.ClientRouting, error) {
	return routingmodel.ClientRouting{
		Rules:    []map[string]any{{"type": "field", "domain": []string{"geosite:ru-standard"}, "outboundTag": "direct"}},
		GeoIPURL: "https://example.com/geoip.dat", GeoSiteURL: "https://example.com/geosite.dat",
	}, nil
}

func TestDeviceJSONProfileTitle(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	links := []string{
		"vless://11111111-1111-4111-8111-111111111111@fi.example:443?security=reality&pbk=" + key + "&sni=example.com&fp=chrome&type=grpc&serviceName=vpn",
		"vless://22222222-2222-4222-8222-222222222222@ru.example:443?security=reality&pbk=" + key + "&sni=example.com&fp=chrome&type=grpc&serviceName=vpn",
	}
	h := DeviceHandler{Profiles: testProfiles{links: links}, Stats: testStats{}, Routes: testRoutes{}, Metadata: subMetadata, JSON: true}
	r := httptest.NewRequest(http.MethodGet, "/happ/device-json/11111111-1111-4111-8111-111111111111", nil)
	r.SetPathValue("deviceID", "11111111-1111-4111-8111-111111111111")
	r.Header.Set("X-HWID", "device123456")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || w.Header().Get("Profile-Title") != "ARIWKA VPN" || w.Header().Get("Content-Type") != "application/json; charset=utf-8" || w.Header().Get("Subscription-Userinfo") != "upload=1024; download=2048; total=0; expire=-8419161601" {
		t.Fatalf("JSON subscription metadata: %d %v", w.Code, w.Header())
	}
	link := strings.TrimPrefix(w.Header().Get("Routing"), "happ://routing/onadd/")
	data, err := base64.StdEncoding.DecodeString(link)
	if err != nil {
		t.Fatalf("invalid Happ routing header: %v", err)
	}
	var profile struct {
		GeoIPURL   string `json:"Geoipurl"`
		GeoSiteURL string `json:"Geositeurl"`
	}
	if err := json.Unmarshal(data, &profile); err != nil || profile.GeoIPURL != "https://example.com/geoip.dat" || profile.GeoSiteURL != "https://example.com/geosite.dat" {
		t.Fatalf("wrong geo file URLs: %+v (%v)", profile, err)
	}
	if strings.Contains(w.Body.String(), "example.ru") || !strings.Contains(w.Body.String(), "geosite:ru-standard") {
		t.Fatalf("JSON must contain geo tag, not expanded list: %s", w.Body.String())
	}
}
