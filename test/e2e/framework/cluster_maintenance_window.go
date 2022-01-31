package framework

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
)

func (f *Framework) getDefaultClusterMaintenanceWindowName() string {
	return f.clusterMWName
}

func (f *Framework) CreateDefaultClusterMaintenanceWindow() error {
	clsMW := &api.ClusterMaintenanceWindow{
		ObjectMeta: metav1.ObjectMeta{
			Name: f.getDefaultClusterMaintenanceWindowName(),
		},
		Spec: api.MaintenanceWindowSpec{
			IsDefault: true,
			Days: map[api.DayOfWeek][]api.TimeWindow{
				"Monday": {
					{
						Start: kmapi.NewTime(f.clock.Now()),
						End:   kmapi.NewTime(f.clock.Now().Add(time.Hour)),
					},
				},
				"Tuesday": {
					{
						Start: kmapi.NewTime(f.clock.Now()),
						End:   kmapi.NewTime(f.clock.Now().Add(time.Hour)),
					},
				},
			},
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
