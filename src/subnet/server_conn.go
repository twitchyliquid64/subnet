package subnet

import (
	"encoding/gob"
	"log"
	"net"
	"subnet/conn"
	"time"
)

type serverConn struct {
	conn net.Conn
	id   int

	outboundIPPkts      chan *IPPacket
	outboundUDPInfoPkts chan *conn.UDPInfo

	server     *Server
	canSendIP  bool
	remoteAddr net.IP

	recvUDPPort int //our listening port
	recvUDPKey  [32]byte
	sendUDPAddr net.Addr //client port
	sendUDPKey  [32]byte
	hasUDPKey   bool
	hasUDPAddr  bool

	connectionOk bool
}

func (c *serverConn) initClient(s *Server) {
	c.outboundIPPkts = make(chan *IPPacket, servPerClientPktQueue)
	c.outboundUDPInfoPkts = make(chan *conn.UDPInfo)
	c.sendUDPKey = *conn.NewEncryptionKey()
	c.recvUDPPort = getPort()

	c.connectionOk = true
	c.server = s
	log.Printf("New connection from %s (%d)\n", c.conn.RemoteAddr().String(), c.id)
	go c.readRoutine(&s.isShuttingDown, s.inboundIPPkts)
	go c.udpReadRoutine(&s.isShuttingDown, s.inboundIPPkts)
	go c.writeRoutine(&s.isShuttingDown)
}

func (c *serverConn) udpReadRoutine(isShuttingDown *bool, ipPacketSink chan *inboundIPPkt) {
	defer freePort(c.recvUDPPort)
	udpAddr := &net.UDPAddr{
		Port: c.recvUDPPort,
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Println("Error listening for UDP: ", err)
		return
	}
	defer udpConn.Close()

	buf := make([]byte, devPktBuffSize+128) //extra encryption bytes etc.

	for !*isShuttingDown && c.connectionOk {
		udpConn.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(250)))
		n, addr, err := udpConn.ReadFromUDP(buf)
		if e, ok := err.(net.Error); ok && e.Timeout() {
			continue
		}
		if err != nil {
			log.Println("ReadUDP Error: ", err)
			return
		}

		log.Printf("UDP from %s of len %d.\n", addr.String(), n)

		plainText, err := conn.Decrypt(buf[:n], &c.sendUDPKey)
		if err != nil {
			log.Printf("Decryption error for %d - dropping. %s\n", c.id, err.Error())
			continue
		}
		c.sendUDPAddr = udpConn.RemoteAddr()
		c.hasUDPAddr = true

		ipPacketSink <- &inboundIPPkt{pkt: &IPPacket{Raw: plainText}, clientID: c.id}
	}
}

func (c *serverConn) writeRoutine(isShuttingDown *bool) {
	encoder := gob.NewEncoder(c.conn)

	encoder.Encode(conn.PktUDPInfo)
	encoder.Encode(conn.UDPInfo{Port: c.recvUDPPort, Key: c.sendUDPKey})

	for !*isShuttingDown && c.connectionOk {
		select { //TODO: Fix minor goroutine leak with a timeout ticker
		case pkt := <-c.outboundIPPkts:
			log.Println(c.recvUDPKey, c.sendUDPAddr)
			encoder.Encode(conn.PktIPPkt)
			err := encoder.Encode(pkt)
			if err != nil {
				log.Printf("Write error for %s: %s\n", c.conn.RemoteAddr().String(), err.Error())
				c.hadError(false)
				return
			}

		case pkt := <-c.outboundUDPInfoPkts:
			encoder.Encode(conn.PktUDPInfo)
			err := encoder.Encode(pkt)
			if err != nil {
				log.Printf("UDPInfo Write error for %s: %s\n", c.conn.RemoteAddr().String(), err.Error())
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

		case conn.PktUDPInfo:
			var info conn.UDPInfo
			err := decoder.Decode(&info)
			if err != nil {
				log.Printf("Could not decode conn.UDPInfo: %s", err.Error())
				c.hadError(false)
				return
			}
			c.recvUDPKey = info.Key
			c.hasUDPKey = true

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
