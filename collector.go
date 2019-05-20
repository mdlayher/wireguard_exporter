package wireguardexporter

import (
	"net"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var _ prometheus.Collector = &collector{}

// A collector is a prometheus.Collector for a WireGuard device.
type collector struct {
	DeviceInfo *prometheus.Desc
	PeerInfo   *prometheus.Desc

	PeerReceiveBytes  *prometheus.Desc
	PeerTransmitBytes *prometheus.Desc

	devices func() ([]*wgtypes.Device, error)
}

// New constructs a prometheus.Collector using a function to fetch WireGuard
// device information (typically using *wgctrl.Client.Devices).
func New(devices func() ([]*wgtypes.Device, error)) prometheus.Collector {
	return &collector{
		DeviceInfo: prometheus.NewDesc(
			"wireguard_device_info",
			"Metadata about a device.",
			[]string{"device", "public_key"},
			nil,
		),

		PeerInfo: prometheus.NewDesc(
			"wireguard_peer_info",
			"Metadata about a peer. The public_key label on peer metrics refers to the peer's public key; not the device's public key.",
			[]string{"device", "public_key", "allowed_ips"},
			nil,
		),

		PeerReceiveBytes: prometheus.NewDesc(
			"wireguard_peer_receive_bytes_total",
			"Number of bytes received from a given peer.",
			[]string{"public_key"},
			nil,
		),

		PeerTransmitBytes: prometheus.NewDesc(
			"wireguard_peer_transmit_bytes_total",
			"Number of bytes transmitted to a given peer.",
			[]string{"public_key"},
			nil,
		),

		devices: devices,
	}
}

// Describe implements prometheus.Collector.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.DeviceInfo,
		c.PeerInfo,
		c.PeerReceiveBytes,
		c.PeerTransmitBytes,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect implements prometheus.Collector.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	devices, err := c.devices()
	if err != nil {
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

			ch <- prometheus.MustNewConstMetric(
				c.PeerInfo,
				prometheus.GaugeValue,
				1,
				// TODO(mdlayher): is there a better way to represent allowed IP
				// ranges? Perhaps most users will use a single CIDR anyway and
				// it won't be a big deal.
				d.Name, pub, ipsString(p.AllowedIPs),
			)

			ch <- prometheus.MustNewConstMetric(
				c.PeerReceiveBytes,
				prometheus.CounterValue,
				float64(p.ReceiveBytes),
				pub,
			)

			ch <- prometheus.MustNewConstMetric(
				c.PeerTransmitBytes,
				prometheus.CounterValue,
				float64(p.TransmitBytes),
				pub,
			)
		}
	}
}

// ipsString produces a string representation of a list of allowed peer IP CIDR
// values.
func ipsString(ipns []net.IPNet) string {
	ss := make([]string, 0, len(ipns))
	for _, ipn := range ipns {
		ss = append(ss, ipn.String())
	}

	return strings.Join(ss, ",")
}
