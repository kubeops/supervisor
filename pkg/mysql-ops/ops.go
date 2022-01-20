package mysql_ops

import (
	"context"
	"encoding/json"
	"gomodules.xyz/x/crypto/rand"
	"k8s.io/apimachinery/pkg/runtime"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MySQLOpsRequest struct {
	req *opsapi.MySQLOpsRequest
}

func NewMySQLOpsRequest(obj runtime.RawExtension) (*MySQLOpsRequest, error) {
	myOpsReq := &opsapi.MySQLOpsRequest{}
	if err := json.Unmarshal(obj.Raw, myOpsReq); err != nil {
		return nil, err
	}
	return &MySQLOpsRequest{
		req: myOpsReq,
	}, nil
}

func (o *MySQLOpsRequest) Execute(kc client.Client) error {
	req := o.req.DeepCopy()
	req.Name = rand.WithUniqSuffix("supervisor")
	return kc.Create(context.TODO(), req)
}
