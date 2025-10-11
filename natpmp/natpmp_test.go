package natpmp

import (
	"bytes"
	"fmt"
	"net/netip"
	"testing"
	"time"
)

func TestGetExternalAddress(t *testing.T) {
	dummyError := fmt.Errorf("dummy error")
	testCases := []struct {
		wantAddr     netip.Addr
		wantDuration time.Duration
		err          error
		cr           callRecord
	}{
		{
			err: dummyError,
			cr:  callRecord{[]uint8{0x0, 0x0}, nil, dummyError},
		},
		{
			wantAddr:     netip.MustParseAddr("73.140.54.154"),
			wantDuration: 1307215 * time.Second,
			cr:           callRecord{[]uint8{0x0, 0x0}, []uint8{0x0, 0x80, 0x0, 0x0, 0x0, 0x13, 0xf2, 0x4f, 0x49, 0x8c, 0x36, 0x9a}, nil},
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Logf("case %d", i)
			c := Client{&fakeCaller{t, tc.cr}, 0}
			addr, duration, err := c.GetExternalAddress()
			if err != nil {
				if err != tc.err {
					t.Error(err)
				}
				return
			}
			if duration != tc.wantDuration {
				t.Errorf("result.EpochDuration=%d != %d", duration, tc.wantDuration)
			}
			if addr != tc.wantAddr {
				t.Errorf("result.ExternalAddr=%v != %v", addr, tc.wantAddr)
			}
		})
	}
}

type addPortMappingRecord struct {
	protocol              string
	internalPort          int
	requestedExternalPort int
	lifetime              int
	result                *PortMapping
	err                   error
	cr                    callRecord
}

func TestAddPortMapping(t *testing.T) {
	dummyError := fmt.Errorf("dummy error")
	testCases := []addPortMappingRecord{
		// Propagate error
		{
			"udp", 123, 456, 1200,
			nil,
			dummyError,
			callRecord{
				[]uint8{0x0, 0x1, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				nil,
				dummyError,
			},
		},
		// Add UDP
		{
			"udp", 123, 456, 1200,
			&PortMapping{
				EpochDuration:      0x13feff * time.Second,
				InternalPort:       0x7b,
				MappedExternalPort: 0x1c8,
				Lifetime:           0x4b0 * time.Second,
			},
			nil,
			callRecord{
				[]uint8{0x0, 0x1, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				[]uint8{0x0, 0x81, 0x0, 0x0, 0x0, 0x13, 0xfe, 0xff, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				nil,
			},
		},
		// Add TCP
		{
			"tcp", 123, 456, 1200,
			&PortMapping{
				EpochDuration:      0x140321 * time.Second,
				InternalPort:       0x7b,
				MappedExternalPort: 0x1c8,
				Lifetime:           0x4b0 * time.Second,
			},
			nil,
			callRecord{
				[]uint8{0x0, 0x2, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				[]uint8{0x0, 0x82, 0x0, 0x0, 0x0, 0x14, 0x3, 0x21, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				nil,
			},
		},
		// Remove UDP
		{
			"udp", 123, 0, 0,
			&PortMapping{
				EpochDuration:      0x1403d5 * time.Second,
				InternalPort:       0x7b,
				MappedExternalPort: 0x0,
				Lifetime:           0x0 * time.Second,
			},
			nil,
			callRecord{
				[]uint8{0x0, 0x1, 0x0, 0x0, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				[]uint8{0x0, 0x81, 0x0, 0x0, 0x0, 0x14, 0x3, 0xd5, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				nil,
			},
		},
		// Remove TCP
		{
			"tcp", 123, 0, 0,
			&PortMapping{
				EpochDuration:      0x140496 * time.Second,
				InternalPort:       0x7b,
				MappedExternalPort: 0x0,
				Lifetime:           0x0 * time.Second,
			},
			nil,
			callRecord{
				[]uint8{0x0, 0x2, 0x0, 0x0, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				[]uint8{0x0, 0x82, 0x0, 0x0, 0x0, 0x14, 0x4, 0x96, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				nil,
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			c := Client{&fakeCaller{t, tc.cr}, 0}
			result, err := c.AddPortMapping(tc.protocol, tc.internalPort, tc.requestedExternalPort, tc.lifetime)
			if err != nil || tc.err != nil {
				if err != tc.err && fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tc.err) {
					t.Errorf("err=%v != %v", err, tc.err)
				}
			} else {
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
			}
		})
	}
}

func TestProtocolChecks(t *testing.T) {
	testCases := []addPortMappingRecord{
		// Unexpected result size.
		{
			"tcp", 123, 456, 1200,
			nil,
			fmt.Errorf("unexpected result size %d, expected %d", 1, 16),
			callRecord{
				[]uint8{0x0, 0x2, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				[]uint8{0x0},
				nil,
			},
		},
		//  Unknown protocol version.
		{
			"tcp", 123, 456, 1200,
			nil,
			fmt.Errorf("unknown protocol version %d", 1),
			callRecord{
				[]uint8{0x0, 0x2, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				[]uint8{0x1, 0x82, 0x0, 0x0, 0x0, 0x14, 0x4, 0x96, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				nil,
			},
		},
		// Unexpected opcode.
		{
			"tcp", 123, 456, 1200,
			nil,
			fmt.Errorf("Unexpected opcode %d. Expected %d", 0x88, 0x82),
			callRecord{
				[]uint8{0x0, 0x2, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				[]uint8{0x0, 0x88, 0x0, 0x0, 0x0, 0x14, 0x4, 0x96, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				nil,
			},
		},
		// Non-success result code.
		{
			"tcp", 123, 456, 1200,
			nil,
			fmt.Errorf("Non-zero result code %d", 17),
			callRecord{
				[]uint8{0x0, 0x2, 0x0, 0x0, 0x0, 0x7b, 0x1, 0xc8, 0x0, 0x0, 0x4, 0xb0},
				[]uint8{0x0, 0x82, 0x0, 0x11, 0x0, 0x14, 0x4, 0x96, 0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				nil,
			},
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			t.Logf("case %d", i)
			c := Client{&fakeCaller{t, tc.cr}, 0}
			result, err := c.AddPortMapping(tc.protocol, tc.internalPort, tc.requestedExternalPort, tc.lifetime)
			if err != tc.err && fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tc.err) {
				t.Errorf("err=%v != %v", err, tc.err)
			}
			if result != nil {
				t.Errorf("result=%v != nil", result)
			}
		})
	}
}

type callRecord struct {
	// The expected msg argument to call.
	wantMsg []byte
	result  []byte
	err     error
}

type fakeCaller struct {
	// test object, used to report errors.
	t  *testing.T
	cr callRecord
}

func (n *fakeCaller) call(msg []byte, timeout time.Duration) (result []byte, err error) {
	if bytes.Compare(msg, n.cr.wantMsg) != 0 {
		n.t.Errorf("wantMsg=%v, expected %v", msg, n.cr.wantMsg)
	}
	return n.cr.result, n.cr.err
}

type getExternalAddressRecord struct {
}
