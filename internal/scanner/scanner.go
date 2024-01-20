package scanner

import (
	"context"
	"fmt"
	coreV1 "k8s.io/api/core/v1"
	"log/slog"
	"time"
)

type Scanner struct {
	Namespace     string
	LabelSelector string
	Client        PodMan
	Logger        *slog.Logger
}

type PodMan interface {
	GetPodsForLabelSelector(ctx context.Context, namespace string, labelSelector string) ([]coreV1.Pod, error)
	DeletePod(ctx context.Context, namespace, name string) error
	Disconnect()
}

func (s Scanner) Scan(ctx context.Context, interval time.Duration) {
	s.Logger.Info("scanner started", "interval", interval, "namespace", s.Namespace, "selector", s.LabelSelector)
	defer s.Logger.Info("scanner stopped")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.ScanOnce(ctx); err != nil {
				s.Logger.Error("scan failed", "err", err)
			}
		}
	}
}

func (s Scanner) ScanOnce(ctx context.Context) error {
	defer s.Client.Disconnect()

	pods, err := s.Client.GetPodsForLabelSelector(ctx, s.Namespace, s.LabelSelector)
	if err != nil {
		return fmt.Errorf("GetPodsForLabelSelector: %w", err)
	}

	for _, pod := range s.getNotReady(pods) {
		if err := s.Client.DeletePod(ctx, s.Namespace, pod.GetName()); err != nil {
			s.Logger.Warn("failed to delete pod", "err", err, "name", pod.GetName())
			continue
		}
		s.Logger.Info("pod deleted", "name", pod.GetName())
	}
	return err
}

func (s Scanner) getNotReady(pods []coreV1.Pod) []coreV1.Pod {
	notReady := make([]coreV1.Pod, 0, len(pods))
	for _, pod := range pods {
		l := s.Logger.With("name", pod.GetName())

		l.Debug("checking pod")

		status := getReadyStatus(pod)
		switch status {
		case coreV1.ConditionTrue:
			l.Debug("pod is ready")
		case coreV1.ConditionFalse:
			l.Debug("pod not ready")
			notReady = append(notReady, pod)
		default:
			l.Debug("pod doesn't appear to be running", "status", status)
		}
	}

	return notReady
}

func getReadyStatus(pod coreV1.Pod) coreV1.ConditionStatus {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == coreV1.PodReady {
			return condition.Status
		}
	}
	return ""
}
