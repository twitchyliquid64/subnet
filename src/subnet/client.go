package subnet

import (
	"errors"
	"log"

	"github.com/songgao/water"
)

// Client represents a connection to a subnet server.
type Client struct {
	addr   string
	iName  string
	manual bool

	intf *water.Interface
}

// NewClient constructs a Client object.
func NewClient(addr, iName string, manual bool) (*Client, error) {
	intf, err := water.NewTUN(iName)
	if err != nil {
		return nil, errors.New("Could not create TUN - " + err.Error())
	}

	log.Printf("Created interface with name %s\n", intf.Name())

	ret := &Client{
		intf:   intf,
		addr:   addr,
		manual: manual,
		iName:  intf.Name(),
	}

	return ret, ret.init()
}

// Initializes connection and changes network configuration as needed, but does not
// activate the client object for use.
func (c *Client) init() error {

	if !c.manual { //setup routes on the system

	}

	return nil
}

// Run starts the client.
func (c *Client) Run() {

}
