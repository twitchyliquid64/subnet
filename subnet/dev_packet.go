package subnet

import (
	"net"

	"github.com/songgao/water/waterutil"
)

// IPPacket represents a packet in transit over the VPN.
type IPPacket struct {
	Raw      []byte
	Dest     net.IP
	Protocol waterutil.IPProtocol
}

type inboundIPPkt struct {
	pkt      *IPPacket
	clientID int
}
