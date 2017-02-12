package subnet

import (
	"encoding/gob"
	"log"
	"net"
	"subnet/conn"
)

type serverConn struct {
	conn net.Conn
	id   int

	outboundIPPkts chan *IPPacket

	server     *Server
	canSendIP  bool
	remoteAddr net.IP

	connectionOk bool
}

func (c *serverConn) initClient(s *Server) {
	c.outboundIPPkts = make(chan *IPPacket, servPerClientPktQueue)
	c.connectionOk = true
	c.server = s
	log.Printf("New connection from %s (%d)\n", c.conn.RemoteAddr().String(), c.id)
	go c.readRoutine(&s.isShuttingDown, s.inboundIPPkts)
	go c.writeRoutine(&s.isShuttingDown)
}

func (c *serverConn) writeRoutine(isShuttingDown *bool) {
	encoder := gob.NewEncoder(c.conn)

	for !*isShuttingDown && c.connectionOk {
		select {
		case pkt := <-c.outboundIPPkts:
			encoder.Encode(conn.PktIPPkt)
			err := encoder.Encode(pkt)
			if err != nil {
				log.Printf("Write error for %s: %s\n", c.conn.RemoteAddr().String(), err.Error())
				c.hadError(false)
				return
			}
		}
	}
}

func (c *serverConn) readRoutine(isShuttingDown *bool, ipPacketSink chan *inboundIPPkt) {
	decoder := gob.NewDecoder(c.conn)

	for !*isShuttingDown && c.connectionOk {
		var pktType conn.PktType
		err := decoder.Decode(&pktType)
		if err != nil {
			if !*isShuttingDown {
				log.Printf("Client read error: %s\n", err.Error())
			}
			c.hadError(true)
			return
		}

		switch pktType {
		case conn.PktLocalAddr:
			var localAddr net.IP
			err := decoder.Decode(&localAddr)
			if err != nil {
				log.Printf("Could not decode net.IP: %s", err.Error())
				c.hadError(false)
				return
			}
			c.remoteAddr = localAddr
			c.server.setAddrForClient(c.id, localAddr)

		case conn.PktIPPkt:
			var ipPkt IPPacket
			err := decoder.Decode(&ipPkt)
			if err != nil {
				log.Printf("Could not decode IPPacket: %s", err.Error())
				c.hadError(false)
				return
			}
			//log.Printf("Packet Received from %d: dest %s, len %d\n", c.id, ipPkt.Dest.String(), len(ipPkt.Raw))
			ipPacketSink <- &inboundIPPkt{pkt: &ipPkt, clientID: c.id}
		}
	}
}

func (c *serverConn) queueIP(pkt *IPPacket) {
	select {
	case c.outboundIPPkts <- pkt:
	default:
		log.Printf("Warning: Dropping packets for %s as outbound msg queue is full.\n", c.remoteAddr.String())
	}
}

func (c *serverConn) hadError(errInRead bool) {
	if !errInRead {
		c.conn.Close()
	}
	c.connectionOk = false
	c.server.removeClientConn(c.id)
}
