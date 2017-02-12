package subnet

import (
	"net"

	"github.com/songgao/water/waterutil"
)

// IPPacket represents a packet in transit over the VPN.
type IPPacket struct {
	raw      []byte
	dest     net.IP
	protocol waterutil.IPProtocol
}
