# kyototycoon_exporter

A kyototycoon exporter for prometheus.

## Build

```
make build
```

## run

```
./kyototycoon_exporter
```

## Options

```
$ ./kyototycoon_exporter -help
Usage of ./kyototycoon_exporter:
  -kt.server string
        HTTP API address of a KyotoTycoon server. (default "http://localhost:1978")
  -kt.timeout duration
        Timeout on HTTP requests to kyototycoon. (default 200ms)
  -log.format value
        Set the log target and format. Example: "logger:syslog?appname=bob&local=7" or "logger:stdout?json=true" (default "logger:stderr")
  -log.level value
        Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]
  -v    Print version information.
  -version
        Print version information.
  -web.listen-address string
        Address to listen on for web interface and telemetry. (default ":9306")
  -web.telemetry-path string
        Path under which to expose metrics. (default "/metrics")
```