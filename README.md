go-nat-pmp
==========

A Go language client for the NAT-PMP internet protocol for port mapping and discovering the external
IP address of a firewall.

NAT-PMP is supported by Apple brand routers and open source routers like Tomato and DD-WRT.

See https://tools.ietf.org/rfc/rfc6886.txt

NOTE: This is a fork & rewrite of https://github.com/jackpal/go-nat-pmp package to
use newer language features and idioms (and as a learning exercise).

Changes / Updates
---------------

* Update all types to the Go native type (neta.IP, time.Duration, time.Time, etc).
* Using encoding/binary with structs for all request / response messages
* Provide a Transport interface (similar to the caller interface) for logging / testing
* Use an Options pattern for configuring Port and Transport
* Tests use an in-memory fake server for interaction.
* Tests use t.Run() for naming the cases.
* CLI (partly) compatible with natpmpc from [MiniUPnP](http://miniupnp.free.fr/libnatpmp.html).

Get the package
---------------

    # Get the go-natpmp package.
    go get -u github.com/nveeser/go-natpmp

Usage
-----

Get one more package, used by the example code:

    go get -u github.com/nveeser/go-natpmp

Create a directory:

    cd ~/go
    mkdir -p src/hello
    cd src/hello

Create a file hello.go with these contents:

    package main

    import (
        "fmt"

        "github.com/jackpal/gateway"
        natpmp "github.com/nveeser/go-natpmp/natpmp"
    )

    func main() {
        gatewayIP, err := gateway.DiscoverGateway()
        if err != nil {
            return
        }

        client := natpmp.NewClient(gatewayIP)
        extIP, duration, err := client.GetExternalAddress()
        if err != nil {
            return
        }
        fmt.Printf("External IP address: %v\n", extIP)
    }

Build the example

    go build
    ./hello

    External IP address: [www xxx yyy zzz]

License
-------

This project is licensed under the Apache License 2.0.
