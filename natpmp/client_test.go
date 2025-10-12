package natpmp

import (
	"bytes"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestGetExternalAddress(t *testing.T) {
	testCases := []struct {
		name         string
		wantAddr     netip.Addr
		wantDuration time.Duration
		err          error
		call         testCall
	}{
		{
			name: "propagate error",
			err:  fmt.Errorf("unexpected result size 0, expected 12"),
			call: testCall{
				req: []uint8{0x0, 0x0},
			},
		},
		{
			name:         "success",
			wantAddr:     netip.MustParseAddr("73.140.54.154"),
			wantDuration: 1307215 * time.Second,
			call: testCall{
				req:  []uint8{0x0, 0x0},
				resp: []uint8{0x0, 0x80, 0x0, 0x0, 0x0, 0x13, 0xf2, 0x4f, 0x49, 0x8c, 0x36, 0x9a},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := &fakeServer{
				Call: tc.call,
			}
			srv.Start(t)
			defer srv.Close()

			ipAddr, port := srv.Addr()
			t.Logf("Test Gateway: %s %d", ipAddr, port)

			c := NewClient(ipAddr, Port(port))
			gotAddr, gotDuration, err := c.GetExternalAddress()
			if tc.err != nil {
				if !errContains(err, tc.err.Error()) {
					t.Errorf("err=%v != %v", err, tc.err)
				}
				return
			}
			if gotDuration != tc.wantDuration {
				t.Errorf("result.EpochDuration=%d != %d", gotDuration, tc.wantDuration)
			}
			if gotAddr != tc.wantAddr {
				t.Errorf("result.ExternalAddr=%v != %v", gotAddr, tc.wantAddr)
			}
		})
	}
}

type portSpec struct {
	protocol              string
	internalPort          int
	requestedExternalPort int
	lifetime              time.Duration
}

func TestAddPortMapping(t *testing.T) {
	testCases := []struct {
		name     string
		portSpec portSpec
		result   *PortMapping
		err      error
		call     testCall
	}{
		{
			"Propagate error",
			portSpec{"udp", 123, 456, time.Duration(1200) * time.Second},
			&PortMapping{},
			fmt.Errorf("unexpected result size 0, expected 16"),
			testCall{
				req: []uint8{0x0, 0x1, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
			},
		},
		{
			"Add UDP",
			portSpec{"udp", 123, 456, time.Duration(1200) * time.Second},
			&PortMapping{
				EpochDuration:      0x13feff * time.Second,
				InternalPort:       0x7b,
				MappedExternalPort: 0x1c8,
				Lifetime:           0x4b0 * time.Second,
			},
			nil,
			testCall{
				req:  []uint8{0x0, 0x1, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				resp: []uint8{0x0, 0x81, 0x0, 0x0, 0x0, 0x13, 0xfe, 0xff, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
			},
		},
		{
			"Add TCP",
			portSpec{"tcp", 123, 456, time.Duration(1200) * time.Second},
			&PortMapping{
				EpochDuration:      0x140321 * time.Second,
				InternalPort:       0x7b,
				MappedExternalPort: 0x1c8,
				Lifetime:           0x4b0 * time.Second,
			},
			nil,
			testCall{
				req:  []uint8{0x0, 0x2, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				resp: []uint8{0x0, 0x82, 0x0, 0x0, 0x0, 0x14, 0x3, 0x21, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
			},
		},

		{
			"Remove UDP",
			portSpec{"udp", 123, 0, 0},
			&PortMapping{
				EpochDuration:      0x1403d5 * time.Second,
				InternalPort:       0x7b,
				MappedExternalPort: 0x0,
				Lifetime:           0x0 * time.Second,
			},
			nil,
			testCall{
				req:  []uint8{0x0, 0x1, 0x0, 0x0, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				resp: []uint8{0x0, 0x81, 0x0, 0x0, 0x0, 0x14, 0x3, 0xd5, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			},
		},
		{
			"Remove TCP",
			portSpec{"tcp", 123, 0, 0},
			&PortMapping{
				EpochDuration:      0x140496 * time.Second,
				InternalPort:       0x7b,
				MappedExternalPort: 0x0,
				Lifetime:           0x0 * time.Second,
			},
			nil,
			testCall{
				req:  []uint8{0x0, 0x2, 0x0, 0x0, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				resp: []uint8{0x0, 0x82, 0x0, 0x0, 0x0, 0x14, 0x4, 0x96, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := &fakeServer{
				Call: tc.call,
			}
			srv.Start(t)
			defer srv.Close()

			ipAddr, port := srv.Addr()
			t.Logf("Test Gateway: %s %d", ipAddr, port)
			c := NewClient(ipAddr, Port(port))

			result, err := c.AddPortMapping(tc.portSpec.protocol, tc.portSpec.internalPort, tc.portSpec.requestedExternalPort, tc.portSpec.lifetime)
			if tc.err != nil {
				if !errContains(err, tc.err.Error()) {
					t.Errorf("err=%v != %v", err, tc.err)
				}
				return
			}
			if result.EpochDuration != tc.result.EpochDuration {
				t.Errorf("result.EpochDuration=%d != %d", result.EpochDuration, tc.result.EpochDuration)
			}
			if result.InternalPort != tc.result.InternalPort {
				t.Errorf("result.InternalPort=%d != %d", result.InternalPort, tc.result.InternalPort)
			}
			if result.MappedExternalPort != tc.result.MappedExternalPort {
				t.Errorf("result.InternalPort=%d != %d", result.MappedExternalPort, tc.result.MappedExternalPort)
			}
			if result.Lifetime != tc.result.Lifetime {
				t.Errorf("result.InternalPort=%d != %d", result.Lifetime, tc.result.Lifetime)
			}
		})
	}
}

func TestProtocolChecks(t *testing.T) {
	testCases := []struct {
		name                  string
		protocol              string
		internalPort          int
		requestedExternalPort int
		lifetime              time.Duration
		result                *PortMapping
		err                   error
		cr                    testCall
	}{
		{
			"Unexpected result size",
			"tcp", 123, 456, time.Duration(1200) * time.Second,
			nil,
			fmt.Errorf("unexpected result size %d, expected %d", 1, 16),
			testCall{
				req:  []uint8{0x0, 0x2, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				resp: []uint8{0x0},
			},
		},
		{
			"Unknown protocol version",
			"tcp", 123, 456, time.Duration(1200) * time.Second,
			nil,
			fmt.Errorf("unknown protocol version %d", 1),
			testCall{
				req:  []uint8{0x0, 0x2, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				resp: []uint8{0x1, 0x82, 0x0, 0x0, 0x0, 0x14, 0x4, 0x96, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			},
		},
		{
			"Unexpected opcode",
			"tcp", 123, 456, time.Duration(1200) * time.Second,
			nil,
			fmt.Errorf("unexpected opcode 0x88 (not 0x82)"),
			testCall{
				req:  []uint8{0x0, 0x2, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				resp: []uint8{0x0, 0x88, 0x0, 0x0, 0x0, 0x14, 0x4, 0x96, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			},
		},
		{
			"Non-success result code",
			"tcp", 123, 456, time.Duration(1200) * time.Second,
			nil,
			fmt.Errorf("non-zero result code 17"),
			testCall{
				req:  []uint8{0x0, 0x2, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				resp: []uint8{0x0, 0x82, 0x0, 0x11, 0x0, 0x14, 0x4, 0x96, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			remote := net.UDPAddr{
				IP:   net.ParseIP("10.0.0.1"),
				Port: defaultPort,
			}
			transport := &testTransport{
				testCall: tc.cr,
			}
			c := NewClient(remote.IP, WithTransport(transport))
			result, err := c.AddPortMapping(tc.protocol, tc.internalPort, tc.requestedExternalPort, tc.lifetime)
			if tc.err != nil {
				if !errContains(err, tc.err.Error()) {
					t.Errorf("err=%v != %v", err, tc.err)
				}
				return
			}
			if result != nil {
				t.Errorf("result=%v != nil", result)
			}
		})
	}
}

func errContains(err error, substr string) bool {
	return err != nil && strings.Contains(err.Error(), substr)
}

type testCall struct {
	req  []byte
	resp []byte
}
type fakeServer struct {
	Call     testCall
	listener net.PacketConn
}

func (s *fakeServer) Addr() (net.IP, int) {
	udp := s.listener.LocalAddr().(*net.UDPAddr)
	return udp.IP, udp.Port
}

func (s *fakeServer) Start(t *testing.T) {
	listener, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start UDP listener on available port: %v", err)
	}
	s.listener = listener
	go func() {
		buffer := make([]byte, 1024)
		n, clientAddr, err := listener.ReadFrom(buffer)
		request := buffer[:n]
		if err != nil {
			t.Errorf("Server failed to read from client: %v", err)
			return
		}
		if bytes.Compare(request, s.Call.req) != 0 {
			t.Errorf("got=%v  want=%v", request, s.Call.req)
		}
		n, err = listener.WriteTo(s.Call.resp[:], clientAddr)
		if err != nil {
			t.Errorf("Server failed to write response to client: %v", err)
		}
		if n != len(s.Call.resp) {
			t.Errorf("Wrote too many bytes?")
		}
	}()
}

func (s *fakeServer) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

type testTransport struct {
	err      error
	gateway  net.IP
	testCall testCall
}

var _ Transport = (*testTransport)(nil)

func (t *testTransport) Open(g net.IP, port int) error {
	t.gateway = g
	return nil
}
func (t *testTransport) Close() error { return nil }
func (t *testTransport) Send(req, resp []byte, deadline time.Time) ([]byte, net.IP, error) {
	if bytes.Compare(req, t.testCall.req) != 0 {
		return nil, nil, fmt.Errorf("got=%v  want=%v", req, t.testCall.req)
	}
	if t.err != nil {
		return nil, nil, t.err
	}
	n := copy(resp, t.testCall.resp)
	return resp[:n], t.gateway, nil
}
