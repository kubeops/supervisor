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
