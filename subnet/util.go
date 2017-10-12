package subnet

import (
	"errors"
	"math/rand"
	"log"
	"net"
	"os/exec"
	"time"
)

func hostToIP(addr string) (net.IP, error) {
	addrs, err := net.LookupIP(addr)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, errors.New("Could not resolve " + addr)
	}

	rand.Seed(time.Now().Unix())
	return addrs[rand.Int()%len(addrs)], nil
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
