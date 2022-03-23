/*
Copyright AppsCode Inc. and Contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mysql_ops

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kubeops.dev/supervisor/apis"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MySQLOpsRequest struct {
	req      *opsapi.MySQLOpsRequest
	timeout  time.Duration
	interval time.Duration
}

var _ apis.OpsRequest = &MySQLOpsRequest{}

func NewMySQLOpsRequest(obj runtime.RawExtension, name string) (*MySQLOpsRequest, error) {
	myOpsReq := &opsapi.MySQLOpsRequest{}
	if err := json.Unmarshal(obj.Raw, myOpsReq); err != nil {
		return nil, err
	}
	myOpsReq.Name = name
	return &MySQLOpsRequest{
		req:      myOpsReq,
		timeout:  30 * time.Minute,
		interval: 5 * time.Second,
	}, nil
}

func (o *MySQLOpsRequest) Execute(ctx context.Context, kc client.Client) error {
	req := o.req.DeepCopy()
	return kc.Create(ctx, req)
}

func (o *MySQLOpsRequest) IsSucceeded(ctx context.Context, kc client.Client) (bool, error) {
	key := client.ObjectKey{Name: o.req.Name, Namespace: o.req.Namespace}
	opsReq := &opsapi.MySQLOpsRequest{}
	if err := kc.Get(ctx, key, opsReq); err != nil {
		return false, err
	}
	if opsReq.Status.Phase == opsapi.OpsRequestPhaseSuccessful {
		return true, nil
	} else if opsReq.Status.Phase == opsapi.OpsRequestPhaseFailed {
		return false, fmt.Errorf("opsrequest failed to execute,reason: %s", o.getOpsRequestFailureReason())
	}
	return false, nil
}

func (o *MySQLOpsRequest) getOpsRequestFailureReason() string {
	for _, con := range o.req.Status.Conditions {
		if con.Status == core.ConditionFalse {
			return con.Reason
		}
	}
	return string(core.ConditionUnknown)
}
