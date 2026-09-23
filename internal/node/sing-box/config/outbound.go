package config

func NewDirectOutbound() outbound {
	return outbound{Type: OutboundTypeDirect}
}
