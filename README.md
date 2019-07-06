# Prometheus Artifactory Tests Exporter
Prometheus exporter for Artifactory SLO/SLI tests

### What this exporter doing
- Push and pull artifacts
- Measure time for push and pull tests
- Data output in "Gauge" and "Histograms".

Default values for parameters which can be redefined in config file [artifactory-tests.yml](dpkg-sources/dirs/opt/prometheus/prometheus-artifactory-tests-exporter/artifactory-tests.yml)

```
listen_address: "127.0.0.1:9702"
metrics_path: "/metrics"
interval: "60s"
timeout: "5s"
test_files_path: "/opt/prometheus/prometheus-artifactory-tests-exporter/test-files"
log_level: "none"

test_files:
  <test_filename>:
    timeout_push: "<maximum possible timeout, see below>"
    timeout_pull: "<maximum possible timeout, see below>"
    verify_checksum: false
```

where,

`interval` - interval between tests

`timeout` - handler timeout

`test_files_path` - path to test files

`log_level` - can be "none", "info", "error" or "debug"

`timeout_push` and `timeout_pull` - timeout for push and pull tests. For full-cycle tests interval maximum possible value should be calculated by formula:
```
<interval> / 2 (push and pull tests) / num files - 1 second
```
If not defined, default value will be maximum possible.

`verify_checksum` - verify checksum for pulled artifact

# Compile and run

Re-download deps (if necessary)
```
rm -rf Gopkg.toml Gopkg.lock vendor && make gopkg.toml
```

Run
```
go run main.go config.go
```

# Build DEB package in Docker

With default variables for Ubuntu Bionic:
```bash
$ make build-deb
```
For Ubuntu Trusty:
```bash
make build-deb-trusty
```

With defined variables:
```bash
$ make build-deb PKG_VENDOR='Pkg Vendor Name' PKG_MAINTAINER='Pkg Maintainer' PKG_URL='http://example.com/no-uri-given'
```

After build, package will be in `deb-packages` local dir.

DEB-package installation dir: `/opt/prometheus/prometheus-artifactory-tests-exporter`
