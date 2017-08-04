package subnet

import (
	"log"
	"net"

	"github.com/songgao/water"
)

// Reverser contains a sequence of functions that need to be called on exit -
// to unwind changes made to global configuration.
type Reverser struct {
	RouteDeletions []routeEntries

	updateGateway bool
	newGW         string

	interfaceToClose *water.Interface
}

type routeEntries struct {
	dest net.IP
	via  net.IP
	dev  string
}

// AddRouteEntry adds a route to the deletion set so it is deleted from the
// routing table when Reverse() is called.
func (r *Reverser) AddRouteEntry(destination net.IP, via net.IP, dev string) {
	r.RouteDeletions = append(r.RouteDeletions, routeEntries{
		dest: destination,
		via:  via,
		dev:  dev,
	})
}

// ResetGatewayOSX tells the reverser what gateway should be set on exit.
func (r *Reverser) ResetGatewayOSX(intf *water.Interface, gw string) {
	r.updateGateway = true
	r.newGW = gw
	r.interfaceToClose = intf
}

// Close applies the changes specified in reverser, such to reverse changes
// to system configuration.
func (r *Reverser) Close() {
	for _, route := range r.RouteDeletions {
		e := DelRoute(route.dest, route.via, route.dev, true)
		if e == nil {
			log.Printf("Deleted route to %s via %s on %s\n", route.dest.String(), route.via.String(), route.dev)
		} else {
			log.Printf("Error: Route delete %s (%s on %s) - %s\n", route.dest.String(), route.via.String(), route.dev, e.Error())
		}
	}
	if r.updateGateway {
		r.interfaceToClose.Close()
		commandExec("route", []string{"add", "default", r.newGW}, false)
	}
}
