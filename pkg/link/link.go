package link

import (
	"net"
	"net/url"
	"strconv"
)

type Hysteria2 struct {
	Tag, Address, Password, ServerName, PinSHA256 string
	Port                                          int32
	Insecure                                      bool
}

func (h Hysteria2) String() string {
	params := url.Values{}
	if h.Insecure || h.PinSHA256 != "" {
		params.Set("insecure", "1")
	}

	if h.PinSHA256 != "" {
		params.Set("pinSHA256", h.PinSHA256)
	}

	if h.ServerName != "" {
		params.Set("sni", h.ServerName)
	}

	return (&url.URL{
		Scheme:   "hysteria2",
		User:     url.User(h.Password),
		Host:     net.JoinHostPort(h.Address, strconv.Itoa(int(h.Port))),
		Path:     "/",
		RawQuery: params.Encode(),
		Fragment: h.Tag,
	}).String()
}

type Transport struct {
	Type, ServiceName, Host, Path string
}

type Vless struct {
	Tag, Address, UUID, ServerName, PublicKey, ShortID string
	Port                                               int32
	Transport                                          Transport
}

func (v Vless) String() string {
	params := url.Values{"encryption": {"none"}, "security": {"reality"},
		"sni": {v.ServerName}, "pbk": {v.PublicKey}, "sid": {v.ShortID}, "fp": {"chrome"}}

	t := v.Transport
	switch t.Type {
	case "grpc":
		params.Set("type", "grpc")
		params.Set("serviceName", t.ServiceName)
	case "http":
		params.Set("type", "http")
		params.Set("path", t.Path)

		if t.Host != "" {
			params.Set("host", t.Host)
		}
	}

	return (&url.URL{Scheme: "vless", User: url.User(v.UUID),
		Host:     net.JoinHostPort(v.Address, strconv.Itoa(int(v.Port))),
		RawQuery: params.Encode(), Fragment: v.Tag}).String()
}
