package subnet

import (
	"crypto/tls"
	"encoding/gob"
	"errors"
	"log"
	"net"
	"subnet/conn"
	"sync"

	"github.com/songgao/water"
)

// Client represents a connection to a subnet server.
type Client struct {
	manual bool

	wg             sync.WaitGroup
	serverIP       net.IP
	localAddr      net.IP
	localNetMask   *net.IPNet
	isShuttingDown bool

	//channels between various components
	packetsIn     chan *IPPacket
	packetsDevOut chan *IPPacket

	intf    *water.Interface
	tlsConf *tls.Config
	tlsConn *tls.Conn //do not use directly

	reverser Reverser
}

// NewClient constructs a Client object.
func NewClient(servAddr, port, network, iName string, manual bool,
	certPemPath, keyPemPath, caCertPath string) (*Client, error) {

	tlsConf, err := conn.TLSConfig(certPemPath, keyPemPath, caCertPath)
	if err != nil {
		return nil, err
	}

	serverIP, err := hostToIP(servAddr)
	if err != nil {
		return nil, err
	}

	netIP, localNetMask, err := net.ParseCIDR(network)
	if err != nil {
		return nil, errors.New("invalid network address/mask - " + err.Error())
	}

	intf, err := water.NewTUN(iName)
	if err != nil {
		return nil, errors.New("Could not create TUN - " + err.Error())
	}

	log.Printf("Created iface %s\n", intf.Name())

	ret := &Client{
		intf:          intf,
		manual:        manual,
		localAddr:     netIP,
		localNetMask:  localNetMask,
		serverIP:      serverIP,
		tlsConf:       tlsConf,
		packetsIn:     make(chan *IPPacket, pktInMaxBuff),
		packetsDevOut: make(chan *IPPacket, 2),
	}

	return ret, ret.init(servAddr, port)
}

// Initializes connection and changes network configuration as needed, but does not
// activate the client object for use.
func (c *Client) init(serverAddr, port string) error {
	tlsConn, err := openTLSConnection(serverAddr, port, c.tlsConf)
	if err != nil {
		return err
	}
	c.tlsConn = tlsConn

	if err := SetDevIP(c.intf.Name(), c.localAddr, c.localNetMask, false); err != nil {
		return err
	}
	log.Printf("IP of %s set to %s, localNetMask %s\n", c.intf.Name(), c.localAddr.String(), net.IP(c.localNetMask.Mask).String())

	if !c.manual {
		// get default gateway information
		gw, gatewayDevice, err := GetNetGateway()
		if err != nil {
			return err
		}
		gateway := net.ParseIP(gw)
		log.Printf("Default gateway is %s on %s\n", gateway, gatewayDevice)

		// route all traffic to the VPN server through the current gateway device
		if err := AddRoute(c.serverIP, gateway, gatewayDevice, false); err != nil {
			return err
		}
		log.Printf("Traffic to %s now routed via %s on %s.\n", c.serverIP.String(), gw, gatewayDevice)
		c.reverser.AddRouteEntry(c.serverIP, gateway, gatewayDevice)

	}

	return nil
}

func openTLSConnection(dest, port string, conf *tls.Config) (*tls.Conn, error) {
	return tls.Dial("tcp", dest+":"+port, conf)
}

func (c *Client) netSendRoutine() {
	encoder := gob.NewEncoder(c.tlsConn)

	c.sendLocalAddr(encoder)
	for !c.isShuttingDown {
		pkt := <-c.packetsIn
		err := encoder.Encode(conn.PktIPPkt)
		if err != nil {
			log.Println("Encode error: ", err)
		}
		err = encoder.Encode(pkt)
		if err != nil {
			log.Println("Encode error: ", err)
		}
	}
}

func (c *Client) netRecvRoutine() {
	decoder := gob.NewDecoder(c.tlsConn)

	for !c.isShuttingDown {
		var pktType conn.PktType
		err := decoder.Decode(&pktType)
		if err != nil {
			if !c.isShuttingDown {
				log.Printf("Net read error: %s\n", err.Error())
			}
			return
		}

		switch pktType {
		case conn.PktIPPkt:
			var ipPkt IPPacket
			err := decoder.Decode(&ipPkt)
			if err != nil {
				log.Printf("Could not decode IPPacket: %s", err.Error())
				return
			}
			log.Printf("[NET] Packet Received: dest %s, len %d\n", ipPkt.Dest.String(), len(ipPkt.Raw))
			c.packetsDevOut <- &ipPkt
		}
	}
}

func (c *Client) sendLocalAddr(encoder *gob.Encoder) error {
	err := encoder.Encode(conn.PktLocalAddr)
	if err != nil {
		log.Println("Encode error: ", err)
	}
	err = encoder.Encode(c.localAddr)
	if err != nil {
		log.Println("Encode error: ", err)
	}
	return err
}

// Run starts the client.
func (c *Client) Run() {

	if !c.manual { //Redirect default traffic via our VPN
	}

	err := SetInterfaceStatus(c.intf.Name(), true, true)
	if err != nil {
		log.Printf("Could not bring up interface %s: %s\n", c.intf.Name(), err.Error())
		return
	}

	go c.netSendRoutine()
	go devReadRoutine(c.intf, c.packetsIn, &c.wg, &c.isShuttingDown)
	go devWriteRoutine(c.intf, c.packetsDevOut, &c.wg, &c.isShuttingDown)
}

// Close shuts down the client, reversing configuration changes to the system.
func (c *Client) Close() error {
	c.isShuttingDown = true
	c.reverser.Close()
	c.tlsConn.Close()
	e := c.intf.Close()
	if e != nil {
		return e
	}

	//c.wg.Wait() //who cares?
	return nil
}
