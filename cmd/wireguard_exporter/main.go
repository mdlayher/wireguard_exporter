// Command wireguard_exporter implements a Prometheus exporter for WireGuard
// devices.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	wireguardexporter "github.com/mdlayher/wireguard_exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.zx2c4.com/wireguard/wgctrl"
)

func main() {
	var (
		metricsAddr     = flag.String("metrics.addr", ":9586", "address for WireGuard exporter")
		metricsPath     = flag.String("metrics.path", "/metrics", "URL path for surfacing collected metrics")
		wgPeerNames     = flag.String("wireguard.peer-names", "", `optional: comma-separated list of colon-separated public keys and friendly peer names, such as: "keyA:foo,keyB:bar"`)
		wgPeerFile      = flag.String("wireguard.peer-file", "", "optional: path to TOML friendly peer names mapping file; takes priority over -wireguard.peer-names and -dsnet.config-file")
		dsnetConfigFile = flag.String("dsnet.config-file", "", "optional: path to dsnet config file for peer name mapping")
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

		log.Printf("loaded %d peer name mappings from command line", len(peerNames))
	}

	// Read peer names from dsnet config file.
	if file := *dsnetConfigFile; file != "" {
		f, err := os.Open(file)
		if err != nil {
			log.Fatalf("failed to open dsnet config file: %v", err)
		}
		defer f.Close()

		names, err := wireguardexporter.ParseDsnetConfig(f)
		if err != nil {
			log.Fatalf("failed to parse peer names from dsnet config: %v", err)
		}

		log.Printf("loaded %d peer name mappings from file %q", len(names), file)

		// Merge file name mappings and overwrite CLI mappings if necessary.
		for k, v := range names {
			peerNames[k] = v
		}
	}

	// In addition, load peer name mappings from a file if specified.
	if file := *wgPeerFile; file != "" {
		f, err := os.Open(file)
		if err != nil {
			log.Fatalf("failed to open peer names file: %v", err)
		}
		defer f.Close()

		names, err := wireguardexporter.ParsePeers(f)
		if err != nil {
			log.Fatalf("failed to parse peer names file: %v", err)
		}
		_ = f.Close()

		log.Printf("loaded %d peer name mappings from file %q", len(names), file)

		// Merge file name mappings and overwrite CLI mappings if necessary.
		for k, v := range names {
			peerNames[k] = v
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
	server := http.Server{
		Addr:         *metricsAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("cannot start WireGuard exporter: %s", err)
	}
}
