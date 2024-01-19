package reaper

import (
	"context"
	"errors"
	"github.com/clambin/k8s-restarter/internal/reaper/mocks"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestReaper_Reap(t *testing.T) {
	tests := []struct {
		name    string
		pods    []coreV1.Pod
		getErr  error
		delErr  error
		wantErr assert.ErrorAssertionFunc
		want    int
	}{
		{
			name:    "no pods",
			pods:    []coreV1.Pod{},
			wantErr: assert.NoError,
			want:    0,
		},
		{
			name:    "get failed",
			pods:    []coreV1.Pod{},
			getErr:  errors.New("failed"),
			wantErr: assert.Error,
			want:    0,
		},
		{
			name: "one not-running pod",
			pods: []coreV1.Pod{{
				ObjectMeta: metaV1.ObjectMeta{Namespace: "media", Name: "foo-1", UID: "pod-foo-1"},
			}},
			wantErr: assert.NoError,
			want:    0,
		},

		{
			name: "one running pod",
			pods: []coreV1.Pod{{
				ObjectMeta: metaV1.ObjectMeta{Namespace: "media", Name: "foo-1", UID: "pod-foo-1"},
				Status:     coreV1.PodStatus{Phase: "Running", Conditions: []coreV1.PodCondition{{Type: "Ready", Status: "True"}}},
			}},
			wantErr: assert.NoError,
			want:    0,
		},
		{
			name: "one failing pod",
			pods: []coreV1.Pod{{
				ObjectMeta: metaV1.ObjectMeta{Namespace: "media", Name: "foo-bad", UID: "pod-foo-1"},
				Status:     coreV1.PodStatus{Phase: "Running", Conditions: []coreV1.PodCondition{{Type: "Ready", Status: "False"}}},
			}},
			wantErr: assert.NoError,
			want:    1,
		},
		{
			name: "one failing pod - delete failed",
			pods: []coreV1.Pod{{
				ObjectMeta: metaV1.ObjectMeta{Namespace: "media", Name: "foo-bad", UID: "pod-foo-1"},
				Status:     coreV1.PodStatus{Phase: "Running", Conditions: []coreV1.PodCondition{{Type: "Ready", Status: "False"}}},
			}},
			delErr:  errors.New("failed"),
			wantErr: assert.NoError,
			want:    0,
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
			wantErr: assert.NoError,
			want:    1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			p := mocks.NewPodMan(t)
			p.EXPECT().GetPodsForLabelSelector(ctx, "namespace", "app=foo").Return(tt.pods, tt.getErr)
			p.EXPECT().DeletePod(ctx, "namespace", "foo-bad").Return(tt.delErr).Maybe()

			r := Reaper{Client: p}
			count, err := r.Reap(context.Background(), "namespace", "app=foo")
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, count)
		})
	}
}
