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

	"strconv"

	"regexp"

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

	runtimeInfo = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "runtime_info"),
		"Was information on the execution environment of kt.",
		[]string{
			"conf_kc_features",
			"conf_kc_version",
			"conf_kt_features",
			"conf_kt_version",
			"serv_thread_count",
			"repl_master_host",
			"repl_master_port",
		}, nil,
	)

	getRequests = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "get_requests"),
		"Is a value called cnt_get in kt report",
		nil, nil,
	)

	getMissesRequests = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "get_misses_requests"),
		"Is a value called cnt_get_misses in kt report",
		nil, nil,
	)

	miscRequests = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "misc_requests"),
		"Is a value called cnt_misc in kt report",
		nil, nil,
	)

	removeRequests = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "remove_requests"),
		"Is a value called cnt_remove in kt report",
		nil, nil,
	)

	removeMissesRequests = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "remove_misses_requests"),
		"Is a value called cnt_remove_misses in kt report",
		nil, nil,
	)

	scriptRequests = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "script_requests"),
		"Is a value called cnt_script in kt report",
		nil, nil,
	)

	setRequests = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "set_requests"),
		"Is a value called cnt_set in kt report",
		nil, nil,
	)

	setMissesRequests = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "set_misses_requests"),
		"Is a value called cnt_set_misses in kt report",
		nil, nil,
	)
	replDelaySeconds = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "repl_delay_seconds"),
		"Is a value called repl_delay in kt report",
		nil, nil,
	)
	replIntervalSeconds = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "repl_interval_seconds"),
		"Is a value called repl_interval in kt report",
		nil, nil,
	)
	serverConnections = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "server_connections"),
		"Is a value called serv_conn_count in kt report",
		nil, nil,
	)
	serverTasks = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "server_tasks"),
		"Is a value called serv_task_count in kt report",
		nil, nil,
	)
	dbRecords = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "db_records"),
		"Is a value called db count in kt report",
		[]string{"path"}, nil,
	)
	dbBytes = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "db_bytes"),
		"Is a value called db size in kt report",
		[]string{"path"}, nil,
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
	ch <- runtimeInfo
	ch <- getRequests
	ch <- getMissesRequests
	ch <- miscRequests
	ch <- removeRequests
	ch <- removeMissesRequests
	ch <- scriptRequests
	ch <- setRequests
	ch <- setMissesRequests
	ch <- replDelaySeconds
	ch <- replIntervalSeconds
	ch <- serverConnections
	ch <- serverTasks
	ch <- dbRecords
	ch <- dbBytes
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

	info := make(map[string]string)
	dbRegex := regexp.MustCompile(`db_\d`)

	lines := strings.Split(body, "\n")
	for _, line := range lines {

		metrics := strings.SplitN(line, "\t", 2)

		switch metrics[0] {
		case "cnt_get":
			e.collectMetric(ch, getRequests, metrics[0], metrics[1])
		case "cnt_get_misses":
			e.collectMetric(ch, getMissesRequests, metrics[0], metrics[1])
		case "cnt_misc":
			e.collectMetric(ch, miscRequests, metrics[0], metrics[1])
		case "cnt_remove":
			e.collectMetric(ch, removeRequests, metrics[0], metrics[1])
		case "cnt_remove_misses":
			e.collectMetric(ch, removeMissesRequests, metrics[0], metrics[1])
		case "cnt_script":
			e.collectMetric(ch, scriptRequests, metrics[0], metrics[1])
		case "cnt_set":
			e.collectMetric(ch, setRequests, metrics[0], metrics[1])
		case "serv_conn_count":
			e.collectMetric(ch, serverConnections, metrics[0], metrics[1])
		case "serv_task_count":
			e.collectMetric(ch, serverTasks, metrics[0], metrics[1])
		case "repl_delay":
			e.collectMetric(ch, replDelaySeconds, metrics[0], metrics[1])
		case "repl_interval":
			e.collectMetric(ch, replIntervalSeconds, metrics[0], metrics[1])
		case "conf_kc_features",
			"conf_kc_version",
			"conf_kt_features",
			"conf_kt_version",
			"serv_thread_count",
			"repl_master_host",
			"repl_master_port":
			info[metrics[0]] = metrics[1]
		}

		if dbRegex.MatchString(metrics[0]) {
			dbInfo := strings.SplitN(metrics[1], " ", 3)
			c := strings.SplitN(dbInfo[0], "=", 2)
			cnt, err := strconv.ParseFloat(c[1], 64)
			if err != nil {
				log.Errorf("Parsing of %v value %v to Int is failed: %v", c[0], c[1], err)
			}
			s := strings.SplitN(dbInfo[1], "=", 2)
			size, err := strconv.ParseFloat(s[1], 64)
			if err != nil {
				log.Errorf("Parsing of %v value %v to Int is failed: %v", s[0], s[1], err)
			}
			path := strings.SplitN(dbInfo[2], "=", 2)
			ch <- prometheus.MustNewConstMetric(
				dbRecords, prometheus.GaugeValue, cnt,
				path[1],
			)
			ch <- prometheus.MustNewConstMetric(
				dbBytes, prometheus.GaugeValue, size,
				path[1],
			)
		}

	}
	ch <- prometheus.MustNewConstMetric(
		runtimeInfo, prometheus.GaugeValue, 1,
		info["conf_kc_features"],
		info["conf_kc_version"],
		info["conf_kt_features"],
		info["conf_kt_version"],
		info["serv_thread_count"],
		info["repl_master_host"],
		info["repl_master_port"],
	)
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

func (e *Exporter) collectMetric(ch chan<- prometheus.Metric, desc *prometheus.Desc, k, v string) {
	val, err := strconv.ParseFloat(v, 64)
	if err != nil {
		log.Errorf("Parsing of %v value %v to Int is failed: %v", k, v, err)
	}
	ch <- prometheus.MustNewConstMetric(
		desc, prometheus.GaugeValue, val,
	)
}

func init() {
	prometheus.MustRegister(version.NewCollector("kyototycoon_exporter"))
}

func main() {
	var (
		showVersion   = flag.Bool("version", false, "Print version information.")
		listenAddress = flag.String("web.listen-address", ":9306", "Address to listen on for web interface and telemetry.")
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
