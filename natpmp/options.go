package natpmp

import (
	"time"
)

// Option is the type for modifying the Client
type Option func(*Client)

// Timeout returns an option which sets the default timeout
// for the client.
func Timeout(timeout time.Duration) Option {
	return func(client *Client) {
		client.timeout = timeout
	}
}

// Port returns an option which sets the port to use on the Gateway for NAT-PMP.
func Port(port int) Option {
	return func(client *Client) {
		if port != 0 {
			client.port = port
		}
	}
}

// WithTransport returns an option which uses the specified Transport for
// sending / receiving bytes to the endpoint. Primarily for logging / testing.
func WithTransport(transport Transport) Option {
	return func(client *Client) {
		client.transport = transport
	}
}
