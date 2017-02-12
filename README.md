
# subnet

_Simple VPN server/client for the rest of us._

## Use cases

#### Tunnel all non-LAN traffic through another box on the internet (traditional VPN).

Setup the server (linux only):

```shell

git clone https://github.com/twitchyliquid64/subnet
cd subnet
export GOPATH=`pwd`
go build
sysctl net.ipv4.ip_forward=1
iptables -t nat -A POSTROUTING -j MASQUERADE
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365
./subnet -mode server -key key.pem -cert cert.pem -network 192.168.69.1/24 0.0.0.0
```

Setup the client (linux for now, hopefully OSX soon):

```shell
cd subnet
export GOPATH=`pwd`
go build
sudo ./subnet -gw 192.168.69.1 -network 192.168.69.4/24 cnc.ciphersink.net
```

Explanation:
 * subnet is downloaded and compiled on both client and server.
 * Server's networking stack is told to allow the forwarding of packets, and to apply NAT to the packets.
 * Server gets the VPN address `192.168.69.1`, managing traffic for `192.168.69.1` - `192.168.69.255`.
 * Client gets the address `192.168.69.4`.
 * Client remaps its default gateway to `192.168.69.1`, forcing all non-LAN traffic through the VPN server.

WARNING: The above commands setup a self-signed certificate and do not perform client verification. This allows anyone access. I highly recommend creating your own
CA which signs all your certificates, and adding it to both the server & client command lines like `-ca ca.pem`. This will validate both sides are permitted.

#### Make a remote LAN accessible on your machine.

Setup the server (linux only):

```shell

git clone https://github.com/twitchyliquid64/subnet
cd subnet
export GOPATH=`pwd`
go build
sysctl net.ipv4.ip_forward=1
iptables -t nat -A POSTROUTING -j MASQUERADE
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365
./subnet -mode server -key key.pem -cert cert.pem -network 192.168.69.1/24 0.0.0.0
```

Setup the client (linux for now, hopefully OSX soon):

```shell
cd subnet
export GOPATH=`pwd`
go build
sudo ./subnet -network 192.168.69.4/24 cnc.ciphersink.net
```

Explanation:
 * subnet is downloaded and compiled on both client and server.
 * Server gets the VPN address `192.168.69.1`, managing traffic for `192.168.69.1` - `192.168.69.255`.
 * Client gets the address `192.168.69.4`. The `/24` subnet mask means traffic for addresses `192.168.69.1` to `192.168.69.255` will be routed through the VPN.
 * Any traffic to `192.168.69.1` will go to the VPN server. Any traffic to `192.168.69.1` to `192.168.69.255` will go to clients connected to the same server with that address. All other traffic is routed outside of subnet.

WARNING: The above commands setup a self-signed certificate and do not perform client verification. This allows anyone access. I highly recommend creating your own
CA which signs all your certificates, and adding it to both the server & client command lines like `-ca ca.pem`. This will validate both sides are permitted.


## Overview

subnet establishes a TLS connection to the server. A TUN interface is created, and setup with the given network parameters (local IP, and subnet). All traffic that matches the localIP + subnet gets routed to the VPN server.

On the server, all traffic which is recieved is checked against all client's localIP's. If it matches, it goes down there. If it doesn't, it gets routed to the servers TUN device (to its network). If the server's kernel is configured correctly, packets coming back into the TUN device will be NATed so we can work out where to send them. They then get routed back to the correct client.

## TODO

 - [x] Fix server crash when processing packet when the client closes connection
 - [x] Document server setup proceedure, inc forward, masquasde & cert setup
 - [x] Make client resilient to connection failures to the server
 - [ ] Test routing between two clients on the same server.
 - [ ] Fix throughput issues - 5% of normal connection speed. Latency is good though.
 - [ ] Get working on OSX.
