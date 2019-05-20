package wireguardexporter

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/prometheus/util/promlint"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestCollector(t *testing.T) {
	tests := []struct {
		name    string
		devices func() ([]*wgtypes.Device, error)
		metrics []string
	}{
		{
			name: "ok",
			devices: func() ([]*wgtypes.Device, error) {
				// Fake public keys used to identify devices and peers.
				var (
					devA  = publicKey(0x01)
					devB  = publicKey(0x02)
					peerA = publicKey(0x03)
				)

				return []*wgtypes.Device{
					{
						Name:      "wg0",
						PublicKey: devA,
						Peers: []wgtypes.Peer{{
							PublicKey:     peerA,
							ReceiveBytes:  1,
							TransmitBytes: 2,
							AllowedIPs: []net.IPNet{
								{
									IP:   net.ParseIP("192.168.1.0"),
									Mask: net.CIDRMask(24, 32),
								},
								{
									IP:   net.ParseIP("2001:db8::"),
									Mask: net.CIDRMask(32, 128),
								},
							},
						}},
					},
					{
						Name:      "wg1",
						PublicKey: devB,
					},
				}, nil
			},
			metrics: []string{
				`wireguard_device_info{device="wg0",public_key="AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="} 1`,
				`wireguard_device_info{device="wg1",public_key="AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI="} 1`,
				`wireguard_peer_info{allowed_ips="192.168.1.0/24,2001:db8::/32",device="wg0",public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 1`,
				`wireguard_peer_receive_bytes_total{public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 1`,
				`wireguard_peer_transmit_bytes_total{public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 2`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := testCollector(t, tt.devices)

			s := bufio.NewScanner(bytes.NewReader(body))
			for s.Scan() {
				// Skip metric HELP and TYPE lines.
				text := s.Text()
				if strings.HasPrefix(text, "#") {
					continue
				}

				var found bool
				for _, m := range tt.metrics {
					if text == m {
						found = true
						break
					}
				}

				if !found {
					t.Log(string(body))
					t.Fatalf("metric string not matched in whitelist: %s", text)
				}
			}

			if err := s.Err(); err != nil {
				t.Fatalf("failed to scan metrics: %v", err)
			}
		})
	}
}

// testCollector uses the input device to generate a blob of Prometheus text
// format metrics.
func testCollector(t *testing.T, devices func() ([]*wgtypes.Device, error)) []byte {
	t.Helper()

	r := prometheus.NewPedanticRegistry()
	r.MustRegister(New(devices))
	h := promhttp.HandlerFor(r, promhttp.HandlerOpts{})

	s := httptest.NewServer(h)
	defer s.Close()

	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	res, err := http.Get(u.String())
	if err != nil {
		t.Fatalf("failed to perform HTTP request: %v", err)
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	// Ensure best practices are followed by linting the metrics.
	problems, err := promlint.New(bytes.NewReader(b)).Lint()
	if err != nil {
		t.Fatalf("failed to lint metrics: %v", err)
	}

	if len(problems) > 0 {
		for _, p := range problems {
			t.Logf("lint: %s: %s", p.Metric, p.Text)
		}

		t.Fatal("one or more promlint errors found")
	}

	return b
}

func publicKey(b byte) wgtypes.Key {
	key, err := wgtypes.NewKey(bytes.Repeat([]byte{b}, wgtypes.KeyLen))
	if err != nil {
		panicf("failed to make public key: %v", err)
	}

	return key
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
