package restarter

import (
	"context"
	"fmt"
	"iter"
	coreV1 "k8s.io/api/core/v1"
	"log/slog"
	"time"
)

type K8SClient interface {
	GetPodsForLabelSelector(ctx context.Context, namespace string, labelSelector string) ([]coreV1.Pod, error)
	DeletePod(ctx context.Context, namespace, name string) error
	Disconnect()
}

func Scan(
	ctx context.Context,
	client K8SClient,
	namespace string,
	labelSelector string,
	interval time.Duration,
	version string,
	logger *slog.Logger,
) error {
	logger.Info("restarter started", "interval", interval, "namespace", namespace, "selector", labelSelector, "version", version)
	defer logger.Info("scanner stopped")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := ScanOnce(ctx, client, namespace, labelSelector, logger); err != nil {
			logger.Error("scan failed", "err", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func ScanOnce(
	ctx context.Context,
	client K8SClient,
	namespace string,
	labelSelector string,
	logger *slog.Logger,
) error {
	defer client.Disconnect()

	pods, err := client.GetPodsForLabelSelector(ctx, namespace, labelSelector)
	if err != nil {
		return fmt.Errorf("GetPodsForLabelSelector: %w", err)
	}

	for pod := range getNotReady(pods, logger) {
		if err = client.DeletePod(ctx, namespace, pod.GetName()); err != nil {
			logger.Warn("failed to delete pod", "err", err, "name", pod.GetName())
			continue
		}
		logger.Info("pod deleted", "name", pod.GetName())
	}
	return nil
}

func getNotReady(pods []coreV1.Pod, logger *slog.Logger) iter.Seq[coreV1.Pod] {
	return func(yield func(coreV1.Pod) bool) {
		for _, pod := range pods {
			l := logger.With("name", pod.GetName())

			l.Debug("checking pod")

			status := getPodConditionStatus(pod, coreV1.PodReady)
			switch status {
			case coreV1.ConditionTrue:
				l.Debug("pod is ready")
			case coreV1.ConditionFalse:
				l.Debug("pod not ready")
				if !yield(pod) {
					return
				}
			default:
				l.Debug("pod doesn't appear to be running", "status", status)
			}
		}
	}
}

func getPodConditionStatus(pod coreV1.Pod, conditionType coreV1.PodConditionType) coreV1.ConditionStatus {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return ""
}
