package client

import (
	"context"
	"fmt"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sync"
)

type ConnectorFunc func() (kubernetes.Interface, error)

type Client struct {
	Connect ConnectorFunc
	lock    sync.Mutex
	kClient kubernetes.Interface
}

func (c *Client) client() kubernetes.Interface {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.kClient == nil {
		var err error
		if c.kClient, err = c.Connect(); err != nil {
			panic(fmt.Errorf("failed to connect to cluster: %w", err))
		}
	}
	return c.kClient
}

func (c *Client) GetPodsForLabelSelector(ctx context.Context, namespace string, labelSelector string) ([]coreV1.Pod, error) {
	pods, err := c.client().
		CoreV1().
		Pods(namespace).
		List(ctx, metaV1.ListOptions{LabelSelector: labelSelector})
	return pods.Items, err
}

func (c *Client) DeletePod(ctx context.Context, namespace, name string) error {
	return c.client().CoreV1().Pods(namespace).Delete(ctx, name, metaV1.DeleteOptions{})
}

func (c *Client) Disconnect() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.kClient = nil
}
