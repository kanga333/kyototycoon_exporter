package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	ktReport = `cnt_get	1
cnt_get_misses	0
cnt_misc	0
cnt_remove	0
cnt_remove_misses	0
cnt_script	0
cnt_set	1
cnt_set_misses	0
conf_kc_features	(atomic)(zlib)
conf_kc_version	1.2.76 (16.13)
conf_kt_features	(epoll)(lua)
conf_kt_version	0.9.56 (2.19)
conf_os_name	Linux
db_0	count=1 size=8388691 path=:
db_total_count	1
db_total_size	8388691
serv_conn_count	1
serv_current_time	1505531246.924222
serv_proc_id	1
serv_running_term	120.520679
serv_task_count	0
serv_thread_count	16
sys_mem_cached	1586905088
sys_mem_free	185495552
sys_mem_peak	27377664
sys_mem_rss	2883584
sys_mem_size	27377664
sys_mem_total	2096164864
sys_ru_stime	0.200000
sys_ru_utime	1.260000
`
)

func checkKtStatus(t *testing.T, report string, metricCount int) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rpc/report" {
			t.Errorf("Invalid URL")
		}
		w.Write([]byte(report))
	})

	server := httptest.NewServer(handler)
	opts := kyototycoonOpts{
		uri:     server.URL,
		timeout: 200 * time.Millisecond,
	}
	e, err := NewExporter(opts)
	if err != nil {
		t.Fatal("err shoud be nil but: ", err)
	}
	ch := make(chan prometheus.Metric)

	go func() {
		defer close(ch)
		e.Collect(ch)
	}()

	counter := 0
	for _ = range ch {
		counter++
	}
	if counter != metricCount {
		t.Errorf("counter should be %d But %d", metricCount, counter)
	}
}

func TestKTStatus(t *testing.T) {
	checkKtStatus(t, ktReport, 14)
}
