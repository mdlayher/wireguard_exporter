# wireguard_exporter [![builds.sr.ht status](https://builds.sr.ht/~mdlayher/wireguard_exporter.svg)](https://builds.sr.ht/~mdlayher/wireguard_exporter?) [![GoDoc](https://godoc.org/github.com/mdlayher/wireguard_exporter?status.svg)](https://godoc.org/github.com/mdlayher/wireguard_exporter) [![Go Report Card](https://goreportcard.com/badge/github.com/mdlayher/wireguard_exporter)](https://goreportcard.com/report/github.com/mdlayher/wireguard_exporter)

Command `wireguard_exporter` implements a Prometheus exporter for WireGuard
devices. MIT Licensed.

## Usage

Use the `-h` flag to see full usage:

```text
$ wireguard_exporter -h
Usage of wireguard_exporter:
  -metrics.addr string
        address for WireGuard exporter (default ":9586")
  -metrics.path string
        URL path for surfacing collected metrics (default "/metrics")
  -wireguard.peer-file string
        optional: path to TOML friendly peer names mapping file; takes priority over -wireguard.peer-names
  -wireguard.peer-names string
        optional: comma-separated list of colon-separated public keys and friendly peer names, such as: "keyA:foo,keyB:bar"
```

For simple deployments, specifying peer name mappings on the command line may
be sufficient:

```text
$ wireguard_exporter -wireguard.peer-names VWRsPtbdGtcNyaQ+cFAZfZnYL05uj+XINQS6yQY5gQ8=:foo
```

For larger deployments, you can also specify a TOML file of friendly peer name
mappings, which will supersede any command line flag mappings.

```toml
[[peer]]
public_key = "VWRsPtbdGtcNyaQ+cFAZfZnYL05uj+XINQS6yQY5gQ8="
name = "foo"

[[peer]]
public_key = "UvwWyMQ1ckLEG82Qdooyr0UzJhqOlzzcx90DXuwMTDA="
name = "bar"
```

```text
$ wireguard_exporter -wireguard.peer-file /etc/wireguard/peers.toml
```

## Example

This exporter exposes metrics about each configured WireGuard device and its
peers, using any device implementation supported by [wgctrl-go](https://github.com/WireGuard/wgctrl-go).

```text
$ curl -s http://localhost:9586/metrics | grep wireguard
# HELP wireguard_device_info Metadata about a device.
# TYPE wireguard_device_info gauge
wireguard_device_info{device="wg0",public_key="QwAmAD1v4wMIX/0gKJbr9hv1o3YX0YTk7Mdj0L4dylI="} 1
# HELP wireguard_peer_allowed_ips_info Metadata about each of a peer's allowed IP subnets for a given device.
# TYPE wireguard_peer_allowed_ips_info gauge
wireguard_peer_allowed_ips_info{allowed_ips="192.168.20.0/24",device="wg0",public_key="UvwWyMQ1ckLEG82Qdooyr0UzJhqOlzzcx90DXuwMTDA="} 1
wireguard_peer_allowed_ips_info{allowed_ips="fd9e:1a04:f01d:20::/64",device="wg0",public_key="UvwWyMQ1ckLEG82Qdooyr0UzJhqOlzzcx90DXuwMTDA="} 1
# HELP wireguard_peer_info Metadata about a peer. The public_key label on peer metrics refers to the peer's public key; not the device's public key.
# TYPE wireguard_peer_info gauge
wireguard_peer_info{device="wg0",endpoint="",name="foo",public_key="VWRsPtbdGtcNyaQ+cFAZfZnYL05uj+XINQS6yQY5gQ8="} 1
wireguard_peer_info{device="wg0",endpoint="[fd9e:1a04:f01d:20:e5c2:7b69:90d8:ca45]:49203",name="bar",public_key="UvwWyMQ1ckLEG82Qdooyr0UzJhqOlzzcx90DXuwMTDA="} 1
# HELP wireguard_peer_last_handshake_seconds UNIX timestamp for the last handshake with a given peer.
# TYPE wireguard_peer_last_handshake_seconds gauge
wireguard_peer_last_handshake_seconds{device="wg0",public_key="UvwWyMQ1ckLEG82Qdooyr0UzJhqOlzzcx90DXuwMTDA="} 1.588274629e+09
wireguard_peer_last_handshake_seconds{device="wg0",public_key="VWRsPtbdGtcNyaQ+cFAZfZnYL05uj+XINQS6yQY5gQ8="} 0
# HELP wireguard_peer_receive_bytes_total Number of bytes received from a given peer.
# TYPE wireguard_peer_receive_bytes_total counter
wireguard_peer_receive_bytes_total{device="wg0",public_key="UvwWyMQ1ckLEG82Qdooyr0UzJhqOlzzcx90DXuwMTDA="} 76728
wireguard_peer_receive_bytes_total{device="wg0",public_key="VWRsPtbdGtcNyaQ+cFAZfZnYL05uj+XINQS6yQY5gQ8="} 0
# HELP wireguard_peer_transmit_bytes_total Number of bytes transmitted to a given peer.
# TYPE wireguard_peer_transmit_bytes_total counter
wireguard_peer_transmit_bytes_total{device="wg0",public_key="UvwWyMQ1ckLEG82Qdooyr0UzJhqOlzzcx90DXuwMTDA="} 76200
wireguard_peer_transmit_bytes_total{device="wg0",public_key="VWRsPtbdGtcNyaQ+cFAZfZnYL05uj+XINQS6yQY5gQ8="} 0
```

### Sample queries

Get the receive and transmit rates of individual peers, and enable querying on
both the WireGuard device name and the peer's friendly name:

```
irate(wireguard_peer_receive_bytes_total[5m]) * on (public_key, device) group_left(name) wireguard_peer_info * on (instance) group_left(device) wireguard_device_info
```
```
irate(wireguard_peer_transmit_bytes_total[5m]) * on (public_key, device) group_left(name) wireguard_peer_info * on (instance) group_left(device) wireguard_device_info
```

## Grafana Dashboard

You can view your data using this [Grafana Dashboard](https://grafana.com/grafana/dashboards/12177) using Prometheus as source.

![Grafana Dashboard](grafana_wireguard.png)

## Build Binary 

```
cd cmd/wireguard_exporter/
go build .
mv wireguard_exporter /usr/local/bin/
```

## Add service file for systemd

```
[Unit]
Description=Prometheus WireGuard Exporter
After=network.target

[Service]
Type=simple
Restart=always
ExecStart=/usr/local/bin/wireguard_exporter

[Install]
WantedBy=multi-user.target
```

Load new service and enable autostart:

```
systemctl daemon-reload
systemctl enable wireguard-exporter.service
```

## Add scraping config to prometheus

In `/etc/prometheus/prometheus.yml` add following config to the section `scrape_configs:` :

```
  - job_name: wireguard
    static_configs:
      - targets: ['localhost:9586']
```
