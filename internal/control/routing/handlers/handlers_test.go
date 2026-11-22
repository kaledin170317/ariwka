package handlers

import (
	"testing"

	"google.golang.org/protobuf/proto"

	api "vpn/gen/control"
)

func TestPolicyFromProtoScope(t *testing.T) {
	legacy := policyFromProto(&api.RoutePolicy{Tag: "legacy"})
	if !legacy.ApplyToNodes || legacy.ApplyToClients {
		t.Fatalf("legacy policies should remain node-only: %+v", legacy)
	}
	client := policyFromProto(&api.RoutePolicy{Tag: "client", ApplyToNodes: proto.Bool(false), ApplyToClients: proto.Bool(true)})
	if client.ApplyToNodes || !client.ApplyToClients {
		t.Fatalf("client-only policy was not preserved: %+v", client)
	}
}
