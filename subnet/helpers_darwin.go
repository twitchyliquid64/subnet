package subnet

import (
  "errors"
  "fmt"
  "net"
  "os/exec"
  "regexp"
  "strings"
)

//SetInterfaceStatus brings up or down a network interface.
func SetInterfaceStatus(iName string, up bool, debug bool) error {
  statusString := "down"
	if up {
		statusString = "up"
	}

  //TODO: Support setting the QLEN
	sargs := fmt.Sprintf("%s %s mtu %d", iName, statusString, devMtuSize)
	return commandExec("ifconfig", strings.Split(sargs, " "), debug)}

//SetDevIP sets the local IP address of a network interface.
func SetDevIP(iName string, localAddr net.IP, addr *net.IPNet, debug bool) error {
  sargs := fmt.Sprintf("set %s MANUAL %s 0x%s", iName, localAddr.String(), addr.Mask)
	return commandExec("ipconfig", strings.Split(sargs, " "), debug)
}

// SetDefaultGateway sets the systems gateway to the IP / device specified.
func SetDefaultGateway(gw, iName string, debug bool) error {
  sargs := fmt.Sprintf("-n change default -interface %s", iName)
	args := strings.Split(sargs, " ")
	return commandExec("route", args, debug)
}

// AddRoute routes all traffic for addr via interface iName.
func AddRoute(addr, viaAddr net.IP, iName string, debug bool) error {
  sargs := fmt.Sprintf("-n add %s %s -ifscope %s", addr.String(), viaAddr.String(), iName)
	args := strings.Split(sargs, " ")
	return commandExec("route", args, debug)
}

// DelRoute deletes the route in the system routing table to a specific destination.
func DelRoute(addr, viaAddr net.IP, iName string, debug bool) error {
  sargs := fmt.Sprintf("-n delete %s %s -ifscope %s", addr.String(), viaAddr.String(), iName)
	args := strings.Split(sargs, " ")
	return commandExec("route", args, debug)
}

var parseRouteGetRegex = regexp.MustCompile(`(?m)^\W*([^\:]+):\W(.*)$`)

// GetNetGateway return net gateway (default route) and nic.
func GetNetGateway() (gw, dev string, err error) {
  cmd := exec.Command("route", "-n", "get", "default")
	output, e := cmd.Output()
	if e != nil {
		return "", "", e
	}

  matches := parseRouteGetRegex.FindAllSubmatch(output, -1)
  defaultRouteInfo := map[string]string{}
  for _, match := range matches {
    defaultRouteInfo[string(match[1])] = string(match[2])
  }

  _, gatewayExists := defaultRouteInfo["gateway"]
  _, interfaceExists := defaultRouteInfo["interface"]
  if !gatewayExists || !interfaceExists {
    return "", "", errors.New("internal error: could not read gateway or interface")
  }

  return defaultRouteInfo["gateway"], defaultRouteInfo["interface"], nil
}
