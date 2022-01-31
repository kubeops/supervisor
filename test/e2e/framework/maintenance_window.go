package framework

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
)

func (f *Framework) CreateDefaultMaintenanceWindow() error {
	mw := &api.MaintenanceWindow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.defaultMaintenanceWindowName(),
			Namespace: f.defaultMaintenanceWindowNamespace(),
		},
		Spec: api.MaintenanceWindowSpec{
			IsDefault: true,
			Dates: []api.DateWindow{
				{
					Start: metav1.Time{Time: time.Date(2022, 01, 01, 0, 0, 0, 0, time.UTC)},
					End:   metav1.Time{Time: time.Date(2023, 01, 01, 0, 0, 0, 0, time.UTC)},
				},
			},
		},
	}
	return f.kc.Create(f.ctx, mw)
}

func (f *Framework) defaultMaintenanceWindowName() string {
	return f.name
}

func (f *Framework) defaultMaintenanceWindowNamespace() string {
	return f.namespace
}

func (f *Framework) DeleteDefaultMaintenanceWindow() error {
	mw := &api.MaintenanceWindow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.defaultMaintenanceWindowName(),
			Namespace: f.defaultMaintenanceWindowNamespace(),
		},
	}

	return f.kc.Delete(f.ctx, mw)
}
