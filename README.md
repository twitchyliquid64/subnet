
# subnet

_Simple VPN server/client for the rest of us._

## Overview

subnet establishes a TLS connection to the server. A TUN interface is created, and setup with the given network parameters (local IP, subnet). All traffic that matches the localIP + subnet gets routed to the VPN server.

On the server, all traffic which is recieved is checked against all client's localIP's. If it matches, it goes down there. If it doesn't, it gets routed to the servers TUN device (to its network). If the server's kernel is configured correctly, packets coming back into the TUN device will be NATed, and hence can be routed correctly. They then get routed back to the correct client.

## Use cases

#### Tunnel all non-LAN traffic through another box on the internet (traditional VPN).

Setup the server:

```shell

git clone https://github.com/twitchyliquid64/subnet
cd subnet
export GOPATH=`pwd`
go build -o subnet *.go
sysctl net.ipv4.ip_forward=1
iptables -t nat -A POSTROUTING -j MASQUERADE
./subnet --mode init-server-certs --cert server.certPEM --key server.keyPEM --ca ca.certPEM --ca_key ca.keyPEM
./subnet --mode server --key server.keyPEM --cert server.certPEM --ca ca.certPEM --network 192.168.69.1/24 0.0.0.0
```

Setup the client:

First, generate a certificate/key pair for each client, by running this on the server:

```shell
./subnet --mode make-client-cert --ca ca.certPEM --ca_key ca.keyPEM client.certPEM client.keyPEM
```

Then, transfer `client.certPEM`, `client.keyPEM` and `ca.certPEM` to your client.

Now, run this on the client:

```shell
cd subnet
export GOPATH=`pwd`
go build -o subnet *.go
sudo ./subnet -gw 192.168.69.1 -network 192.168.69.4/24 -cert client.certPEM -key client.keyPEM -ca ca.certPEM <server address>
```

Explanation:
 * subnet is downloaded and compiled on both client and server.
 * A CA certificate is generated, and a server certificate is generated which is signed by the CA cert (init-server-certs mode).
 * A client certificate is generated, which again is based off the CA cert (make-client-cert mode).
 * Server's networking stack is told to allow the forwarding of packets, and to apply NAT to the packets.
 * Server gets the VPN address `192.168.69.1`, managing traffic for `192.168.69.1` - `192.168.69.255`.
 * Client gets the address `192.168.69.4`.
 * Client remaps its default gateway to `192.168.69.1`, forcing all non-LAN traffic through the VPN server.
 * On connection, both sides verify the TLS cert against the ca cert given on the command line.


#### Make a remote LAN accessible on your machine.

Setup the server (linux only):

```shell

git clone https://github.com/twitchyliquid64/subnet
cd subnet
export GOPATH=`pwd`
go build -o subnet *.go
sysctl net.ipv4.ip_forward=1
iptables -t nat -A POSTROUTING -j MASQUERADE
./subnet --mode init-server-certs --cert server.certPEM --key server.keyPEM --ca ca.certPEM --ca_key ca.keyPEM
./subnet --mode server --key server.keyPEM --cert server.certPEM --ca ca.certPEM --network 192.168.69.1/24 0.0.0.0
```

Setup the client:

First, generate a certificate/key pair for each client, by running this on the server:

```shell
./subnet --mode make-client-cert --ca ca.certPEM --ca_key ca.keyPEM client.certPEM client.keyPEM
```

Then, transfer `client.certPEM`, `client.keyPEM` and `ca.certPEM` to your client.

Now, run this on the client:

```shell
cd subnet
export GOPATH=`pwd`
go build -o subnet *.go
sudo ./subnet -network 192.168.69.4/24 -cert client.certPEM -key client.keyPEM -ca ca.certPEM <server address>
```

Explanation:
 * subnet is downloaded and compiled on both client and server.
 * Certificates are generated, all based on the CA cert which is also generated.
 * Server gets the VPN address `192.168.69.1`, managing traffic for `192.168.69.1` - `192.168.69.255`.
 * Client gets the address `192.168.69.4`. The `/24` subnet mask means traffic for addresses `192.168.69.1` to `192.168.69.255` will be routed through the VPN.
 * Any traffic to `192.168.69.1` will go to the VPN server. Any traffic to `192.168.69.1` to `192.168.69.255` will go to clients connected to the same server with that address. All other traffic is routed outside of subnet.


## Usage

```
Usage of ./subnet:
./subnet <server address>
  -blockProfile
    	Enable block profiling
  -ca string
    	Path to PEM-encoded cert to validate client/serv
  -ca_key string
    	Path to PEM-encoded key to use generating certificates
  -cert string
    	Path to PEM-encoded cert for our side of the connection
  -cpuProfile
    	Enable CPU profiling
  -gw string
    	(Client only) Set the default gateway to this value
  -i string
    	TUN interface, one is picked if not specified
  -key string
    	Path to PEM-encoded key for our cert
  -mode string
    	Whether the process starts a server or as a client (default "client")
  -network string
    	Address for this interface with netmask (default "192.168.69.1/24")
  -port string
    	Port for the VPN connection (default "3234")
```

## TODO

 - [x] Fix server crash when processing packet when the client closes connection
 - [x] Document server setup proceedure, inc forward, masquasde & cert setup
 - [x] Make client resilient to connection failures to the server
 - [ ] Test routing between two clients on the same server.
 - [x] Fix throughput issues - 5% of normal connection speed. Latency is good though.
 - [x] Get working on OSX.
