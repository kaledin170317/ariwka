package happ

import "encoding/base64"

const dnsRoutingProfile = `{"Name":"ARIWKA VPN DNS","GlobalProxy":"true","RemoteDNSType":"DoU","RemoteDNSIP":"1.1.1.1","DomesticDNSType":"DoU","DomesticDNSIP":"1.1.1.1"}`

var dnsRoutingLink = "happ://routing/onadd/" + base64.StdEncoding.EncodeToString([]byte(dnsRoutingProfile))
