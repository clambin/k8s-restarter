package scanner

import (
	"context"
	"log/slog"
	"time"
)

type Scanner struct {
	Reaper
	Namespace     string
	LabelSelector string
	Logger        *slog.Logger
}

type Reaper interface {
	Reap(context.Context, string, string) error
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
	return s.Reap(ctx, s.Namespace, s.LabelSelector)
}
