package reaper

import (
	"context"
	coreV1 "k8s.io/api/core/v1"
	"log/slog"
)

type Reaper struct {
	Client PodMan
}

type PodMan interface {
	GetPodsForLabelSelector(ctx context.Context, namespace string, labelSelector string) ([]coreV1.Pod, error)
	DeletePod(ctx context.Context, namespace, name string) error
}

func (r *Reaper) Reap(ctx context.Context, namespace, labelSelector string) (int, error) {
	pods, err := r.Client.GetPodsForLabelSelector(ctx, namespace, labelSelector)
	if err != nil {
		return 0, err
	}
	slog.Debug("found pods", "count", len(pods))

	var deleted int
	for _, pod := range r.getNotReady(pods) {
		if err := r.Client.DeletePod(ctx, namespace, pod.GetName()); err != nil {
			slog.Warn("failed to delete pod", "err", err, "name", pod.GetName())
			continue
		}

		deleted++
		slog.Info("pod deleted", "name", pod.GetName())
	}

	return deleted, err
}

func (r *Reaper) getNotReady(pods []coreV1.Pod) []coreV1.Pod {
	notReady := make([]coreV1.Pod, 0, len(pods))
	for _, pod := range pods {
		l := slog.With("name", pod.GetName())

		l.Debug("checking pod")
		var found bool
		var status coreV1.ConditionStatus
		for _, condition := range pod.Status.Conditions {
			if condition.Type == "Ready" {
				status = condition.Status
				found = true
				break
			}
		}

		l.Debug("pod status", "status", string(status))

		if !found {
			l.Debug("pod doesn't appear to be running")
			continue
		}

		if status == "True" {
			l.Debug("pod is ready")
			continue
		}

		l.Debug("pod not ready")
		notReady = append(notReady, pod)
	}

	return notReady
}
