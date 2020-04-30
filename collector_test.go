package wireguardexporter

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/promtest"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestCollector(t *testing.T) {
	// Fake public keys used to identify devices and peers.
	var (
		devA  = publicKey(0x01)
		devB  = publicKey(0x02)
		peerA = publicKey(0x03)
	)

	tests := []struct {
		name      string
		devices   func() ([]*wgtypes.Device, error)
		peerNames map[string]string
		metrics   []string
	}{
		{
			name: "ok",
			devices: func() ([]*wgtypes.Device, error) {
				return []*wgtypes.Device{
					{
						Name:      "wg0",
						PublicKey: devA,
						Peers: []wgtypes.Peer{{
							PublicKey: peerA,
							Endpoint: &net.UDPAddr{
								IP:   net.ParseIP("fd00::1"),
								Port: 51820,
							},
							LastHandshakeTime: time.Unix(10, 0),
							ReceiveBytes:      1,
							TransmitBytes:     2,
							AllowedIPs: []net.IPNet{
								mustCIDR("192.168.1.0/24"),
								mustCIDR("2001:db8::/32"),
							},
						}},
					},
					{
						Name:      "wg1",
						PublicKey: devB,
						// Allow the same peer to be associated with
						// multiple devices.
						Peers: []wgtypes.Peer{{
							PublicKey: peerA,
							AllowedIPs: []net.IPNet{
								mustCIDR("0.0.0.0/0"),
							},
						}},
					},
				}, nil
			},
			peerNames: map[string]string{
				peerA.String(): "foo",
			},
			metrics: []string{
				`wireguard_device_info{device="wg0",public_key="AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="} 1`,
				`wireguard_device_info{device="wg1",public_key="AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI="} 1`,
				`wireguard_peer_info{allowed_ips="192.168.1.0/24,2001:db8::/32",device="wg0",endpoint="[fd00::1]:51820",name="foo",public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 1`,
				`wireguard_peer_info{allowed_ips="0.0.0.0/0",device="wg1",endpoint="",name="foo",public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 1`,
				`wireguard_peer_last_handshake_seconds{device="wg0",public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 10`,
				`wireguard_peer_last_handshake_seconds{device="wg1",public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 0`,
				`wireguard_peer_receive_bytes_total{device="wg0",public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 1`,
				`wireguard_peer_receive_bytes_total{device="wg1",public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 0`,
				`wireguard_peer_transmit_bytes_total{device="wg0",public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 2`,
				`wireguard_peer_transmit_bytes_total{device="wg1",public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 0`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := promtest.Collect(t, New(tt.devices, tt.peerNames))

			if !promtest.Lint(t, body) {
				t.Fatal("one or more promlint errors found")
			}

			if !promtest.Match(t, body, tt.metrics) {
				t.Fatal("metrics did not match whitelist")
			}
		})
	}
}

func Test_ipsString(t *testing.T) {
	tests := []struct {
		name string
		in   []net.IPNet
		out  string
	}{
		{
			name: "empty",
		},
		{
			name: "noop",
			in: []net.IPNet{
				mustCIDR("192.168.1.0/24"),
				mustCIDR("2001:db8::/32"),
			},
			out: "192.168.1.0/24,2001:db8::/32",
		},
		{
			name: "all",
			in: []net.IPNet{
				mustCIDR("192.0.2.0/24"),
				mustCIDR("2001:db8::/64"),
				mustCIDR("192.51.100.1/32"),
				mustCIDR("2001:db8:aaaa::2/128"),
				mustCIDR("2001:db8:aaaa::1/128"),
				mustCIDR("192.168.0.0/16"),
				mustCIDR("2001:db8:ffff::/48"),
			},
			out: "192.168.0.0/16,192.0.2.0/24,192.51.100.1/32,2001:db8:ffff::/48,2001:db8::/64,2001:db8:aaaa::1/128,2001:db8:aaaa::2/128",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.out, ipsString(tt.in)); diff != "" {
				t.Fatalf("unexpected output (-want +got):\n%s", diff)
			}
		})
	}
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
