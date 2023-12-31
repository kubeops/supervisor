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
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (f *Framework) getDefaultClusterMaintenanceWindowName() string {
	return f.clusterMWName
}

func (f *Framework) CreateDefaultClusterMaintenanceWindow(days map[api.DayOfWeek][]api.TimeWindow, dates []api.DateWindow) error {
	clsMW := &api.ClusterMaintenanceWindow{
		ObjectMeta: metav1.ObjectMeta{
			Name: f.getDefaultClusterMaintenanceWindowName(),
		},
		Spec: api.MaintenanceWindowSpec{
			IsDefault: true,
			Days:      days,
			Dates:     dates,
		},
	}

	err := f.kc.Create(f.ctx, clsMW)
	if err != nil {
		return err
	}

	return wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
		cmwObj := &api.ClusterMaintenanceWindow{}
		key := client.ObjectKey{Name: clsMW.Name}

		if err := f.kc.Get(f.ctx, key, cmwObj); err != nil {
			return false, client.IgnoreNotFound(err)
		}

		return true, nil
	})
}

func (f *Framework) GetClusterMaintenanceWindow(key client.ObjectKey) (*api.ClusterMaintenanceWindow, error) {
	cmw := &api.ClusterMaintenanceWindow{}
	err := f.kc.Get(f.ctx, key, cmw)
	if err != nil {
		return nil, err
	}
	return cmw, nil
}

func (f *Framework) GetDefaultClusterMaintenanceWindow() (*api.ClusterMaintenanceWindow, error) {
	key := client.ObjectKey{Name: f.getDefaultClusterMaintenanceWindowName()}
	return f.GetClusterMaintenanceWindow(key)
}

func (f *Framework) DeleteDefaultClusterMaintenanceWindow() error {
	clsMW := &api.ClusterMaintenanceWindow{
		ObjectMeta: metav1.ObjectMeta{
			Name: f.getDefaultClusterMaintenanceWindowName(),
		},
	}
	return f.kc.Delete(f.ctx, clsMW)
}
