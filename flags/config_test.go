package flags

import (
	"errors"
	diffcmp "github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"net"
	"strings"
	"testing"
	"time"
)

func TestFlags(t *testing.T) {
	testGW := net.ParseIP("10.0.0.1")
	testCases := []struct {
		name       string
		args       []string
		wantErr    error
		wantConfig *Config
	}{
		{
			name:       "help",
			args:       []string{"-h"},
			wantConfig: &Config{Help: true},
		},
		{
			name: "gateway",
			args: []string{"-g", "10.0.0.1"},
			wantConfig: &Config{
				Gateway: IPValue(testGW),
			},
		},
		{
			name: "add-no-lifetime",
			args: []string{"-a", "10", "10", "udp"},
			wantConfig: &Config{
				AddSpec: PortSpec{10, 10, "udp", 0},
			},
		},
		{
			name: "add-lifetime",
			args: []string{"-a", "10", "10", "udp", "100"},
			wantConfig: &Config{
				AddSpec: PortSpec{10, 10, "udp", 100 * time.Second},
			},
		},
		{
			name:    "err/missing-external",
			args:    []string{"-a", "10"},
			wantErr: errors.New("missing public port"),
		},
		{
			name:    "err/missing-protocol",
			args:    []string{"-a", "10", "10"},
			wantErr: errors.New("missing protocol"),
		},
		{
			name:    "err/missing-port-flag",
			args:    []string{"-a", "10", "-g", "10.0.0.1"},
			wantErr: errors.New("missing public port"),
		},
		{
			name:    "err/missing-protocol",
			args:    []string{"-a", "10", "10"},
			wantErr: errors.New("missing protocol"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var cfg Config
			err := cfg.ParseArgs(nil, tc.args)
			switch {
			case err == nil && tc.wantErr != nil:
				t.Errorf("got error %v wanted %v", err, tc.wantErr)
			case err != nil && tc.wantErr == nil:
				t.Errorf("got error %v wanted %v", err, tc.wantErr)
			case err != nil && tc.wantErr != nil:
				if !strings.Contains(err.Error(), tc.wantErr.Error()) {
					t.Errorf("got error %v wanted %v", err, tc.wantErr)
				}
			}

			if err == nil {
				if diff := diffcmp.Diff(&cfg, tc.wantConfig,
					cmpopts.IgnoreUnexported(PortSpec{})); diff != "" {
					t.Errorf("Got Diff %v", diff)
				}
			}
		})
	}
}
