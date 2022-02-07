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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
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

	return f.kc.Create(f.ctx, clsMW)
}

func (f *Framework) DeleteDefaultClusterMaintenanceWindow() error {
	clsMW := &api.ClusterMaintenanceWindow{
		ObjectMeta: metav1.ObjectMeta{
			Name: f.getDefaultClusterMaintenanceWindowName(),
		},
	}
	return f.kc.Delete(f.ctx, clsMW)
}
