package handlers

import (
	api "vpn/gen/node"
	"vpn/internal/node/sing-box/service"
)

type VPNHandler struct {
	api.UnimplementedVPNServiceServer

	service *service.SingBoxService
}

func NewVPNHandler(service *service.SingBoxService) *VPNHandler {
	return &VPNHandler{service: service}
}
