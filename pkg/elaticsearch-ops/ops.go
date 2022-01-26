package elaticsearch_ops

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gomodules.xyz/x/crypto/rand"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ESOpsRequest struct {
	req      *opsapi.ElasticsearchOpsRequest
	timeout  time.Duration
	interval time.Duration
}

func NewESOpsRequest(obj runtime.RawExtension) (*ESOpsRequest, error) {
	esOpsReq := &opsapi.ElasticsearchOpsRequest{}
	if err := json.Unmarshal(obj.Raw, esOpsReq); err != nil {
		return nil, err
	}
	esOpsReq.Name = rand.WithUniqSuffix("supervisor")
	return &ESOpsRequest{
		req:      esOpsReq,
		timeout:  30 * time.Minute,
		interval: 5 * time.Second,
	}, nil
}

func (o *ESOpsRequest) Execute(ctx context.Context, kc client.Client) error {
	req := o.req.DeepCopy()
	return kc.Create(ctx, req)
}

func (o *ESOpsRequest) WaitForOpsRequestToBeCompleted(ctx context.Context, kc client.Client) error {
	key := client.ObjectKey{Name: o.req.Name, Namespace: o.req.Namespace}
	opsReq := &opsapi.ElasticsearchOpsRequest{}
	return wait.PollImmediate(o.interval, o.timeout, func() (bool, error) {
		if err := kc.Get(ctx, key, opsReq); err != nil {
			return false, err
		}
		if opsReq.Status.Phase == opsapi.OpsRequestPhaseSuccessful {
			return true, nil
		} else if opsReq.Status.Phase == opsapi.OpsRequestPhaseFailed {
			return false, errors.New("ops request is failed")
		}
		return false, nil
	})
}