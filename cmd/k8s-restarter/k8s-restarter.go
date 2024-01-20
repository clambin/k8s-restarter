package main

import (
	"context"
	"errors"
	"flag"
	"github.com/clambin/k8s-restarter/internal/client"
	"github.com/clambin/k8s-restarter/internal/k8s"
	"github.com/clambin/k8s-restarter/internal/reaper"
	"github.com/clambin/k8s-restarter/internal/scanner"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "k8s.io/client-go"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var (
	version = "change_me"

	namespace = flag.String("namespace", "media", "namespace")
	selector  = flag.String("selector", "app=transmission", "label selector")
	interval  = flag.Duration("interval", time.Minute, "scanning interval")
	once      = flag.Bool("once", false, "scan once and exit")
	port      = flag.Int("port", 9091, "Prometheus metrics port")
	debug     = flag.Bool("debug", false, "enable debug mode")
)

func main() {
	flag.Parse()

	var opts slog.HandlerOptions
	if *debug {
		opts.Level = slog.LevelDebug
	}
	l := slog.New(slog.NewJSONHandler(os.Stderr, &opts))

	l.Info("k8s-restarter", "version", version)
	go runPrometheusServer(l)

	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	s := scanner.Scanner{
		Namespace:     *namespace,
		LabelSelector: *selector,
		Reaper: &reaper.Reaper{
			Client: &client.Client{Connect: k8s.Connector(l)},
			Logger: l.With("component", "reaper"),
		},
		Logger: l,
	}

	if err := s.ScanOnce(ctx); err != nil {
		l.Error("failed to scan for dead pods", "err", err)
		return
	}

	if *once {
		return
	}

	go s.Scan(ctx, *interval)
	<-ctx.Done()
}

func runPrometheusServer(l *slog.Logger) {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":"+strconv.Itoa(*port), nil)
	if !errors.Is(err, http.ErrServerClosed) {
		l.Error("failed to start prometheus metrics server", "err", err)
	}
}
