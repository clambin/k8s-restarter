package scanner_test

import (
	"context"
	"errors"
	"github.com/clambin/k8s-restarter/internal/scanner"
	"github.com/clambin/k8s-restarter/internal/scanner/mocks"
	"github.com/stretchr/testify/mock"
	"log/slog"
	"testing"
	"time"
)

func TestScanner_Scan(t *testing.T) {
	r := mocks.NewReaper(t)
	s := scanner.Scanner{
		Reaper:        r,
		Namespace:     "namespace",
		LabelSelector: "app=foo",
		Logger:        slog.Default(),
	}

	ch := make(chan struct{})
	r.EXPECT().Reap(mock.Anything, "namespace", "app=foo").RunAndReturn(func(ctx context.Context, s string, s2 string) error {
		ch <- struct{}{}
		return errors.New("fail")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.Scan(ctx, 10*time.Millisecond)
	<-ch
}
