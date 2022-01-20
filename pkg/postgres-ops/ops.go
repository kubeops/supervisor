package postgres_ops

import (
	"context"
	"encoding/json"
	"gomodules.xyz/x/crypto/rand"
	"k8s.io/apimachinery/pkg/runtime"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PostgresOpsRequest struct {
	req *opsapi.PostgresOpsRequest
}

func NewPostgresOpsRequest(obj runtime.RawExtension) (*PostgresOpsRequest, error) {
	pgOpsReq := &opsapi.PostgresOpsRequest{}
	if err := json.Unmarshal(obj.Raw, pgOpsReq); err != nil {
		return nil, err
	}
	return &PostgresOpsRequest{
		req: pgOpsReq,
	}, nil
}

func (o *PostgresOpsRequest) Execute(kc client.Client) error {
	req := o.req.DeepCopy()
	req.Name = rand.WithUniqSuffix("supervisor")
	return kc.Create(context.TODO(), req)
}
