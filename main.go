package main

import (
	"fmt"
	"log"
	"os"
	"subnet"
)

func main() {
	parseFlags()

	switch modeVar {
	case "client":
		c, err := subnet.NewClient(serverAddressVar, interfaceNameVar, manualClientMode)
		checkErr(err, "subnet.NewClient()")
		c.Run()
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
