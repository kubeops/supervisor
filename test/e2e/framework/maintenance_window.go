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

func (f *Framework) CreateMaintenanceWindow(days map[api.DayOfWeek][]api.TimeWindow, dates []api.DateWindow) (*api.MaintenanceWindow, error) {
	mw := &api.MaintenanceWindow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("supervisor-mw"),
			Namespace: f.namespace,
		},
		Spec: api.MaintenanceWindowSpec{
			Dates: dates,
			Days:  days,
		},
	}
	err := f.kc.Create(f.ctx, mw)
	if err != nil {
		return nil, err
	}

	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		mwObj := &api.MaintenanceWindow{}
		key := client.ObjectKey{Namespace: mw.Namespace, Name: mw.Name}

		if err := f.kc.Get(f.ctx, key, mwObj); err != nil {
			return false, client.IgnoreNotFound(err)
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return mw, nil
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

func (f *Framework) GetAllDayOfWeekTimeWindow() map[api.DayOfWeek][]api.TimeWindow {
	return map[api.DayOfWeek][]api.TimeWindow{
		api.Monday: {
			{
				Start: kmapi.Date(0, 0, 0),
				End:   kmapi.Date(23, 59, 59),
			},
		},
		api.Tuesday: {
			{
				Start: kmapi.Date(0, 0, 0),
				End:   kmapi.Date(23, 59, 59),
			},
		},
		api.Wednesday: {
			{
				Start: kmapi.Date(0, 0, 0),
				End:   kmapi.Date(23, 59, 59),
			},
		},
		api.Thursday: {
			{
				Start: kmapi.Date(0, 0, 0),
				End:   kmapi.Date(23, 59, 59),
			},
		},
		api.Friday: {
			{
				Start: kmapi.Date(0, 0, 0),
				End:   kmapi.Date(23, 59, 59),
			},
		},
		api.Saturday: {
			{
				Start: kmapi.Date(0, 0, 0),
				End:   kmapi.Date(23, 59, 59),
			},
		},
		api.Sunday: {
			{
				Start: kmapi.Date(0, 0, 0),
				End:   kmapi.Date(23, 59, 59),
			},
		},
	}
}

func (f *Framework) GetCurrentDateWindow() []api.DateWindow {
	return []api.DateWindow{
		{
			Start: metav1.Time{Time: f.clock.Now().Add(time.Minute * 5)},
			End:   metav1.Time{Time: f.clock.Now().Add(time.Hour)},
		},
	}
}

func (f *Framework) GetDateWindowsAfter(after time.Duration, duration time.Duration) []api.DateWindow {
	return []api.DateWindow{
		{
			Start: metav1.Time{Time: f.clock.Now().Add(after)},
			End:   metav1.Time{Time: f.clock.Now().Add(after + duration)},
		},
	}
}

func (f *Framework) GetMaintenanceWindow(key client.ObjectKey) (*api.MaintenanceWindow, error) {
	mw := &api.MaintenanceWindow{}
	err := f.kc.Get(f.ctx, key, mw)
	if err != nil {
		return nil, err
	}
	return mw, nil
}

func (f *Framework) GetDefaultMaintenanceWindow() (*api.MaintenanceWindow, error) {
	key := client.ObjectKey{Name: f.defaultMaintenanceWindowName(), Namespace: f.defaultMaintenanceWindowNamespace()}
	return f.GetMaintenanceWindow(key)
}

func (f *Framework) DeleteMaintenanceWindow(key client.ObjectKey) error {
	mw := &api.MaintenanceWindow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}

	return f.kc.Delete(f.ctx, mw)
}
