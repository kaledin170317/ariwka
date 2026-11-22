package happ

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// The routing header tells Happ to download the geo files for the full JSON.
// Its DirectSites/DirectIp are deliberately empty: routing decisions stay in
// the Xray config, which Happ passes through without injecting profile rules.
func geoRoutingLink(geoIPURL, geoSiteURL string) (string, error) {
	if geoIPURL == "" && geoSiteURL == "" {
		return "", nil
	}
	if geoIPURL == "" || geoSiteURL == "" {
		return "", errors.New("both geo file URLs are required")
	}
	profile := map[string]any{
		"Name": "ARIWKA VPN Geo", "GlobalProxy": "true",
		"UseChunkFiles": true,
		"RemoteDNSType": "DoH", "RemoteDNSIP": "8.8.8.8", "RemoteDNSDomain": "https://8.8.8.8/dns-query",
		"DomesticDNSType": "DoH", "DomesticDNSIP": "8.8.8.8", "DomesticDNSDomain": "https://8.8.8.8/dns-query",
		"Geoipurl": geoIPURL, "Geositeurl": geoSiteURL,
		"DirectSites": []string{}, "DirectIp": []string{},
	}
	data, err := json.Marshal(profile)
	if err != nil {
		return "", err
	}
	return "happ://routing/onadd/" + base64.StdEncoding.EncodeToString(data), nil
}
