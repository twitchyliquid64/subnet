package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"subnet"
	"syscall"
)

func main() {
	parseFlags()
	fatalErrChan := make(chan error)

	switch modeVar {
	case "client":
		c, err := subnet.NewClient(serverAddressVar, connPortVar, networkAddrVar, interfaceNameVar, manualClientMode, ourCertPathVar, ourKeyPathVar, caCertPathVar)
		checkErr(err, "subnet.NewClient()")
		c.Run()
		defer func() { checkErr(c.Close(), "client.Close()") }()
		waitInterrupt(fatalErrChan)

	case "server":
		s, err := subnet.NewServer(serverAddressVar, connPortVar, networkAddrVar, interfaceNameVar, manualClientMode, ourCertPathVar, ourKeyPathVar, caCertPathVar)
		checkErr(err, "subnet.NewServer()")
		s.Run()
		defer func() { checkErr(s.Close(), "server.Close()") }()
		waitInterrupt(fatalErrChan)

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
