package natpmp

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"time"
)

// Implement the NAT-PMP protocol, typically supported by Apple routers and open source
// routers such as DD-WRT and Tomato.
//
// See https://tools.ietf.org/rfc/rfc6886.txt
//
// Usage:
//
//    client := natpmp.NewClient(gatewayIP)
//    response, err := client.GetExternalAddress()

// The recommended mapping lifetime for AddPortMapping.
const recommendedMappingLifetime = 3600 * time.Second

// Interface used to make remote procedure calls.
type caller interface {
	call(msg []byte, timeout time.Duration) (result []byte, err error)
}

type Option func(*Client)

func WithTimeout(timeout time.Duration) Option {
	return func(client *Client) {
		client.timeout = timeout
	}
}

// Client is a NAT-PMP protocol client.
type Client struct {
	caller  caller
	timeout time.Duration
}

// Create a NAT-PMP client for the NAT-PMP server at the gateway.
// Uses default timeout which is around 128 seconds.
func NewClient(gateway net.IP, opts ...Option) (nat *Client) {
	return &Client{&network{gateway}, 0}
}

// GetExternalAddress returns the external address of the router.
// Note that this call can take up to 128 seconds to return.
func (n *Client) GetExternalAddress() (addr netip.Addr, duration time.Duration, err error) {
	msg := make([]byte, 2)
	msg[0] = 0 // Version 0
	msg[1] = 0 // OP Code 0
	response, err := n.rpc(msg, 12)
	if err != nil {
		return
	}
	addr = netip.AddrFrom4([4]byte(response[8:12]))
	duration = time.Duration(binary.BigEndian.Uint32(response[4:8])) * time.Second
	return addr, duration, nil
}

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
func (n *Client) AddPortMapping(protocol string, internalPort, requestedExternalPort int, lifetime int) (result *PortMapping, err error) {
	var opcode byte
	if protocol == "udp" {
		opcode = 1
	} else if protocol == "tcp" {
		opcode = 2
	} else {
		err = fmt.Errorf("unknown protocol %v", protocol)
		return
	}
	msg := make([]byte, 12)
	msg[0] = 0 // Version 0
	msg[1] = opcode
	// [2:3] is reserved.
	binary.BigEndian.PutUint16(msg[4:6], uint16(internalPort))
	binary.BigEndian.PutUint16(msg[6:8], uint16(requestedExternalPort))
	binary.BigEndian.PutUint32(msg[8:12], uint32(lifetime))
	response, err := n.rpc(msg, 16)
	if err != nil {
		return
	}
	result = &PortMapping{
		EpochDuration:      time.Duration(binary.BigEndian.Uint32(response[4:8])) * time.Second,
		InternalPort:       binary.BigEndian.Uint16(response[8:10]),
		MappedExternalPort: binary.BigEndian.Uint16(response[10:12]),
		Lifetime:           time.Duration(binary.BigEndian.Uint32(response[12:16])) * time.Second,
	}
	return
}

func (n *Client) rpc(msg []byte, resultSize int) (result []byte, err error) {
	result, err = n.caller.call(msg, n.timeout)
	if err != nil {
		return
	}
	err = protocolChecks(msg, resultSize, result)
	return
}

func protocolChecks(msg []byte, n int, result []byte) (err error) {
	if len(result) != n {
		return fmt.Errorf("unexpected result size %d, expected %d", len(result), n)
	}
	if result[0] != 0 {
		return fmt.Errorf("unknown protocol version %d", result[0])
	}
	expectedOp := msg[1] | 0x80
	if result[1] != expectedOp {
		return fmt.Errorf("Unexpected opcode %d. Expected %d", result[1], expectedOp)
	}
	resultCode := binary.BigEndian.Uint16(result[2:4])
	if resultCode != 0 {
		return fmt.Errorf("Non-zero result code %d", resultCode)
	}
	return nil
}
