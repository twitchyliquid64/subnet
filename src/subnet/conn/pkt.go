package conn

// PktType represents the type of a packet on the wire.
type PktType byte

// Packet types
const (
	PktUnknown PktType = iota
	PktIPPkt
	PktLocalAddr
	PktUDPInfo
)

//UDPInfo represents the data sent on the wire to indicate the remote client
//should start using UDP for packet transfers.
type UDPInfo struct {
	Port int
	Key  [32]byte
}
