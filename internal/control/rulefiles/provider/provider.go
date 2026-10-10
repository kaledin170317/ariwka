package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/xtls/xray-core/common/geodata"
	"google.golang.org/protobuf/proto"

	"vpn/internal/control/rulefiles/model"
)

type Data struct {
	Domains []*geodata.DomainRule
	IPs     []*geodata.IPRule
}

type entry struct {
	content []byte
	until   time.Time
}

// Provider loads Xray's native geoip.dat and geosite.dat, shared by all policies.
type Provider struct {
	mu    sync.Mutex
	cache map[string]entry
	http  *http.Client
}

func New() *Provider {
	return &Provider{cache: make(map[string]entry), http: &http.Client{Timeout: 30 * time.Second}}
}

func (p *Provider) Load(ctx context.Context, item model.Item, code string) (Data, error) {
	var result Data

	if item.GeoIPURL != "" {
		content, err := p.download(ctx, item.GeoIPURL)
		if err != nil {
			return Data{}, err
		}

		ips := new(geodata.GeoIPList)

		err = proto.Unmarshal(content, ips)
		if err != nil {
			return Data{}, fmt.Errorf("decode %s: %w", item.GeoIPURL, err)
		}

		for _, group := range ips.Entry {
			if !strings.EqualFold(group.Code, code) {
				continue
			}

			for _, cidr := range group.Cidr {
				result.IPs = append(result.IPs, &geodata.IPRule{Value: &geodata.IPRule_Custom{Custom: &geodata.CIDRRule{Cidr: cidr}}})
			}
		}

		if len(result.IPs) == 0 {
			return Data{}, fmt.Errorf("geoip code %s not found in %s", code, item.GeoIPURL)
		}
	}

	if item.GeoSiteURL != "" {
		content, err := p.download(ctx, item.GeoSiteURL)
		if err != nil {
			return Data{}, err
		}

		sites := new(geodata.GeoSiteList)

		err = proto.Unmarshal(content, sites)
		if err != nil {
			return Data{}, fmt.Errorf("decode %s: %w", item.GeoSiteURL, err)
		}

		for _, group := range sites.Entry {
			if !strings.EqualFold(group.Code, code) {
				continue
			}

			for _, domain := range group.Domain {
				result.Domains = append(result.Domains, &geodata.DomainRule{Value: &geodata.DomainRule_Custom{Custom: domain}})
			}
		}

		if len(result.Domains) == 0 {
			return Data{}, fmt.Errorf("geosite code %s not found in %s", code, item.GeoSiteURL)
		}
	}

	return result, nil
}

func ValidURL(raw string) bool {
	if raw == "" {
		return true
	}

	u, err := url.Parse(raw)

	return err == nil && u.Scheme == "https" && u.Host != "" && strings.HasSuffix(u.Path, ".dat")
}

func (p *Provider) download(ctx context.Context, address string) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	old, ok := p.cache[address]
	if ok && time.Now().Before(old.until) {
		return old.content, nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}

	response, err := p.http.Do(request)
	if err != nil {
		if ok {
			return old.content, nil
		}

		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		if ok {
			return old.content, nil
		}

		return nil, fmt.Errorf("download %s: HTTP %d", address, response.StatusCode)
	}

	content, err := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
	if err != nil {
		return nil, err
	}

	if len(content) > 8<<20 {
		return nil, fmt.Errorf("rule set %s is too large", address)
	}

	p.cache[address] = entry{content: content, until: time.Now().Add(24 * time.Hour)}

	return content, nil
}
