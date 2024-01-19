package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestClient_GetPodsForLabelSelector(t *testing.T) {
	type args struct {
		namespace     string
		labelSelector string
	}
	tests := []struct {
		name      string
		args      args
		objects   []runtime.Object
		wantErr   assert.ErrorAssertionFunc
		wantCount int
	}{
		{
			name: "match",
			args: args{"media", "app=foo"},
			objects: []runtime.Object{&coreV1.PodList{
				TypeMeta: metaV1.TypeMeta{Kind: "pods", APIVersion: "v1"},
				Items: []coreV1.Pod{
					{
						ObjectMeta: metaV1.ObjectMeta{Namespace: "media", Name: "foo-1", UID: "pod-foo-1", Labels: map[string]string{"app": "foo"}},
						Status:     coreV1.PodStatus{Phase: "Running", Conditions: []coreV1.PodCondition{{Type: "Ready", Status: "True"}}},
					},
				},
			}},
			wantErr:   assert.NoError,
			wantCount: 1,
		},
		{
			name: "wrong label",
			args: args{"media", "app=foo"},
			objects: []runtime.Object{&coreV1.PodList{
				TypeMeta: metaV1.TypeMeta{Kind: "pods", APIVersion: "v1"},
				Items: []coreV1.Pod{
					{
						ObjectMeta: metaV1.ObjectMeta{Namespace: "media", Name: "foo-1", UID: "pod-foo-1", Labels: map[string]string{"app": "bar"}},
						Status:     coreV1.PodStatus{Phase: "Running", Conditions: []coreV1.PodCondition{{Type: "Ready", Status: "True"}}},
					},
				},
			}},
			wantErr:   assert.NoError,
			wantCount: 0,
		},
		{
			name: "wrong namespace",
			args: args{"media", "app=foo"},
			objects: []runtime.Object{&coreV1.PodList{
				TypeMeta: metaV1.TypeMeta{Kind: "pods", APIVersion: "v1"},
				Items: []coreV1.Pod{
					{
						ObjectMeta: metaV1.ObjectMeta{Namespace: "other", Name: "foo-1", UID: "pod-foo-1", Labels: map[string]string{"app": "foo"}},
						Status:     coreV1.PodStatus{Phase: "Running", Conditions: []coreV1.PodCondition{{Type: "Ready", Status: "True"}}},
					},
				},
			}},
			wantErr:   assert.NoError,
			wantCount: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			c := Client{Connector: func() (kubernetes.Interface, error) {
				return fake.NewSimpleClientset(tt.objects...), nil
			}}
			pods, err := c.GetPodsForLabelSelector(ctx, tt.args.namespace, tt.args.labelSelector)
			tt.wantErr(t, err)
			if err != nil {
				return
			}

			assert.Len(t, pods, tt.wantCount)
			for _, pod := range pods {
				assert.NoError(t, c.DeletePod(ctx, tt.args.namespace, pod.GetName()))
			}
		})
	}
}
