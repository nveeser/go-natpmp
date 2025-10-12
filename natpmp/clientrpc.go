package natpmp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"time"
)

const defaultPort = 5351
const maxRetries = 9
const initialPause = 250 * time.Millisecond

type request interface {
	version() int
	opcode() byte
}
type response interface {
	request
	resultCode() int
}

func (c *Client) rpc(req request, resp response) error {
	if err := c.transport.Open(c.gatewayIP, c.port); err != nil {
		return fmt.Errorf("error net.DialUDP(): %w", err)
	}
	defer c.transport.Close()

	var reqBuf bytes.Buffer
	if err := binary.Write(&reqBuf, binary.BigEndian, req); err != nil {
		return fmt.Errorf("error Write(%T) request: %w", req, err)
	}

	retry := &retry{
		initial:    initialPause,
		maxRetries: maxRetries,
		timeout:    c.timeout,
		retryDelay: retryTimeoutErrors,
		retryImmediate: func(err error) bool {
			// Ignore this packet.
			// Continue without increasing retransmission timeout or deadline.
			return errors.Is(err, &mistmatchedGatewayErr{})
		},
	}
	if retry.timeout == 0 {
		retry.timeout = 1 * time.Second
	}

	// 16 bytes is the maximum result size.
	result := make([]byte, 16)
	err := retry.run(func(deadline time.Time) error {
		d, remoteIP, err := c.transport.Send(reqBuf.Bytes(), result, deadline)
		if !remoteIP.Equal(c.gatewayIP) {
			// Ignore this packet.
			// Continue without increasing retransmission timeout or deadline.
			return &mistmatchedGatewayErr{c.gatewayIP, remoteIP}
		}
		result = d
		return err
	})
	if err != nil {
		return err
	}

	expectedSize := int(reflect.Indirect(reflect.ValueOf(resp)).Type().Size())
	expectedOp := req.opcode() | 0x80
	err = binary.Read(bytes.NewReader(result), binary.BigEndian, resp)

	switch {
	case len(result) != expectedSize:
		return fmt.Errorf("unexpected result size %d, expected %d", len(result), expectedSize)
	case errors.Is(err, io.EOF):
		return fmt.Errorf("unexpected result size %d for type %T", len(result), resp)
	case resp.version() != 0:
		return fmt.Errorf("unknown protocol version %d", resp.version())
	case resp.opcode() != expectedOp:
		return fmt.Errorf("unexpected opcode 0x%X (not 0x%X)", resp.opcode(), expectedOp)
	case resp.resultCode() != 0:
		return ResultCodeErr(resp.resultCode())
	}
	return nil
}

func retryTimeoutErrors(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

type mistmatchedGatewayErr struct {
	Remote   net.IP
	Gateways net.IP
}

func (e *mistmatchedGatewayErr) Is(err error) bool {
	_, ok := err.(*mistmatchedGatewayErr)
	return ok
}

func (e *mistmatchedGatewayErr) Error() string {
	return fmt.Sprintf("error remote address %s does not match specified gateway %s", e.Remote, e.Gateways)
}

type ResultCodeErr int

func (r ResultCodeErr) Error() string {
	return fmt.Sprintf("error NAT-PMP non-zero result code:%d", int(r))
}
