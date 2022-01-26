package mongodb_ops

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"gomodules.xyz/x/crypto/rand"

	"k8s.io/apimachinery/pkg/runtime"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MongoDBOpsRequest struct {
	req      *opsapi.MongoDBOpsRequest
	timeout  time.Duration
	interval time.Duration
}

func NewMongoDBOpsRequest(obj runtime.RawExtension) (*MongoDBOpsRequest, error) {
	mgOpsReq := &opsapi.MongoDBOpsRequest{}
	if err := json.Unmarshal(obj.Raw, mgOpsReq); err != nil {
		return nil, err
	}
	mgOpsReq.Name = rand.WithUniqSuffix("supervisor")
	return &MongoDBOpsRequest{
		req:      mgOpsReq,
		timeout:  30 * time.Minute,
		interval: 5 * time.Second,
	}, nil
}

func (o *MongoDBOpsRequest) Execute(ctx context.Context, kc client.Client) error {
	req := o.req.DeepCopy()
	return kc.Create(ctx, req)
}

func (o *MongoDBOpsRequest) WaitForOpsRequestToBeCompleted(ctx context.Context, kc client.Client) error {
	key := client.ObjectKey{Name: o.req.Name, Namespace: o.req.Namespace}
	opsReq := &opsapi.MongoDBOpsRequest{}
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
