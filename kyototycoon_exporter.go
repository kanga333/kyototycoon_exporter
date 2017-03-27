package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
)

const (
	namespace = "kyototycoon"
)

var (
	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Was the last query of kt successful.",
		nil, nil,
	)
)

// Exporter collects kyototycoon stats from the given server and exports them using
// the prometheus metrics package.
type Exporter struct {
	client *http.Client
	url    string
}

type kyototycoonOpts struct {
	uri     string
	timeout time.Duration
}

// NewExporter returns an initialized Exporter.
func NewExporter(opts kyototycoonOpts) (*Exporter, error) {
	// Check url and format it to the correct url
	uri := opts.uri
	if !strings.Contains(uri, "://") {
		uri = "http://" + uri
	}
	if !strings.Contains(uri, "/rpc/report") {
		uri = uri + "/rpc/report"
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid kt URL: %s", err)
	}
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("invalid kt URL: %s", uri)
	}

	// Init http client.
	client := &http.Client{Timeout: opts.timeout}

	// Init our exporter.
	return &Exporter{
		client: client,
		url:    uri,
	}, nil
}

// Describe describes all the metrics ever exported by the KyotoTycoon exporter.
// It implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
}

// Collect fetches the stats from configured KyotoTycoon location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	// How many peers are in the Consul cluster?
	body, err := e.getKtReport()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0,
		)
		log.Errorf("Access to /rpc/report failed: %v", err)
		return
	}
	// We'll use peers to decide that we're up.
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1,
	)
	log.Infoln("get body", body)

}

func (e *Exporter) getKtReport() (string, error) {
	resp, err := e.client.Get(e.url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("err Status %s (%d): %s", resp.Status, resp.StatusCode, body)
	}
	return string(body), nil
}

func init() {
	prometheus.MustRegister(version.NewCollector("kyototycoon_exporter"))
}

func main() {
	var (
		showVersion   = flag.Bool("version", false, "Print version information.")
		listenAddress = flag.String("web.listen-address", ":9107", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")

		opts = kyototycoonOpts{}
	)
	flag.BoolVar(showVersion, "v", false, "Print version information.")
	flag.StringVar(&opts.uri, "kt.server", "http://localhost:1978", "HTTP API address of a KyotoTycoon server.")
	flag.DurationVar(&opts.timeout, "kt.timeout", 200*time.Millisecond, "Timeout on HTTP requests to kyototycoon.")

	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Print("kyototycoon_exporter"))
		os.Exit(0)
	}

	log.Infoln("Starting kyototycoon_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	exporter, err := NewExporter(opts)
	if err != nil {
		log.Fatalln(err)
	}
	prometheus.MustRegister(exporter)
	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>KyotoTycoon Exporter</title></head>
             <body>
             <h1>KyotoTycoon Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))

}
