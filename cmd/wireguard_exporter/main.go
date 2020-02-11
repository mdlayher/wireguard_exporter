// Command wireguard_exporter implements a Prometheus exporter for WireGuard
// devices.
package main

import (
	"flag"
	"log"
	"net/http"
	"runtime"
	"strings"

	wireguardexporter "github.com/mdlayher/wireguard_exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vishvananda/netns"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func main() {
	var (
		metricsAddr = flag.String("metrics.addr", ":9586", "address for WireGuard exporter")
		metricsPath = flag.String("metrics.path", "/metrics", "URL path for surfacing collected metrics")
		wgPeerNames = flag.String("wireguard.peer-names", "", `optional: comma-separated list of colon-separated public keys and friendly peer names, such as: "keyA:foo,keyB:bar"`)
		netnsNames  = flag.String("netns", "", `optional: comma-separated list of network namespace names to check for wireguard interfaces. e.g: "foo,bar"`)

		deviceFunc func() ([]*wgtypes.Device, error)
	)

	flag.Parse()

	if *netnsNames == "" {
		client, err := wgctrl.New()
		if err != nil {
			log.Fatalf("failed to open WireGuard control client: %v", err)
		}
		defer client.Close()
		deviceFunc = client.Devices
	} else {
		deviceFunc = allDevicesAcrossNamespaces(strings.Split(*netnsNames, ","))
	}

	// Configure the friendly peer names map if the flag is not empty.
	peerNames := make(map[string]string)
	if *wgPeerNames != "" {
		for _, kvs := range strings.Split(*wgPeerNames, ",") {
			kv := strings.Split(kvs, ":")
			if len(kv) != 2 {
				log.Fatalf("failed to parse %q as a valid public key and peer name", kv)
			}

			peerNames[kv[0]] = kv[1]
		}
	}

	// Make Prometheus client aware of our collector.
	c := wireguardexporter.New(deviceFunc, peerNames)
	prometheus.MustRegister(c)

	// Set up HTTP handler for metrics.
	mux := http.NewServeMux()
	mux.Handle(*metricsPath, promhttp.Handler())

	// Start listening for HTTP connections.
	log.Printf("starting WireGuard exporter on %q", *metricsAddr)
	if err := http.ListenAndServe(*metricsAddr, mux); err != nil {
		log.Fatalf("cannot start WireGuard exporter: %s", err)
	}
}

// iterates across all given network namespaces and returns devices discovered amongst all
func allDevicesAcrossNamespaces(namespaces []string) func() ([]*wgtypes.Device, error) {
	return func() ([]*wgtypes.Device, error) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		toplevelNamespace, _ := netns.Get()
		defer netns.Set(toplevelNamespace)

		toReturn := make([]*wgtypes.Device, 0, 10)
		for _, namespace := range namespaces {
			ns, err := netns.GetFromName(namespace)
			if err != nil {
				return nil, err
			}

			netns.Set(ns)
			client, err := wgctrl.New()
			if err != nil {
				return nil, err
			}

			devices, err := client.Devices()
			if err != nil {
				return nil, err
			}

			toReturn = append(toReturn, devices...)
		}

		return toReturn, nil
	}
}
