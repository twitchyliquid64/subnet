package main

import (
	"flag"
	"fmt"
	"os"
)

var interfaceNameVar string
var caCertPathVar string
var serverAddressVar string
var modeVar string
var manualClientMode bool

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "%s [-i <interface>] [-ca <certificate path>] -mode client/server <server address>\n", os.Args[0])
	flag.PrintDefaults()
}

func parseFlags() {
	flag.StringVar(&interfaceNameVar, "i", "", "TUN interface")
	flag.StringVar(&caCertPathVar, "ca", "", "Path to PEM-encoded certificate used to validate server")
	flag.StringVar(&modeVar, "mode", "client", "Sets whether the process starts a server or as a client")
	flag.BoolVar(&manualClientMode, "manual", false, "Prevents subnet from changing config to route default traffic through it")

	flag.Usage = printUsage
	flag.Parse()

	if flag.NArg() != 1 {
		printUsage()
		os.Exit(2)
	}
}
