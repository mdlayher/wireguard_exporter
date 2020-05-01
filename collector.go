package wireguardexporter

import (
	"fmt"
	"log"
	"net"

	"github.com/prometheus/client_golang/prometheus"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var _ prometheus.Collector = &collector{}

// A collector is a prometheus.Collector for a WireGuard device.
type collector struct {
	DeviceInfo *prometheus.Desc

	PeerInfo           *prometheus.Desc
	PeerAllowedIPsInfo *prometheus.Desc
	PeerReceiveBytes   *prometheus.Desc
	PeerTransmitBytes  *prometheus.Desc
	PeerLastHandshake  *prometheus.Desc

	devices   func() ([]*wgtypes.Device, error)
	peerNames map[string]string
}

// New constructs a prometheus.Collector using a function to fetch WireGuard
// device information (typically using *wgctrl.Client.Devices).
func New(devices func() ([]*wgtypes.Device, error), peerNames map[string]string) prometheus.Collector {
	// Permit nil map to mean no peer names configured.
	if peerNames == nil {
		peerNames = make(map[string]string)
	}

	// Per-peer metrics are keyed on both device and public key since a peer
	// can be associated with multiple devices.
	labels := []string{"device", "public_key"}

	return &collector{
		DeviceInfo: prometheus.NewDesc(
			"wireguard_device_info",
			"Metadata about a device.",
			labels,
			nil,
		),

		PeerInfo: prometheus.NewDesc(
			"wireguard_peer_info",
			"Metadata about a peer. The public_key label on peer metrics refers to the peer's public key; not the device's public key.",
			append(labels, []string{"endpoint", "name"}...),
			nil,
		),

		PeerAllowedIPsInfo: prometheus.NewDesc(
			"wireguard_peer_allowed_ips_info",
			"Metadata about each of a peer's allowed IP subnets for a given device.",
			append(labels, []string{"allowed_ips", "family"}...),
			nil,
		),

		PeerReceiveBytes: prometheus.NewDesc(
			"wireguard_peer_receive_bytes_total",
			"Number of bytes received from a given peer.",
			labels,
			nil,
		),

		PeerTransmitBytes: prometheus.NewDesc(
			"wireguard_peer_transmit_bytes_total",
			"Number of bytes transmitted to a given peer.",
			labels,
			nil,
		),

		PeerLastHandshake: prometheus.NewDesc(
			"wireguard_peer_last_handshake_seconds",
			"UNIX timestamp for the last handshake with a given peer.",
			labels,
			nil,
		),

		devices:   devices,
		peerNames: peerNames,
	}
}

// Describe implements prometheus.Collector.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.DeviceInfo,
		c.PeerInfo,
		c.PeerAllowedIPsInfo,
		c.PeerReceiveBytes,
		c.PeerTransmitBytes,
		c.PeerLastHandshake,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect implements prometheus.Collector.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	devices, err := c.devices()
	if err != nil {
		log.Printf("failed to list devices: %v", err)
		ch <- prometheus.NewInvalidMetric(c.DeviceInfo, err)
		return
	}

	for _, d := range devices {
		ch <- prometheus.MustNewConstMetric(
			c.DeviceInfo,
			prometheus.GaugeValue,
			1,
			d.Name, d.PublicKey.String(),
		)

		for _, p := range d.Peers {
			pub := p.PublicKey.String()

			// Use empty string instead of special Go <nil> syntax for no endpoint.
			var endpoint string
			if p.Endpoint != nil {
				endpoint = p.Endpoint.String()
			}

			// If a friendly name is configured, add it as a label value.
			name := c.peerNames[pub]

			ch <- prometheus.MustNewConstMetric(
				c.PeerInfo,
				prometheus.GaugeValue,
				1,
				d.Name, pub, endpoint, name,
			)

			for _, ip := range p.AllowedIPs {
				ch <- prometheus.MustNewConstMetric(
					c.PeerAllowedIPsInfo,
					prometheus.GaugeValue,
					1,
					d.Name, pub, ip.String(), ipFamily(ip.IP),
				)
			}

			ch <- prometheus.MustNewConstMetric(
				c.PeerReceiveBytes,
				prometheus.CounterValue,
				float64(p.ReceiveBytes),
				d.Name, pub,
			)

			ch <- prometheus.MustNewConstMetric(
				c.PeerTransmitBytes,
				prometheus.CounterValue,
				float64(p.TransmitBytes),
				d.Name, pub,
			)

			// Expose last handshake of 0 unless a last handshake time is set.
			var last float64
			if !p.LastHandshakeTime.IsZero() {
				last = float64(p.LastHandshakeTime.Unix())
			}

			ch <- prometheus.MustNewConstMetric(
				c.PeerLastHandshake,
				prometheus.GaugeValue,
				last,
				d.Name, pub,
			)
		}
	}
}

func ipFamily(ip net.IP) string {
	if ip.To16() == nil {
		panicf("invalid IP address: %q", ip)
	}

	if ip.To4() == nil {
		return "IPv6"
	}

	return "IPv4"
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
