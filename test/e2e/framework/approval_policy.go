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

package framework

import (
	"time"

	"gomodules.xyz/x/crypto/rand"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kmapi "kmodules.xyz/client-go/api/v1"
	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (f *Framework) CreateNewApprovalPolicy(target []api.TargetRef, mwRef client.ObjectKey) (*api.ApprovalPolicy, error) {
	ap := &api.ApprovalPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("supervisor-ap"),
			Namespace: f.namespace,
		},
		MaintenanceWindowRef: kmapi.TypedObjectReference{
			Namespace: mwRef.Namespace,
			Name:      mwRef.Name,
		},
		Targets: target,
	}
	if err := f.kc.Create(f.ctx, ap); err != nil {
		return nil, err
	}

	err := wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		createdAp := &api.ApprovalPolicy{}
		key := client.ObjectKey{Namespace: ap.Namespace, Name: ap.Name}
		if err := f.kc.Get(f.ctx, key, createdAp); err != nil {
			return false, client.IgnoreNotFound(err)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return ap, nil
}

func (f *Framework) DeleteApprovalPolicy(key client.ObjectKey) error {
	ap := &api.ApprovalPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}

	return f.kc.Delete(f.ctx, ap)
}
