// Command wireguard_exporter implements a Prometheus exporter for WireGuard
// devices.
package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	wireguardexporter "github.com/mdlayher/wireguard_exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.zx2c4.com/wireguard/wgctrl"
)

func main() {
	var (
		metricsAddr = flag.String("metrics.addr", ":9586", "address for WireGuard exporter")
		metricsPath = flag.String("metrics.path", "/metrics", "URL path for surfacing collected metrics")
		wgPeerNames = flag.String("wireguard.peer-names", "", `optional: comma-separated list of colon-separated public keys and friendly peer names, such as: "keyA:foo,keyB:bar"`)
	)

	flag.Parse()

	client, err := wgctrl.New()
	if err != nil {
		log.Fatalf("failed to open WireGuard control client: %v", err)
	}
	defer client.Close()

	if _, err := client.Devices(); err != nil {
		log.Fatalf("failed to fetch WireGuard devices: %v", err)
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
	c := wireguardexporter.New(client.Devices, peerNames)
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
