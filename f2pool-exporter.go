package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"strings"
	"os"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	listenAddress = flag.String("listen-address", ":5896", "Address to listen on for web interface and telemetry")
	metricsPath   = flag.String("telemetry-path", "/metrics", "Path to expose metrics of the exporter")
	resourcesArg = flag.String("resources", "", "Resources ({currency}/{user or address}) to retrieve, separated by commas")
	version string
	build   string

	f2pool_balance = prometheus.NewDesc(prometheus.BuildFQName("f2pool", "", "balance"), "Unpaid balance", []string {"currency", "account"} , nil)
	f2pool_paid = prometheus.NewDesc(prometheus.BuildFQName("f2pool", "", "paid"), "Paid balance", []string {"currency", "account"} , nil)
	f2pool_value = prometheus.NewDesc(prometheus.BuildFQName("f2pool", "", "value"), "Total revenue", []string {"currency", "account"} , nil)
	f2pool_value_last_day = prometheus.NewDesc(prometheus.BuildFQName("f2pool", "", "value_last_day"),
		"Revenue of last 24 hours", []string {"currency", "account"} , nil)
	f2pool_stale_hashes_rejected_last_day = prometheus.NewDesc(prometheus.BuildFQName("f2pool", "", "stale_hashes_rejected_last_day"),
		"Stale rejected hashes of last 24 hours", []string {"currency", "account", "worker"} , nil)
	f2pool_stale_hashes_rejected_last_hour = prometheus.NewDesc(prometheus.BuildFQName("f2pool", "", "stale_hashes_rejected_last_hour"),
		"Stale rejected hashes of last hour", []string {"currency", "account", "worker"} , nil)
	f2pool_hashes_last_day = prometheus.NewDesc(prometheus.BuildFQName("f2pool", "", "hashes_last_day"), "Hashes of last 24 hours",
		[]string {"currency", "account", "worker"}, nil)
	f2pool_hashes_last_hour = prometheus.NewDesc(prometheus.BuildFQName("f2pool", "", "hashes_last_hour"), "Hashes of last hour",
		[]string {"currency", "account", "worker"}, nil)
	f2pool_hashrate = prometheus.NewDesc(prometheus.BuildFQName("f2pool", "", "hashrate"), "Current hashrate",
		[]string {"currency", "account", "worker"}, nil)
	f2pool_worker_shares_time = prometheus.NewDesc(prometheus.BuildFQName("f2pool", "", "worker_shares_time"),
		"Recently submitted shares time (in seconds)", []string {"currency", "account", "worker"}, nil)
)


type F2PoolExporter struct {
	client *http.Client
	resources []string
}

func NewF2PoolExporter(resources []string) (*F2PoolExporter, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify : true},
	}
	h := &http.Client{ Timeout: 10 * time.Second, Transport: tr }

	return &F2PoolExporter{ client: h, resources: resources }, nil
}

func (e *F2PoolExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- f2pool_balance
	ch <- f2pool_paid
	ch <- f2pool_value
	ch <- f2pool_value_last_day
	ch <- f2pool_stale_hashes_rejected_last_day
	ch <- f2pool_stale_hashes_rejected_last_hour
	ch <- f2pool_hashes_last_day
	ch <- f2pool_hashes_last_hour
	ch <- f2pool_hashrate
}

func (e *F2PoolExporter) Collect(ch chan<- prometheus.Metric) {
	for _, resource := range e.resources {
		tmp := strings.Split(resource, "/")
		currency := tmp[0]
		account := tmp[1]


		var infos map[string]interface{}
		infosBody := HttpGetCall(e.client, "https://api.f2pool.com/" + resource)
		err := json.Unmarshal([]byte(infosBody), &infos)
		if err != nil {
			log.Fatal(err)
		}
		
		ch <- prometheus.MustNewConstMetric(f2pool_balance, prometheus.GaugeValue, infos["balance"].(float64), currency, account)
		ch <- prometheus.MustNewConstMetric(f2pool_paid, prometheus.GaugeValue, infos["paid"].(float64), currency, account)
		ch <- prometheus.MustNewConstMetric(f2pool_value, prometheus.GaugeValue, infos["value"].(float64), currency, account)
		ch <- prometheus.MustNewConstMetric(f2pool_value_last_day, prometheus.GaugeValue, infos["value_last_day"].(float64), currency, account)
		ch <- prometheus.MustNewConstMetric(f2pool_stale_hashes_rejected_last_day, prometheus.GaugeValue, infos["stale_hashes_rejected_last_day"].(float64), currency, account, "all")
		ch <- prometheus.MustNewConstMetric(f2pool_stale_hashes_rejected_last_hour, prometheus.GaugeValue, infos["stale_hashes_rejected_last_hour"].(float64), currency, account, "all")
		ch <- prometheus.MustNewConstMetric(f2pool_hashes_last_day, prometheus.GaugeValue, infos["hashes_last_day"].(float64), currency, account, "all")
		ch <- prometheus.MustNewConstMetric(f2pool_hashes_last_hour, prometheus.GaugeValue, infos["hashes_last_hour"].(float64), currency, account, "all")
		ch <- prometheus.MustNewConstMetric(f2pool_hashrate, prometheus.GaugeValue, infos["hashrate"].(float64), currency, account, "all")

		for _, w := range infos["workers"].([]interface{}) {
			worker := w.([]interface{})
			label := worker[0].(string)

			ch <- prometheus.MustNewConstMetric(f2pool_hashrate, prometheus.GaugeValue, worker[1].(float64), currency, account, label)
			ch <- prometheus.MustNewConstMetric(f2pool_hashes_last_hour, prometheus.GaugeValue, worker[2].(float64), currency, account, label)
			ch <- prometheus.MustNewConstMetric(f2pool_hashes_last_day, prometheus.GaugeValue, worker[4].(float64), currency, account, label)
			ch <- prometheus.MustNewConstMetric(f2pool_stale_hashes_rejected_last_hour, prometheus.GaugeValue, worker[3].(float64), currency, account, label)
			ch <- prometheus.MustNewConstMetric(f2pool_stale_hashes_rejected_last_day, prometheus.GaugeValue, worker[5].(float64), currency, account, label)
			t, e := time.Parse(time.RFC3339, worker[6].(string))
			if e != nil {
				ch <- prometheus.MustNewConstMetric(f2pool_worker_shares_time, prometheus.GaugeValue, float64(t.Unix()), currency, account, label)
			}
		}
	}
}

func main() {
	flag.Parse()

	resources := strings.Split(*resourcesArg, ",")

	if len(*resourcesArg) == 0 {
		log.Fatal("Resources required")
		os.Exit(1)
	}

	fmt.Println("Version:", version)
	fmt.Println("Build Time:", build)
	fmt.Println("Resources:", resources)
	fmt.Println("Metrics Path:", *metricsPath)

	exporter, err := NewF2PoolExporter(resources)
	if err != nil {
		log.Fatal("Error initializing exporter")
		os.Exit(1)
	}

	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, *metricsPath, http.StatusMovedPermanently)
	})
	
	fmt.Println("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}



// HTTP call utility method

func HttpGetCall(client *http.Client, uri string) (string) {
	req, err := http.NewRequest("GET", uri, nil)

	if err != nil {
        log.Fatal(err)
    }

	resp, err := client.Do(req)

	if err != nil {
        log.Fatal(err)
    }

    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)

    if err != nil {
        log.Fatal(err)
    }

    return string(body)
}
