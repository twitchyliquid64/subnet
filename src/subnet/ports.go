package subnet

import "sync"

const portPoolStart = 10200

var issuedPorts = 0
var freedPorts = []int{}
var freePortLock sync.Mutex

func getPort() int {
	freePortLock.Lock()
	defer freePortLock.Unlock()
	if len(freedPorts) > 0 {
		var x int
		x, freedPorts = freedPorts[len(freedPorts)-1], freedPorts[:len(freedPorts)-1]
		return x
	}

	issuedPorts++
	return portPoolStart + issuedPorts
}

func freePort(port int) {
	freePortLock.Lock()
	defer freePortLock.Unlock()

	freedPorts = append(freedPorts, port)
}
