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
	for !*isShuttingDown {
		n, err := dev.Read(packet)
		if err != nil {
			if !*isShuttingDown {
				log.Printf("%s read err: %s\n", dev.Name(), err.Error())
			}
			close(packetsIn)
			return
		}
		p := &IPPacket{
			Raw:      packet[:n],
			Dest:     waterutil.IPv4Destination(packet[:n]),
			Protocol: waterutil.IPv4Protocol(packet[:n]),
		}
		packetsIn <- p
		log.Printf("Packet Received: dest %s, len %d\n", p.Dest.String(), len(p.Raw))
	}
}

func devWriteRoutine(dev *water.Interface, packetsOut chan *IPPacket, wg *sync.WaitGroup, isShuttingDown *bool) {
	wg.Add(1)
	defer wg.Done()

	for !*isShuttingDown {
		pkt := <-packetsOut
		_, err := dev.Write(pkt.Raw)
		if err != nil {
			log.Printf("Write to %s failed: %s\n", dev.Name(), err.Error())
			return
		}
	}
}
