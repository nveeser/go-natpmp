package main

import (
	"flag"
	"fmt"
	"github.com/jackpal/gateway"
	"github.com/nveeser/go-natpmp/flags"
	"github.com/nveeser/go-natpmp/natpmp"
	"log"
	"net"
	"os"
)

func main() {
	var cfg flags.Config
	if err := cfg.ParseArgs(flag.CommandLine, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
	if cfg.Help {
		flag.Usage()
		return
	}

	gwIP, err := findGatewayIP(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	client := natpmp.NewClient(gwIP, natpmp.Port(cfg.Port))
	if cfg.AddSpec.IsSet() {
		fmt.Printf("Port: %s %+v\n", &cfg.AddSpec, os.Args[1:])
		spec := cfg.AddSpec
		mapping, err := client.AddPortMapping(spec.Protocol, spec.IntPort, spec.ExtPort, spec.Lifetime)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("RemotePort: %d (%s)\n", mapping.MappedExternalPort, mapping.Lifetime)
		return
	}
	ea, duration, err := client.GetExternalAddress()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("External Address: %s %s\n", ea, duration)
}

func findGatewayIP(c *flags.Config) (gwIP net.IP, err error) {
	if c.Gateway != nil {
		return net.IP(c.Gateway), nil
	}
	return gateway.DiscoverGateway()
}
