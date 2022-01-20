package elaticsearch_ops

import (
	"context"
	"encoding/json"
	"gomodules.xyz/x/crypto/rand"
	"k8s.io/apimachinery/pkg/runtime"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ESOpsRequest struct {
	req *opsapi.ElasticsearchOpsRequest
}

func NewESOpsRequest(obj runtime.RawExtension) (*ESOpsRequest, error) {
	esOpsReq := &opsapi.ElasticsearchOpsRequest{}
	if err := json.Unmarshal(obj.Raw, esOpsReq); err != nil {
		return nil, err
	}
	return &ESOpsRequest{
		req: esOpsReq,
	}, nil
}

func (o *ESOpsRequest) Execute(kc client.Client) error {
	req := o.req.DeepCopy()
	req.Name = rand.WithUniqSuffix("supervisor")
	return kc.Create(context.TODO(), req)
}
