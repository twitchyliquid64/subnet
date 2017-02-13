package subnet

const (
	//Queue from TUN -> router(server) / remote end (client)
	pktInMaxBuff  = 150
	pktOutMaxBuff = 150

	devMtuSize     = 1500
	devPktBuffSize = 4096

	//Queue from network clients to ingestion
	servMaxInboundPktQueue = 80
	//Queue out to each network client
	servPerClientPktQueue = 40
)
