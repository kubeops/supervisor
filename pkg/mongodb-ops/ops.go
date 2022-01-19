package mongodb_ops

import (
	"context"
	"encoding/json"
	"gomodules.xyz/x/crypto/rand"
	"k8s.io/apimachinery/pkg/runtime"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MongoDBOpsRequest struct {
	req *opsapi.MongoDBOpsRequest
}

func NewMongoDBOpsRequest(obj runtime.RawExtension) (*MongoDBOpsRequest, error) {
	mgOpsReq := &opsapi.MongoDBOpsRequest{}
	if err := json.Unmarshal(obj.Raw, mgOpsReq); err != nil {
		return nil, err
	}
	return &MongoDBOpsRequest{
		req: mgOpsReq,
	}, nil
}

func (o *MongoDBOpsRequest) Execute(kc client.Client) error {
	req := o.req.DeepCopy()
	req.Name = rand.WithUniqSuffix("supervisor")
	return kc.Create(context.TODO(), req)
}
