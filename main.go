package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jenningsloy318/redfish_exporter/collector"
	"github.com/stmcginnis/gofish"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

const (
	exporterName = "redfish_exporter"
)

var (
	configFile  = kingpin.Flag("config.file", "Path to configuration file.").Default("config.yml").String()
	webConfig   = kingpinflag.AddFlags(kingpin.CommandLine, ":9610")
	printConfig = kingpin.Flag("print.config", "Print the loaded configuration and exit.").Bool()
	logger      = promslog.NewNopLogger()

	Version       string
	BuildRevision string
	BuildBranch   string
	BuildTime     string
	BuildHost     string

	sc = &SafeConfig{
		C: &Config{},
	}
	reloadCh chan chan error
)

func reloadHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" || r.Method == "PUT" {
			logger.Info("Triggered configuration reload from /-/reload HTTP endpoint")
			err := sc.ReloadConfig(*configFile)
			if err != nil {
				logger.Error("failed to reload config file", "error", err)
				http.Error(w, "failed to reload config file", http.StatusInternalServerError)
			}
			logger.With("operation", "sc.ReloadConfig").Info("config file reloaded")

			w.WriteHeader(http.StatusOK)
			_, err = io.WriteString(w, "Configuration reloaded successfully!")
			if err != nil {
				logger.Warn("failed to send configuration reload status message")
			}
		} else {
			http.Error(w, "Only PUT and POST methods are allowed", http.StatusBadRequest)
		}
	}
}

// define new http handleer
func metricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()
		target := r.URL.Query().Get("target")
		if target == "" {
			http.Error(w, "'target' parameter must be specified", 400)
			return
		}
		targetLoggerCtx := logger.With("target", target)
		targetLoggerCtx.Info("scraping target host")

		var (
			hostConfig *HostConfig
			err        error
			ok         bool
			module     []string
		)

		module, ok = r.URL.Query()["module"]

		if ok && len(module[0]) >= 1 {
			// Trying to get hostConfig from group.
			if hostConfig, err = sc.HostConfigForModule(module[0]); err != nil {
				targetLoggerCtx.With("error", err).Error("error getting credentials")
				return
			}
		}

		// Always falling back to single host config when group config failed.
		if hostConfig == nil {
			if hostConfig, err = sc.HostConfigForTarget(target); err != nil {
				targetLoggerCtx.With("error", err).Error("error getting credentials")
				return
			}
		}

		redfishClient, err := newRedfishClient(target, hostConfig.Username, hostConfig.Password)
		if err != nil {
			logger.Error("error creating redfish client", "error", err)
			http.Error(w, "error creating redfish client", 500)
			return
		}
		defer redfishClient.Logout()

		collector := collector.NewRedfishCollector(redfishClient, targetLoggerCtx)
		registry.MustRegister(collector)
		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}

func newRedfishClient(host string, username string, password string) (*gofish.APIClient, error) {
	url := fmt.Sprintf("https://%s", host)

	config := gofish.ClientConfig{
		Endpoint: url,
		Username: username,
		Password: password,
		Insecure: true, // todo: security
	}
	redfishClient, err := gofish.Connect(config)
	if err != nil {
		return nil, err
	}
	return redfishClient, nil
}

func main() {
	kingpin.Version(version.Print(exporterName))
	promslogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promslogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger = promslog.New(promslogConfig)

	logger.Info("starting app")
	// load config  first time
	if err := sc.ReloadConfig(*configFile); err != nil {
		logger.With("error", err).Error("error parsing config file")
		panic(err)
	}

	logger.With("operation", "sc.ReloadConfig").Info("config file loaded")

	// load config in background to watch for config changes
	hup := make(chan os.Signal, 1)
	reloadCh = make(chan chan error)
	signal.Notify(hup, syscall.SIGHUP)

	go func() {
		for {
			select {
			case <-hup:
				if err := sc.ReloadConfig(*configFile); err != nil {
					logger.With("error", err).Error("failed to reload config file")
					break
				}
				logger.With("operation", "sc.ReloadConfig").Info("config file reload")
			case rc := <-reloadCh:
				if err := sc.ReloadConfig(*configFile); err != nil {
					logger.With("error", err).Error("failed to reload config file")
					rc <- err
					break
				}
				logger.With("operation", "sc.ReloadConfig").Info("config file reloaded")
				rc <- nil
			}
		}
	}()

	http.Handle("/redfish", metricsHandler())       // Regular metrics endpoint for local Redfish metrics.
	http.Handle("/-/reload", reloadHandler(logger)) // HTTP endpoint for triggering configuration reload
	http.Handle("/metrics", promhttp.Handler())
	landingConfig := web.LandingConfig{
		Name:        "Redfish Exporter",
		Description: "Prometheus Redfish Exporter",
		Version:     version.Info(),
		Links: []web.LandingLinks{
			{
				Address: "/metrics",
				Text:    "Exporter Metrics",
			},
			{
				Address: "/redfish",
				Text:    "Multi-target Redfish Exporter Endpoint",
			},
		},
	}
	landingPage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		logger.Error("error creating landing page", "err", err)
		os.Exit(1)
	}
	http.Handle("/", landingPage)

	srv := &http.Server{}
	if err := web.ListenAndServe(srv, webConfig, logger); err != nil {
		logger.Error("Error running HTTP server", "err", err)
		os.Exit(1)
	}
}
