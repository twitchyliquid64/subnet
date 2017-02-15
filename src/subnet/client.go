package subnet

import (
	"crypto/tls"
	"encoding/gob"
	"errors"
	"log"
	"net"
	"subnet/conn"
	"sync"
	"time"

	"github.com/songgao/water"
)

// Client represents a connection to a subnet server.
type Client struct {
	newGateway string
	serverAddr string
	port       string

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

	// if false, packets are dropped
	connectionOk  bool
	connResetLock sync.Mutex

	udpInitialized bool
	recvUDPKey     [32]byte
	sendUDPPort    int
	sendUDPEnabled bool
	sendUDPKey     [32]byte
	udpControlPkts chan *conn.UDPInfo
	udpConn        *net.UDPConn

	reverser Reverser
}

// NewClient constructs a Client object.
func NewClient(servAddr, port, network, iName string, newGateway string,
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
		intf:           intf,
		newGateway:     newGateway,
		serverAddr:     servAddr,
		port:           port,
		localAddr:      netIP,
		localNetMask:   localNetMask,
		serverIP:       serverIP,
		tlsConf:        tlsConf,
		packetsIn:      make(chan *IPPacket, pktInMaxBuff),
		packetsDevOut:  make(chan *IPPacket, pktOutMaxBuff),
		udpControlPkts: make(chan *conn.UDPInfo),
	}

	return ret, ret.init(servAddr, port)
}

// Initializes connection and changes network configuration as needed, but does not
// activate the client object for use.
func (c *Client) init(serverAddr, port string) error {
	tlsConn, err := tls.Dial("tcp", serverAddr+":"+port, c.tlsConf)
	if err != nil {
		return err
	}
	c.tlsConn = tlsConn
	c.connectionOk = true

	if err := SetDevIP(c.intf.Name(), c.localAddr, c.localNetMask, false); err != nil {
		return err
	}
	log.Printf("IP of %s set to %s, localNetMask %s\n", c.intf.Name(), c.localAddr.String(), net.IP(c.localNetMask.Mask).String())

	if c.newGateway != "" {
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

// Run starts the client.
func (c *Client) Run() {

	if c.newGateway != "" { //Redirect default traffic via our VPN
		err := SetDefaultGateway(c.newGateway, c.intf.Name(), false)
		if err != nil {
			log.Printf("Could set gateway: %s\n", err.Error())
			return
		}
	}

	err := SetInterfaceStatus(c.intf.Name(), true, false)
	if err != nil {
		log.Printf("Could not bring up interface %s: %s\n", c.intf.Name(), err.Error())
		return
	}

	go c.netSendRoutine()
	go c.netRecvRoutine()
	go devReadRoutine(c.intf, c.packetsIn, &c.wg, &c.isShuttingDown)
	go devWriteRoutine(c.intf, c.packetsDevOut, &c.wg, &c.isShuttingDown)
}

func (c *Client) netSendRoutine() {
	c.wg.Add(1)
	defer c.wg.Done()

	for !c.isShuttingDown {
		encoder := gob.NewEncoder(c.tlsConn)
		connOK := c.connectionOk
		if connOK {
			c.sendLocalAddr(encoder)
		}

		for c.connectionOk && connOK {
			select {
			case pkt := <-c.udpControlPkts:

				err := encoder.Encode(conn.PktUDPInfo)
				if err != nil {
					log.Println("Encode error: ", err)
					c.connectionProblem()
					break
				}
				err = encoder.Encode(pkt)
				if err != nil {
					log.Println("Encode error: ", err)
					c.connectionProblem()
					break
				}

			case pkt := <-c.packetsIn:

				if pkt.Dest.IsMulticast() { //Don't forward multicast
					continue
				}

				if c.sendUDPPort > 0 && c.udpInitialized {
					c.sendViaUDP(pkt.Raw)
					continue
				}

				err := encoder.Encode(conn.PktIPPkt)
				if err != nil {
					log.Println("Encode error: ", err)
					c.connectionProblem()
					break
				}
				err = encoder.Encode(pkt)
				if err != nil {
					log.Println("Encode error: ", err)
					c.connectionProblem()
					break
				}
			}

		}
		time.Sleep(time.Millisecond * 150)
		dropSendBuffer(c.packetsIn)
	}
}

func (c *Client) sendViaUDP(data []byte) {
	//log.Printf("Sending data of len %d\n", len(data))
	ciphertext, err := conn.Encrypt(data, &c.sendUDPKey)
	if err != nil {
		log.Println("Could not encrypt for UDP: ", err)
	}
	_, err = c.udpConn.Write(ciphertext)
	if err != nil {
		log.Println("Could not transmit via UDP: ", err)
	}
}

func dropSendBuffer(buffer chan *IPPacket) {
	for {
		select {
		case <-buffer:
		default:
			return
		}
	}
}

func (c *Client) udpRecvRoutine() {
	c.wg.Add(1)
	defer c.wg.Done()
	udpAddr := &net.UDPAddr{
		IP:   c.serverIP,
		Port: c.sendUDPPort,
	}

	udpConn, err := net.DialUDP("udp", &net.UDPAddr{}, udpAddr)
	c.udpConn = udpConn
	if err != nil {
		log.Println("Error opening socket for UDP: ", err)
		return
	}

	buf := make([]byte, devPktBuffSize+128) //extra encryption bytes etc.
	for c.connectionOk {
		udpConn.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(20)))
		n, _, err := udpConn.ReadFromUDP(buf)
		if e, ok := err.(net.Error); ok && e.Timeout() {
			continue
		}
		if err != nil {
			log.Println("ReadUDP Error: ", err)
			break
		}

		//log.Printf("UDP from %s of len %d.\n", addr.String(), n)

		plainText, err := conn.Decrypt(buf[:n], &c.recvUDPKey)
		if err != nil {
			log.Printf("Decryption error - dropping. %s\n", err.Error())
			continue
		}
		c.packetsDevOut <- &IPPacket{Raw: plainText}
	}
	c.udpInitialized = false
	udpConn.Close()
}

