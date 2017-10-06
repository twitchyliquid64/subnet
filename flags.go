package main

import (
	"flag"
	"fmt"
	"os"
)

var interfaceNameVar string
var networkAddrVar string

var caCertPathVar string
var caKeyPathVar string
var ourCertPathVar string
var ourKeyPathVar string
var serverAddressVar string
var connPortVar string

var modeVar string
var gatewayVar string

var crlPathVar string

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "%s <server address>\n", os.Args[0])
	flag.PrintDefaults()
}

func parseFlags() {
	flag.StringVar(&interfaceNameVar, "i", "", "TUN interface, one is picked if not specified")
	flag.StringVar(&caCertPathVar, "ca", "", "Path to PEM-encoded cert to validate client/serv")
	flag.StringVar(&caKeyPathVar, "ca_key", "", "Path to PEM-encoded key to use generating certificates")
	flag.StringVar(&ourCertPathVar, "cert", "", "Path to PEM-encoded cert for our side of the connection")
	flag.StringVar(&ourKeyPathVar, "key", "", "Path to PEM-encoded key for our cert")
	flag.StringVar(&connPortVar, "port", "3234", "Port for the VPN connection")
	flag.StringVar(&modeVar, "mode", "client", "Whether the process starts a server or as a client")
	flag.StringVar(&networkAddrVar, "network", "192.168.69.1/24", "Address for this interface with netmask")
	flag.StringVar(&gatewayVar, "gw", "", "(Client only) Set the default gateway to this value")
	flag.StringVar(&crlPathVar, "crl", "", "Optional path to JSON-CRL file")

	flag.Usage = printUsage
	flag.Parse()

	if modeVar != "init-server-certs" && modeVar != "make-client-cert" && modeVar != "blacklist-cert" && flag.NArg() != 1 {
		printUsage()
		os.Exit(2)
	}

	if modeVar == "server" {
		if ourCertPathVar == "" || ourKeyPathVar == "" {
			fmt.Fprintf(os.Stderr, "Err: Certificate and key must be specified for server mode.\n")
			flag.PrintDefaults()
			os.Exit(2)
		}
	}

	if modeVar == "init-server-certs" {
		if ourCertPathVar == "" || ourKeyPathVar == "" {
			fmt.Fprintf(os.Stderr, "Err: Certificate and key path must be specified for generating certs.\n")
			flag.PrintDefaults()
			os.Exit(2)
		}
		if caCertPathVar == "" || caKeyPathVar == "" {
			fmt.Fprintf(os.Stderr, "Err: CA Certificate and key path must be specified for generating certs.\n")
			flag.PrintDefaults()
			os.Exit(2)
		}
	}

	if modeVar == "make-client-cert" {
		if caCertPathVar == "" || caKeyPathVar == "" {
			fmt.Fprintf(os.Stderr, "Err: CA Certificate and key path must be specified for generating certs.\n")
			flag.PrintDefaults()
			os.Exit(2)
		}
		if flag.NArg() != 2 {
			fmt.Fprintf(os.Stderr, "Err: Expected 2 arguments. EG: ./subnet OPTIONS certPath keyPath\n")
			os.Exit(2)
		}
	}

	if modeVar == "blacklist-cert" {
		if crlPathVar == "" {
			fmt.Fprintf(os.Stderr, "Err: CRL path must be specified.\n")
			flag.PrintDefaults()
			os.Exit(2)
		}
		if flag.NArg() != 2 {
			fmt.Fprintf(os.Stderr, "Err: Expected 2 arguments. EG: ./subnet -crl <crlPath> -mode blacklist-cert <certPath> \"justification\"\n")
			os.Exit(2)
		}
	}

	serverAddressVar = flag.Arg(0)
}
