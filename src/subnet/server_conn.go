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
}

func (c *serverConn) initClient(s *Server) {
	c.outboundIPPkts = make(chan *IPPacket, 2)
	c.server = s
	log.Printf("New connection from %s (%d)\n", c.conn.RemoteAddr().String(), c.id)
	go c.readRoutine(&s.isShuttingDown, s.inboundIPPkts)
	go c.writeRoutine(&s.isShuttingDown)
}

func (c *serverConn) writeRoutine(isShuttingDown *bool) {
	encoder := gob.NewEncoder(c.conn)

	for !*isShuttingDown {
		select {
		case pkt := <-c.outboundIPPkts:
			encoder.Encode(conn.PktIPPkt)
			err := encoder.Encode(pkt)
			if err != nil {
				log.Printf("Write error for %s: %s\n", c.conn.RemoteAddr().String(), err.Error())
				return
			}
		}
	}
}

func (c *serverConn) readRoutine(isShuttingDown *bool, ipPacketSink chan *inboundIPPkt) {
	decoder := gob.NewDecoder(c.conn)

	for !*isShuttingDown {
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
			log.Printf("Packet Received: dest %s, len %d\n", ipPkt.Dest.String(), len(ipPkt.Raw))
			ipPacketSink <- &inboundIPPkt{pkt: &ipPkt, clientID: c.id}
		}
	}
}

func (c *serverConn) queueIP(pkt *inboundIPPkt) {
	c.outboundIPPkts <- pkt.pkt
}

func (c *serverConn) hadError(errInRead bool) {
	if !errInRead {
		c.conn.Close()
	}
	c.server.removeClientConn(c.id)
}
