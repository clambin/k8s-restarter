package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/clambin/k8s-restarter/internal/client"
	"github.com/clambin/k8s-restarter/internal/reaper"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "k8s.io/client-go"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &opts)))

	slog.Info("k8s-restarter", "version", version)
	go runPrometheusServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go scan(ctx, *interval)

	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()
	<-ctx.Done()
}

func scan(ctx context.Context, interval time.Duration) {
	slog.Info("scanner started", "interval", interval, "namespace", *namespace, "deployment", *selector)
	defer slog.Info("scanner stopped")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := check(ctx, *namespace, *selector); err != nil {
			slog.Error("scan failed", "err", err)
		}
		if *once {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func check(ctx context.Context, namespace, name string) error {
	r := reaper.Reaper{Client: &client.Client{Connector: connect}}
	deleted, err := r.Reap(ctx, namespace, name)
	if err == nil {
		if deleted > 0 {
			slog.Info("deleted failing deployment", "count", deleted)
		}
	}
	return err
}

func connect() (kubernetes.Interface, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		// not running inside cluster. try to connect as external client
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("user home dir: %w", err)
		}
		kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")
		slog.Debug("not running inside cluster. using kube config", "filename", kubeConfigPath)

		cfg, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, fmt.Errorf("kubernetes config: %w", err)
		}
	}
	return kubernetes.NewForConfig(cfg)
}

func runPrometheusServer() {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":"+strconv.Itoa(*port), nil)
	if !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start prometheus metrics server", "err", err)
	}
}
