# Temperature and humidity sensor

Reading DHT22 sensor data from a Go binary and serving it to a prometheus instance.

Run: `./tempsensor` (or make it a service)

## Build
Requires:
* bash
* Go 1.11+
* Makefile
* for ARM `arm-linux-gnueabi-gcc` lib installed eg: `sudo apt install gcc-arm-linux-gnueabi`

Build: `make build` and copy the binary to the machine/rasberry pi.

## Prometheus

```bash
apt-get install prometheus prometheus-node-exporter prometheus-pushgateway prometheus-alertmanager
```
Add to /etc/prometheus/prometheus.yml
```yaml
  - job_name: 'livingroom'
    static_configs:
     - targets: ['localhost:9080']
```
Restart prometheus and check: http://localhost:9090/targets

