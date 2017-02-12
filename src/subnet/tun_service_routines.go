package subnet

import (
	"log"
	"sync"

	"github.com/songgao/water"
	"github.com/songgao/water/waterutil"
)

func devReadRoutine(dev *water.Interface, packetsIn chan *IPPacket, wg *sync.WaitGroup, isShuttingDown *bool) {
	wg.Add(1)
	defer wg.Done()

	packet := make([]byte, devPktBuffSize)
	for {
		n, err := dev.Read(packet)
		if err != nil {
			if !*isShuttingDown {
				log.Printf("%s read err: %s\n", dev.Name(), err.Error())
			}
			close(packetsIn)
			return
		}
		p := &IPPacket{
			raw:      packet[:n],
			dest:     waterutil.IPv4Destination(packet[:n]),
			protocol: waterutil.IPv4Protocol(packet[:n]),
		}
		packetsIn <- p
		log.Printf("Packet Received: dest %s, len %d\n", p.dest.String(), len(p.raw))
	}
}
