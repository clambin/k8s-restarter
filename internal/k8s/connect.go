package k8s

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log/slog"
	"os"
	"path/filepath"
)

func Connector(logger *slog.Logger) func() (kubernetes.Interface, error) {
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
