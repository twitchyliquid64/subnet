package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/twitchyliquid64/subnet/subnet"
	"github.com/twitchyliquid64/subnet/subnet/cert"
)

func main() {
	parseFlags()
	fatalErrChan := make(chan error)

	if crlPathVar != "" && (modeVar == "client" || modeVar == "server") {
		crlStartErr := cert.InitCRL(crlPathVar)
		checkErr(crlStartErr, "init-crl")
	}

	switch modeVar {
	case "client":
		var additionalAddrs []net.IP
		for _, addrStr := range strings.Split(additionalClientAddrs, ",") {
			if addrStr != "" {
				additionalAddrs = append(additionalAddrs, net.ParseIP(addrStr))
			}
		}
		c, err := subnet.NewClient(serverAddressVar, connPortVar, networkAddrVar, interfaceNameVar, gatewayVar, ourCertPathVar, ourKeyPathVar, caCertPathVar, additionalAddrs)
		checkErr(err, "subnet.NewClient()")
		c.Run()
		defer func() { checkErr(c.Close(), "client.Close()") }()
		waitInterrupt(fatalErrChan)

	case "server":
		s, err := subnet.NewServer(serverAddressVar, connPortVar, networkAddrVar, interfaceNameVar, ourCertPathVar, ourKeyPathVar, caCertPathVar)
		checkErr(err, "subnet.NewServer()")
		s.Run()
		defer func() { checkErr(s.Close(), "server.Close()") }()
		waitInterrupt(fatalErrChan)

	case "init-server-certs":
		err := cert.MakeServerCert(ourCertPathVar, ourKeyPathVar, caCertPathVar, caKeyPathVar)
		checkErr(err, "init-server-certs")
		fmt.Println("NOTICE: Certificates expire (and will need to be rotated) one year from now.")

	case "make-client-cert":
		err := cert.IssueClientCert(caCertPathVar, caKeyPathVar, flag.Arg(0), flag.Arg(1))
		fmt.Println("NOTICE: Certificates expire (and will need to be rotated) one year from now.")
		checkErr(err, "make-client-cert")

	case "blacklist-cert":
		err := cert.AddToCRL(crlPathVar, flag.Arg(0), flag.Arg(1))
		checkErr(err, "blacklist-cert")

	default:
		fmt.Fprintf(os.Stderr, "Err: Unrecognised mode. Mode must be either client/server.\n")
		os.Exit(3)
	}
}

func checkErr(err error, component string) {
	if err != nil {
		log.Printf("%s err: %s", component, err.Error())
		os.Exit(1)
	}
}

func waitInterrupt(fatalErrChan chan error) {
	sig := make(chan os.Signal, 2)
	done := make(chan bool, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		done <- true
	}()

	select {
	case <-done:
		log.Println("Recieved interrupt, shutting down.")
	case err := <-fatalErrChan:
		log.Println("Fatal internal error: ", err)
	}
}
