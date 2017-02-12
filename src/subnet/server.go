package subnet

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"subnet/conn"
	"sync"
	"time"

	"github.com/songgao/water"
)

//Server represents a service providing a VPN service to subnet clients.
type Server struct {
	manual bool

	tlsConf        *tls.Config
	tlsListener    net.Listener
	localAddr      net.IP
	localNetMask   *net.IPNet
	isShuttingDown bool

	clientIDByAddress map[string]int
	clients           map[int]*serverConn
	clientsLock       sync.Mutex
	lastClientID      int

	inboundIPPkts   chan *inboundIPPkt
	outboundDevPkts chan *IPPacket

	intf     *water.Interface
	reverser Reverser
	wg       sync.WaitGroup
}

// NewServer returns a new server object representing a VPN service.
func NewServer(servHost, port, network, iName string, manual bool,
	certPemPath, keyPemPath, caCertPath string) (*Server, error) {
	tlsConf, err := conn.TLSConfig(certPemPath, keyPemPath, caCertPath)
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

	s := &Server{
		intf:              intf,
		localAddr:         netIP,
		localNetMask:      localNetMask,
		manual:            manual,
		tlsConf:           tlsConf,
		inboundIPPkts:     make(chan *inboundIPPkt, servMaxInboundPktQueue),
		outboundDevPkts:   make(chan *IPPacket, 2),
		clientIDByAddress: map[string]int{},
		clients:           map[int]*serverConn{},
	}

	return s, s.Init(servHost + ":" + port)
}

// Init sets up the server.
func (s *Server) Init(servHost string) (err error) {
	s.tlsListener, err = tls.Listen("tcp", servHost, s.tlsConf)
	if err != nil {
		return err
	}
	if err = SetDevIP(s.intf.Name(), s.localAddr, s.localNetMask, false); err != nil {
		return err
	}
	log.Printf("IP of %s set to %s, localNetMask %s\n", s.intf.Name(), s.localAddr.String(), net.IP(s.localNetMask.Mask).String())

	return err
}

// Run starts the server
func (s *Server) Run() {
	go s.acceptRoutine()
	go s.dispatchRoutine()
	go devWriteRoutine(s.intf, s.outboundDevPkts, &s.wg, &s.isShuttingDown)
}

func (s *Server) acceptRoutine() {
	var tcpListener *net.TCPListener
	tcpListener, _ = s.tlsListener.(*net.TCPListener)
	s.wg.Add(1)
	defer s.wg.Done()

	for !s.isShuttingDown {
		if tcpListener != nil {
			tcpListener.SetDeadline(time.Now().Add(time.Millisecond * 300))
		}
		conn, err := s.tlsListener.Accept()
		if err != nil {
			if !s.isShuttingDown {
				log.Printf("Listener err: %s\n", err.Error())
			}
			return
		}
		s.handleClient(conn)
	}
}

// routing from inboundIPPkts to client/TUN.
func (s *Server) dispatchRoutine() {
	for !s.isShuttingDown {
		pkt := <-s.inboundIPPkts

		s.clientsLock.Lock()
		destClientID, canRouteDirectly := s.clientIDByAddress[pkt.pkt.Dest.String()]
		if canRouteDirectly {
			destClient := s.clients[destClientID]
			destClient.queueIP(pkt)
		}
		s.clientsLock.Unlock()
		if !canRouteDirectly {
			s.outboundDevPkts <- pkt.pkt
		}
	}
}

func (s *Server) handleClient(conn net.Conn) {
	c := serverConn{
		conn:      conn,
		canSendIP: true,
	}
	s.enrollClientConn(&c)
	c.initClient(s)
}

func (s *Server) enrollClientConn(c *serverConn) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()
	c.id = s.lastClientID
	s.lastClientID++
	s.clients[c.id] = c
}

func (s *Server) setAddrForClient(id int, addr net.IP) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	s.clientIDByAddress[addr.String()] = id
}

func (s *Server) removeClientConn(id int) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	delete(s.clients, id)
}

// Close shuts down the server, reversing configuration changes to the system.
func (s *Server) Close() error {
	s.isShuttingDown = true
	s.reverser.Close()

	err := s.tlsListener.Close()
	if err != nil {
		return err
	}
	err = s.intf.Close()
	if err != nil {
		return err
	}

	//s.wg.Wait() //who cares
	return nil
}
