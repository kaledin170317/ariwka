package port

import (
	"fmt"
	"net"
	"strconv"
)

func Check(port int32) error {
	address := net.JoinHostPort("::", strconv.Itoa(int(port)))

	tcp, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("TCP port %d is unavailable: %w", port, err)
	}
	defer tcp.Close()

	udp, err := net.ListenPacket("udp", address)
	if err != nil {
		return fmt.Errorf("UDP port %d is unavailable: %w", port, err)
	}

	return udp.Close()
}
