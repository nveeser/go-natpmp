// Package natpmp implements the NAT-PMP protocol, typically supported by Apple
// routers and open source routers such as DD-WRT and Tomato.
//
// See https://tools.ietf.org/rfc/rfc6886.txt
//
// Usage:
//
//	client := natpmp.NewClient(gatewayIP)
//	response, err := client.GetExternalAddress()
package natpmp

import (
	"fmt"
	"net"
	"net/netip"
	"time"
)

// Client is a NAT-PMP protocol client.
type Client struct {
	gatewayIP net.IP
	port      int
	timeout   time.Duration
	transport Transport
}

// NewClient create a NAT-PMP client for the NAT-PMP server at the gateway.
// Uses default timeout which is around 128 seconds.
func NewClient(gatewayIP net.IP, opts ...Option) (nat *Client) {
	c := &Client{
		gatewayIP: gatewayIP,
		port:      defaultPort,
		transport: DefaultTransport(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// GetExternalAddress returns the external address of the router.
// Note that this call can take up to 128 seconds to return.
func (c *Client) GetExternalAddress() (addr netip.Addr, duration time.Duration, err error) {
	var resp extAddrResp
	if err := c.rpc(&extAddrReq{0, 0}, &resp); err != nil {
		return netip.Addr{}, 0, fmt.Errorf("ExternalAddress Failed: %w", err)
	}
	return netip.AddrFrom4(resp.IPAddr), time.Duration(resp.DurationSecs) * time.Second, nil
}

type extAddrReq struct {
	Version byte
	Opcode  byte
}

func (r extAddrReq) version() int { return int(r.Version) }
func (r extAddrReq) opcode() byte { return r.Opcode }

type extAddrResp struct {
	Version      byte
	Opcode       byte
	ResultCode   uint16
	DurationSecs uint32
	IPAddr       [4]byte
}

func (r extAddrResp) version() int    { return int(r.Version) }
func (r extAddrResp) opcode() byte    { return r.Opcode }
func (r extAddrResp) resultCode() int { return int(r.ResultCode) }

// PortMapping holds the result of calling AddPortMapping.
type PortMapping struct {
	// aka SecondsSinceStartOfEpoc
	EpochDuration      time.Duration
	InternalPort       uint16
	MappedExternalPort uint16
	// aka PortMappingLifetimeInSeconds
	Lifetime time.Duration
}

// AddPortMapping Adds (or deletes) a port mapping. To delete a mapping, set the requestedExternalPort and lifetime to 0.
// Note that this call can take up to 128 seconds to return.
func (c *Client) AddPortMapping(protocol string, internalPort, requestedExternalPort int, lifetime time.Duration) (result *PortMapping, err error) {
	var opcode byte
	switch protocol {
	case "udp":
		opcode = 1
	case "tcp":
		opcode = 2
	default:
		return nil, fmt.Errorf("unknown protocol %v", protocol)
	}
	req := mappingReq{
		Version:       0,
		Opcode:        opcode,
		InternalPort:  uint16(internalPort),
		RequestedPort: uint16(requestedExternalPort),
		LifetimeSecs:  uint32(lifetime.Seconds()),
	}
	var resp mappingResp
	if err := c.rpc(&req, &resp); err != nil {
		return nil, fmt.Errorf("ExternalAddress Failed: %w", err)
	}
	return &PortMapping{
		EpochDuration:      time.Duration(resp.DurationSecs) * time.Second,
		InternalPort:       resp.InternalPort,
		MappedExternalPort: resp.MappedPort,
		Lifetime:           time.Duration(resp.LifetimeSecs) * time.Second,
	}, nil
}

type mappingReq struct {
	Version       byte
	Opcode        byte
	_             int16 // reserved
	InternalPort  uint16
	RequestedPort uint16
	LifetimeSecs  uint32
}

func (r mappingReq) version() int { return int(r.Version) }
func (r mappingReq) opcode() byte { return r.Opcode }

type mappingResp struct {
	Version      byte
	Opcode       byte
	ResultCode   uint16
	DurationSecs uint32
	InternalPort uint16
	MappedPort   uint16
	LifetimeSecs uint32
}

func (r mappingResp) version() int    { return int(r.Version) }
func (r mappingResp) opcode() byte    { return r.Opcode }
func (r mappingResp) resultCode() int { return int(r.ResultCode) }
