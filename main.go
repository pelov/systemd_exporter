package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/povilasv/systemd_exporter/systemd"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	var (
		listenAddress = kingpin.Flag(
			"web.listen-address",
			"Address on which to expose metrics and web interface.",
		).Default(":9558").String()
		metricsPath = kingpin.Flag(
			"web.telemetry-path",
			"Path under which to expose metrics.",
		).Default("/metrics").String()
		disableExporterMetrics = kingpin.Flag(
			"web.disable-exporter-metrics",
			"Exclude metrics about the exporter itself (promhttp_*, process_*, go_*).",
		).Bool()
		maxRequests = kingpin.Flag(
			"web.max-requests",
			"Maximum number of parallel scrape requests. Use 0 to disable.",
		).Default("40").Int()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("systemd_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting systemd_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	exporterMetricsRegistry := prometheus.NewRegistry()
	r := prometheus.NewRegistry()

	r.MustRegister(version.NewCollector("systemd_exporter"))

	collector, err := systemd.NewCollector(log.Base())
	if err != nil {
		log.Fatalf("couldn't create collector: %s", err)
	}

	if err := r.Register(collector); err != nil {
		log.Fatalf("couldn't register systemd collector: %s", err)
	}

	handler := promhttp.HandlerFor(
		prometheus.Gatherers{exporterMetricsRegistry, r},
		promhttp.HandlerOpts{
			ErrorLog:            log.NewErrorLogger(),
			ErrorHandling:       promhttp.ContinueOnError,
			MaxRequestsInFlight: *maxRequests,
		},
	)

	if !*disableExporterMetrics {
		handler = promhttp.InstrumentMetricHandler(
			exporterMetricsRegistry, handler,
		)
	}

	http.Handle(*metricsPath, handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`<html>
			<head><title>Systemd Exporter</title></head>
			<body>
			<h1>Systemd Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
		if err != nil {
			log.Errorf("couldn't write response: %s", err)
		}
	})

	log.Infoln("Listening on", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		log.Fatal(err)
	}
}
