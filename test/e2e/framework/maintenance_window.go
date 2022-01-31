package framework

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/util/wait"

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
	err := f.kc.Create(f.ctx, mw)
	if err != nil {
		return err
	}

	return wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		mwObj := &api.MaintenanceWindow{}
		key := client.ObjectKey{Namespace: mw.Namespace, Name: mw.Name}

		if err := f.kc.Get(f.ctx, key, mwObj); err != nil {
			return false, client.IgnoreNotFound(err)
		}

		return true, nil
	})
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
