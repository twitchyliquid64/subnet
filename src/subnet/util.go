package subnet

import (
	"errors"
	"math/rand"
	"net"
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
