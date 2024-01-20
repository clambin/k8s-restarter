package scanner_test

import (
	"context"
	"errors"
	"github.com/clambin/k8s-restarter/internal/scanner"
	"github.com/clambin/k8s-restarter/internal/scanner/mocks"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"testing"
	"time"
)

func TestScanner_ScanOnce(t *testing.T) {
	tests := []struct {
		name       string
		pods       []coreV1.Pod
		getErr     error
		delErr     error
		wantErr    assert.ErrorAssertionFunc
		wantDelete bool
	}{
		{
			name:    "no pods",
			pods:    []coreV1.Pod{},
			wantErr: assert.NoError,
		},
		{
			name:    "get failed",
			pods:    []coreV1.Pod{},
			getErr:  errors.New("failed"),
			wantErr: assert.Error,
		},
		{
			name: "one not-running pod",
			pods: []coreV1.Pod{{
				ObjectMeta: metaV1.ObjectMeta{Namespace: "media", Name: "foo-1"},
			}},
			wantErr: assert.NoError,
		},

		{
			name: "one running pod",
			pods: []coreV1.Pod{{
				ObjectMeta: metaV1.ObjectMeta{Namespace: "media", Name: "foo-1", UID: "pod-foo-1"},
				Status:     coreV1.PodStatus{Phase: "Running", Conditions: []coreV1.PodCondition{{Type: "Ready", Status: "True"}}},
			}},
			wantErr: assert.NoError,
		},
		{
			name: "one failing pod",
			pods: []coreV1.Pod{{
				ObjectMeta: metaV1.ObjectMeta{Namespace: "media", Name: "foo-bad", UID: "pod-foo-1"},
				Status:     coreV1.PodStatus{Phase: "Running", Conditions: []coreV1.PodCondition{{Type: "Ready", Status: "False"}}},
			}},
			wantErr:    assert.NoError,
			wantDelete: true,
		},
		{
			name: "one failing pod - delete failed",
			pods: []coreV1.Pod{{
				ObjectMeta: metaV1.ObjectMeta{Namespace: "media", Name: "foo-bad", UID: "pod-foo-1"},
				Status:     coreV1.PodStatus{Phase: "Running", Conditions: []coreV1.PodCondition{{Type: "Ready", Status: "False"}}},
			}},
			delErr:     errors.New("failed"),
			wantErr:    assert.NoError,
			wantDelete: true,
		},
		{
			name: "one of many pods failing",
			pods: []coreV1.Pod{{
				ObjectMeta: metaV1.ObjectMeta{Namespace: "media", Name: "foo-bad", UID: "pod-foo-1"},
				Status:     coreV1.PodStatus{Phase: "Running", Conditions: []coreV1.PodCondition{{Type: "Ready", Status: "False"}}},
			}, {
				ObjectMeta: metaV1.ObjectMeta{Namespace: "media", Name: "foo-2", UID: "pod-foo-2", OwnerReferences: []metaV1.OwnerReference{{UID: "rs-foo-1"}}},
				Status:     coreV1.PodStatus{Phase: "Running", Conditions: []coreV1.PodCondition{{Type: "Ready", Status: "True"}}},
			}},
			wantErr:    assert.NoError,
			wantDelete: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			p := mocks.NewPodMan(t)
			p.EXPECT().GetPodsForLabelSelector(ctx, "namespace", "app=foo").Return(tt.pods, tt.getErr).Once()
			if tt.wantDelete {
				p.EXPECT().DeletePod(ctx, "namespace", "foo-bad").Return(tt.delErr).Once()
			}
			p.EXPECT().Disconnect().Once()

			s := scanner.Scanner{
				Client:        p,
				Namespace:     "namespace",
				LabelSelector: "app=foo",
				Logger:        slog.Default()}
			err := s.ScanOnce(context.Background())
			tt.wantErr(t, err)
		})
	}
}

func TestScanner_Scan(t *testing.T) {
	c := mocks.NewPodMan(t)
	s := scanner.Scanner{
		Namespace:     "namespace",
		LabelSelector: "app=foo",
		Client:        c,
		Logger:        slog.Default(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan struct{})
	c.EXPECT().GetPodsForLabelSelector(ctx, s.Namespace, s.LabelSelector).Return(nil, errors.New("fail"))
	c.EXPECT().Disconnect().Run(func() {
		ch <- struct{}{}
	})

	go s.Scan(ctx, 10*time.Millisecond)
	<-ch
}
