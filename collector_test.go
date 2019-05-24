package wireguardexporter

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mdlayher/promtest"
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
							PublicKey: peerA,
							Endpoint: &net.UDPAddr{
								IP:   net.ParseIP("fd00::1"),
								Port: 51820,
							},
							LastHandshakeTime: time.Unix(10, 0),
							ReceiveBytes:      1,
							TransmitBytes:     2,
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
				`wireguard_peer_info{allowed_ips="192.168.1.0/24,2001:db8::/32",device="wg0",endpoint="[fd00::1]:51820",public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 1`,
				`wireguard_peer_last_handshake_seconds{public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 10`,
				`wireguard_peer_receive_bytes_total{public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 1`,
				`wireguard_peer_transmit_bytes_total{public_key="AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM="} 2`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := promtest.Collect(t, New(tt.devices))

			if !promtest.Lint(t, body) {
				t.Fatal("one or more promlint errors found")
			}

			if !promtest.Match(t, body, tt.metrics) {
				t.Fatal("metrics did not match whitelist")
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
