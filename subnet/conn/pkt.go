package conn

// PktType represents the type of a packet on the wire.
type PktType byte

// Packet types
const (
	PktUnknown PktType = iota
	PktIPPkt
	PktLocalAddr
)
