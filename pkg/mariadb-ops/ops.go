package mariadb_ops

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

type MariaDBOpsRequest struct {
	req      *opsapi.MariaDBOpsRequest
	timeout  time.Duration
	interval time.Duration
}

func NewMariaDBOpsRequest(obj runtime.RawExtension) (*MariaDBOpsRequest, error) {
	mdOpsReq := &opsapi.MariaDBOpsRequest{}
	if err := json.Unmarshal(obj.Raw, mdOpsReq); err != nil {
		return nil, err
	}
	mdOpsReq.Name = rand.WithUniqSuffix("supervisor")
	return &MariaDBOpsRequest{
		req:      mdOpsReq,
		timeout:  30 * time.Minute,
		interval: 5 * time.Second,
	}, nil
}

func (o *MariaDBOpsRequest) Execute(ctx context.Context, kc client.Client) error {
	req := o.req.DeepCopy()
	return kc.Create(context.TODO(), req)
}

func (o *MariaDBOpsRequest) WaitForOpsRequestToBeCompleted(ctx context.Context, kc client.Client) error {
	key := client.ObjectKey{Name: o.req.Name, Namespace: o.req.Namespace}
	opsReq := &opsapi.MariaDBOpsRequest{}
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
