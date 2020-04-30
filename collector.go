package wireguardexporter

import (
	"bytes"
	"net"
	"sort"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var _ prometheus.Collector = &collector{}

// A collector is a prometheus.Collector for a WireGuard device.
type collector struct {
	DeviceInfo *prometheus.Desc

	PeerInfo          *prometheus.Desc
	PeerReceiveBytes  *prometheus.Desc
	PeerTransmitBytes *prometheus.Desc
	PeerLastHandshake *prometheus.Desc

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
			append(labels, []string{"allowed_ips", "endpoint", "name"}...),
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
				// TODO(mdlayher): is there a better way to represent allowed IP
				// ranges? Perhaps most users will use a single CIDR anyway and
				// it won't be a big deal.
				d.Name, pub, ipsString(p.AllowedIPs), endpoint, name,
			)

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

// ipsString produces a string representation of a list of allowed peer IP CIDR
// values.
func ipsString(ipns []net.IPNet) string {
	// In order to sort these values properly, we first convert them all to
	// strings. Sorting behavior appears to be indeterminate when dealing with
	// the net.IPNet types directly, likely due to net.IP and net.IPMask being
	// slice types.
	//
	// By this point, we assume all of the input net.IPNet values are valid,
	// and thus mustCIDR will force a panic if any of them are not.
	ss := make([]string, 0, len(ipns))
	for _, ipn := range ipns {
		ss = append(ss, ipn.String())
	}

	sort.SliceStable(ss, func(i, j int) bool {
		// Parse the strings for each check so we can sort by IP family, mask
		// length, and finally lexical order of addresses.
		ci, cj := mustCIDR(ss[i]), mustCIDR(ss[j])

		onesI, bitsI := ci.Mask.Size()
		onesJ, bitsJ := cj.Mask.Size()

		if bitsI < bitsJ || onesI < onesJ {
			return true
		}

		return bytes.Compare(ci.IP, cj.IP) == -1
	})

	return strings.Join(ss, ",")
}

// mustCIDR parses s as a net.IPNet or panics.
func mustCIDR(s string) net.IPNet {
	ip, cidr, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	cidr.IP = ip

	return *cidr
}
