package subnet

const (
	//Queue from TUN -> router(server) / remote end (client)
	pktInMaxBuff = 150
	//Queue to TUN
	pktOutMaxBuff = 150

	devMtuSize     = 1500
	devPktBuffSize = 4096
	devTxQueLen    = 300

	//Queue from network clients to ingestion
	servMaxInboundPktQueue = 400
	//Queue out to each network client
	servPerClientPktQueue = 200
)