func (c *Client) netRecvRoutine() {
	c.wg.Add(1)
	defer c.wg.Done()

	for !c.isShuttingDown {
		decoder := gob.NewDecoder(c.tlsConn)

		for c.connectionOk {
			var pktType conn.PktType
			err := decoder.Decode(&pktType)

			if err != nil {
				if !c.isShuttingDown {
					log.Printf("Net read error: %s\n", err.Error())
					c.connectionProblem()
					break
				}
				return
			}

			switch pktType {
			default:
				log.Println("Got unexpected packet type: ", pktType)
			case conn.PktUDPInfo:
				var info conn.UDPInfo
				err := decoder.Decode(&info)
				if err != nil {
					log.Printf("Could not decode conn.UDPInfo: %s", err.Error())
					c.connectionProblem()
					break
				}
				c.sendUDPKey = info.Key
				if !c.udpInitialized {
					c.sendUDPPort = info.Port
					c.recvUDPKey = *conn.NewEncryptionKey()
					c.udpControlPkts <- &conn.UDPInfo{Key: c.recvUDPKey}
					c.udpInitialized = true
					go c.udpRecvRoutine()
				}

			case conn.PktIPPkt:
				var ipPkt IPPacket
				err := decoder.Decode(&ipPkt)
				if err != nil {
					log.Printf("Could not decode IPPacket: %s", err.Error())
					c.connectionProblem()
					break
				}
				//log.Printf("[NET] Packet Received: dest %s, len %d\n", ipPkt.Dest.String(), len(ipPkt.Raw))
				c.packetsDevOut <- &ipPkt
			}
		}
		time.Sleep(time.Millisecond * 150)
	}
}

func (c *Client) connectionProblem() {
	if !c.connectionOk {
		return
	}
	if c.isShuttingDown {
		return
	}

	c.connResetLock.Lock()
	defer c.connResetLock.Unlock()

	if c.connectionOk {
		log.Println("Connection problem detected. Re-connecting.")
		c.connectionOk = false

		c.tlsConn.Close()
		for i := 0; true; i++ {
			tlsConn, err := tls.Dial("tcp", c.serverAddr+":"+c.port, c.tlsConf)
			if err == nil {
				c.tlsConn = tlsConn
				c.connectionOk = true
				log.Println("Connection re-established.")
				break
			} else {
				log.Printf("Reconnect failure: %s. Retrying in %d seconds.\n", err.Error(), i*i*5)
				time.Sleep(time.Second * time.Duration(i*i*5))
			}
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
