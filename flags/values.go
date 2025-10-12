package flags

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// IPValue is a net.IP which implements the flag.Value interface
type IPValue net.IP

func (v *IPValue) String() string { return (*net.IP)(v).String() }
func (v *IPValue) IsSet() bool    { return *v == nil }
func (v *IPValue) Set(s string) error {
	x := net.ParseIP(s)
	if v == nil {
		return fmt.Errorf("invalid gateway IP: %s", s)
	}
	*v = IPValue(x)
	return nil
}

type PortSpec struct {
	ExtPort  int
	IntPort  int
	Protocol string
	Lifetime time.Duration
}

func (p *PortSpec) IsSet() bool {
	return p.IntPort > 0 && p.ExtPort > 0 && p.Protocol != ""
}

func (p *PortSpec) String() string {
	return fmt.Sprintf("%d %d %s %s", p.ExtPort, p.IntPort, p.Protocol, p.Lifetime.String())
}

func (p *PortSpec) Set(s string) error {
	d, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	p.ExtPort = d
	if p.ExtPort <= 0 {
		return fmt.Errorf("invalid ext port: %d", p.ExtPort)
	}
	return nil
}

func (p *PortSpec) consume(args []string) ([]string, error) {
	// Only consume args when the ExpPort has been set
	// and the other values have not been set.
	if p.ExtPort == 0 || p.IsSet() {
		return args, nil
	}
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return args, fmt.Errorf("missing public port")
	}
	d, err := strconv.Atoi(args[0])
	if err != nil {
		return args, fmt.Errorf("invalid private port: %s", args[0])
	}
	p.IntPort = d
	if len(args) < 2 || strings.HasPrefix(args[1], "-") {
		return args, fmt.Errorf("missing protocol")
	}
	p.Protocol = args[1]
	if len(args) < 3 || strings.HasPrefix(args[2], "-") {
		return args[2:], nil
	}
	d, err = strconv.Atoi(args[2])
	if err != nil {
		return args, fmt.Errorf("invalid Lifetime: %s", args[2])
	}
	p.Lifetime = time.Duration(d) * time.Second
	return args[3:], nil
}
