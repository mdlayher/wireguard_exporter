// Command wireguard_exporter implements a Prometheus exporter for WireGuard
// devices.
package main

import (
	"flag"
	"log"
	"net/http"

	wireguardexporter "github.com/mdlayher/wireguard_exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.zx2c4.com/wireguard/wgctrl"
)

func main() {
	var (
		metricsAddr = flag.String("metrics.addr", ":9586", "address for WireGuard exporter")
		metricsPath = flag.String("metrics.path", "/metrics", "URL path for surfacing collected metrics")
	)

	flag.Parse()

	client, err := wgctrl.New()
	if err != nil {
		log.Fatalf("failed to open WireGuard control client: %v", err)
	}
	defer client.Close()

	// Make Prometheus client aware of our collector.
	c := wireguardexporter.New(client.Devices)
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
