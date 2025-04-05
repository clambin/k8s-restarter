package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/clambin/k8s-restarter/internal/restarter"
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
	"syscall"
	"time"
)

var (
	version = "change_me"

	namespace     = flag.String("namespace", "media", "namespace")
	labelSelector = flag.String("selector", "app=transmission", "label selector")
	interval      = flag.Duration("interval", time.Minute, "scanning interval")
	once          = flag.Bool("once", false, "scan once and exit")
	addr          = flag.String("addr", ":9091", "Prometheus metrics port")
	debug         = flag.Bool("debug", false, "enable debug mode")
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	err := Run(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "restarter exited with error: %v\n", err)
		os.Exit(1)
	}
}

func Run(ctx context.Context) error {
	flag.Parse()
	var opts slog.HandlerOptions
	if *debug {
		opts.Level = slog.LevelDebug
	}
	l := slog.New(slog.NewJSONHandler(os.Stderr, &opts))
	k8sClient := &restarter.Client{Connect: k8sConnector(l)}
	if *once {
		return restarter.ScanOnce(ctx, k8sClient, *namespace, *labelSelector, l)
	}
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(*addr, nil); !errors.Is(err, http.ErrServerClosed) {
			l.Error("failed to start metrics server", "err", err)
		}
	}()
	return restarter.Scan(ctx, k8sClient, *namespace, *labelSelector, *interval, version, l)
}

func k8sConnector(logger *slog.Logger) func() (kubernetes.Interface, error) {
	return func() (kubernetes.Interface, error) {
		cfg, err := rest.InClusterConfig()
		if err != nil {
			// not running inside cluster. try to connect as external client
			userHomeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("user home dir: %w", err)
			}
			kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")
			logger.Debug("not running inside cluster. using kube config", "filename", kubeConfigPath)

			cfg, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
			if err != nil {
				return nil, fmt.Errorf("kubernetes config: %w", err)
			}
		}
		return kubernetes.NewForConfig(cfg)
	}
}
