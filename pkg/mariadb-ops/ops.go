package mariadb_ops

import (
	"context"
	"encoding/json"
	"gomodules.xyz/x/crypto/rand"
	"k8s.io/apimachinery/pkg/runtime"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MariaDBOpsRequest struct {
	req *opsapi.MariaDBOpsRequest
}

func NewMariaDBOpsRequest(obj runtime.RawExtension) (*MariaDBOpsRequest, error) {
	mdOpsReq := &opsapi.MariaDBOpsRequest{}
	if err := json.Unmarshal(obj.Raw, mdOpsReq); err != nil {
		return nil, err
	}
	return &MariaDBOpsRequest{
		req: mdOpsReq,
	}, nil
}

func (o *MariaDBOpsRequest) Execute(kc client.Client) error {
	req := o.req.DeepCopy()
	req.Name = rand.WithUniqSuffix("supervisor")
	return kc.Create(context.TODO(), req)
}
