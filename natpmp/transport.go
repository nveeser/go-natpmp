package natpmp

import (
	"fmt"
	"net"
	"time"
)

// Transport is the interface for opening a connection and
// sending and receiving data with the NAT-PMP gateway.
type Transport interface {
	Open(gateway net.IP, port int) error
	Close() error
	Send(req, resp []byte, deadline time.Time) (result []byte, remoteIP net.IP, err error)
}

// DefaultTransport returns the default transport
// which uses UDP to send / receive bytes from the gateway.
func DefaultTransport() Transport {
	return &udpTransport{}
}

type udpTransport struct {
	gatewayAddr *net.UDPAddr
	conn        *net.UDPConn
}

func (c *udpTransport) Open(gateway net.IP, port int) error {
	var err error
	c.gatewayAddr = &net.UDPAddr{
		IP:   gateway,
		Port: port,
	}
	c.conn, err = net.DialUDP("udp", nil, c.gatewayAddr)
	return err
}

func (c *udpTransport) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *udpTransport) Send(req, resp []byte, deadline time.Time) ([]byte, net.IP, error) {
	if err := c.conn.SetDeadline(deadline); err != nil {
		return nil, nil, fmt.Errorf("SetDeadline(): %w", err)
	}
	_, err := c.conn.Write(req)
	if err != nil {
		return nil, nil, fmt.Errorf("Write(): %w", err)
	}
	n, remoteAddr, err := c.conn.ReadFromUDP(resp)
	if err != nil {
		return nil, nil, fmt.Errorf("ReadFromUDP(): %w", err)
	}
	// Trim result to actual number of bytes received
	if n < len(resp) {
		resp = resp[:n]
	}
	return resp, remoteAddr.IP, nil
}
