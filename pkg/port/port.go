package port

import (
	"fmt"
	"net"
	"strconv"
)

func CheckUDP(port int32) error {
	listener, err := net.ListenPacket("udp", net.JoinHostPort("::", strconv.Itoa(int(port))))
	if err != nil {
		return fmt.Errorf("UDP port %d is unavailable: %w", port, err)
	}
	return listener.Close()
}
