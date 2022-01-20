package redis_ops

import (
	"context"
	"encoding/json"
	"gomodules.xyz/x/crypto/rand"
	"k8s.io/apimachinery/pkg/runtime"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RedisOpsRequest struct {
	req *opsapi.RedisOpsRequest
}

func NewRedisOpsRequest(obj runtime.RawExtension) (*RedisOpsRequest, error) {
	rdOpsReq := &opsapi.RedisOpsRequest{}
	if err := json.Unmarshal(obj.Raw, rdOpsReq); err != nil {
		return nil, err
	}
	return &RedisOpsRequest{
		req: rdOpsReq,
	}, nil
}

func (o *RedisOpsRequest) Execute(kc client.Client) error {
	req := o.req.DeepCopy()
	req.Name = rand.WithUniqSuffix("supervisor")
	return kc.Create(context.TODO(), req)
}
