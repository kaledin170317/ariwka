package happ

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestGeoRoutingUsesChunkFiles(t *testing.T) {
	link, err := geoRoutingLink("https://example.com/geoip.dat", "https://example.com/geosite.dat")
	if err != nil || !strings.HasPrefix(link, "happ://routing/onadd/") {
		t.Fatalf("invalid routing link: %q (%v)", link, err)
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(link, "happ://routing/onadd/"))
	if err != nil {
		t.Fatal(err)
	}
	var profile struct {
		UseChunkFiles     bool `json:"UseChunkFiles"`
		RemoteDNSType     string
		RemoteDNSDomain   string
		DomesticDNSType   string
		DomesticDNSDomain string
	}
	if err := json.Unmarshal(data, &profile); err != nil || !profile.UseChunkFiles {
		t.Fatalf("Happ must trim downloaded geo files: %s (%v)", data, err)
	}
	if profile.RemoteDNSType != "DoH" || profile.DomesticDNSType != "DoH" || profile.RemoteDNSDomain != "https://8.8.8.8/dns-query" || profile.DomesticDNSDomain != profile.RemoteDNSDomain {
		t.Fatalf("tunnel DNS must use DoH too: %s", data)
	}
}
