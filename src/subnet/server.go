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
		intf:         intf,
		localAddr:    netIP,
		localNetMask: localNetMask,
		manual:       manual,
		tlsConf:      tlsConf,
	}

	return s, s.Init(servHost + ":" + port)
}

// Init sets up the server.
func (s *Server) Init(servHost string) (err error) {
	s.tlsListener, err = tls.Listen("tcp", servHost, s.tlsConf)
	return err
}

// Run starts the server
func (s *Server) Run() {
	go s.acceptRoutine()
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

func (s *Server) handleClient(conn net.Conn) {

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

	s.wg.Wait()
	return nil
}
