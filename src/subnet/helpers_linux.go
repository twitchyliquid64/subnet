package subnet

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
)

//SetInterfaceStatus brings up or down a network interface.
func SetInterfaceStatus(iName string, up bool, debug bool) error {
	statusString := "down"
	if up {
		statusString = "up"
	}
	sargs := fmt.Sprintf("link set dev %s %s qlen 100", iName, statusString)
	args := strings.Split(sargs, " ")
	return commandExec("ip", args, debug)
}

//SetDevIP sets the local IP address of a network interface.
func SetDevIP(iName string, ip net.IP, maskBits int) error {
	sargs := fmt.Sprintf("%s %s netmask %s", iName, ip.String())
	args := strings.Split(sargs, " ")
	return nil
}

func commandExec(command string, args []string, debug bool) error {
	cmd := exec.Command(command, args...)
	if debug {
		log.Println("exec "+command+": ", args)
	}
	e := cmd.Run()
	if e != nil {
		log.Println("Command failed: ", e)
	}
	return e
}
