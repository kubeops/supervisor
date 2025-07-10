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

package maintenance

import (
	"context"
	"errors"
	"fmt"
	"k8s.io/klog/v2"
	"time"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"

	"github.com/jonboulle/clockwork"
	"gomodules.xyz/pointer"
	kmapi "kmodules.xyz/client-go/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RecommendationMaintenance struct {
	ctx   context.Context
	kc    client.Client
	rcmd  *api.Recommendation
	clock clockwork.Clock
}

func NewRecommendationMaintenance(ctx context.Context, kc client.Client, rcmd *api.Recommendation, clock clockwork.Clock) *RecommendationMaintenance {
	return &RecommendationMaintenance{
		ctx:   ctx,
		kc:    kc,
		rcmd:  rcmd,
		clock: clock,
	}
}

func (r *RecommendationMaintenance) IsMaintenanceTime() (bool, error) {
	aw := r.rcmd.Status.ApprovedWindow
	if aw != nil && aw.Window == api.Immediate {
		return true, nil
	} else if aw != nil && aw.Window == api.SpecificDates {
		if len(aw.Dates) == 0 {
			return false, errors.New("WindowType is SpecificDates but no DateWindow is provided")
		}
		if r.isMaintenanceDateWindow(aw.Dates) {
			return true, nil
		}
		if r.isMaintenanceDateWindowPassed(aw.Dates) {
			return false, errors.New("given date windows have been already passed")
		}
		return false, nil
	}

	mwList, err := r.getAvailableMaintenanceWindowList()
	if err != nil {
		return false, err
	}
	if len(mwList.Items) == 0 {
		return false, errors.New("no available MaintenanceWindow is found")
	}
	klog.Infof("Found available maintenance windows..........: %s", mwList.Items[0].Name)
	mwPassedFlag := true

	for _, mw := range mwList.Items {
		if mw.Spec.Days != nil {
			mwPassedFlag = false
		}
		loc, err := getLocation(mw.Spec.Timezone)
		if err != nil {
			return false, err
		}
		day := getCurrentDay(r.clock, loc)

		mTimes, found := mw.Spec.Days[api.DayOfWeek(day)]
		klog.Infof("check loc, day, times.....")
		fmt.Println(loc, day, mTimes)
		if found {
			if r.isMaintenanceTimeWindow(mTimes, loc) {
				return true, nil
			}
		}

		if r.isMaintenanceDateWindow(mw.Spec.Dates) {
			return true, nil
		} else if mwPassedFlag && !r.isMaintenanceDateWindowPassed(mw.Spec.Dates) {
			mwPassedFlag = false
		}
	}

	if mwPassedFlag {
		return false, errors.New("invalid Maintenance Window: given date windows have been already passed")
	}

	return false, nil
}

func (r *RecommendationMaintenance) getDefaultMaintenanceWindow() (*api.MaintenanceWindow, error) {
	mwList := &api.MaintenanceWindowList{}
	if err := r.kc.List(r.ctx, mwList, client.InNamespace(r.rcmd.Namespace), client.MatchingFields{
		api.DefaultMaintenanceWindowKey: "true",
	}); err != nil {
		return nil, err
	}

	if len(mwList.Items) > 1 {
		return nil, fmt.Errorf("can't get default Maintenance window, expect one default maintenance window but got %v", len(mwList.Items))
	} else if len(mwList.Items) == 0 {
		return nil, nil
	}
	return &mwList.Items[0], nil
}

func (r *RecommendationMaintenance) getDefaultClusterMaintenanceWindow() (*api.MaintenanceWindow, error) {
	clusterMWList := &api.ClusterMaintenanceWindowList{}
	if err := r.kc.List(r.ctx, clusterMWList, client.MatchingFields{
		api.DefaultClusterMaintenanceWindowKey: "true",
	}); err != nil {
		return nil, err
	}

	if len(clusterMWList.Items) > 1 {
		return nil, fmt.Errorf("can't get default Maintenance window, expect one default maintenance window but got %v", len(clusterMWList.Items))
	} else if len(clusterMWList.Items) == 0 {
		return nil, nil
	}

	mw := &api.MaintenanceWindow{
		Spec:   clusterMWList.Items[0].Spec,
		Status: clusterMWList.Items[0].Status,
	}
	return mw, nil
}

func (r *RecommendationMaintenance) getMaintenanceWindow(key client.ObjectKey) (*api.MaintenanceWindow, error) {
	mw := &api.MaintenanceWindow{}
	if key.Namespace == "" {
		key.Namespace = r.rcmd.Namespace
	}
	if err := r.kc.Get(r.ctx, key, mw); err != nil {
		return nil, err
	}
	return mw, nil
}

func (r *RecommendationMaintenance) getMaintenanceWindows() (*api.MaintenanceWindowList, error) {
	mwList := &api.MaintenanceWindowList{}
	if err := r.kc.List(r.ctx, mwList, client.InNamespace(r.rcmd.Namespace)); err != nil {
		return nil, err
	}
	return mwList, nil
}

func (r *RecommendationMaintenance) getMWListFromClusterMWList() (*api.MaintenanceWindowList, error) {
	clusterMWList := &api.ClusterMaintenanceWindowList{}
	if err := r.kc.List(r.ctx, clusterMWList); err != nil {
		return nil, err
	}
	mwList := &api.MaintenanceWindowList{}
	for _, cMW := range clusterMWList.Items {
		mw := api.MaintenanceWindow{
			Spec:   cMW.Spec,
			Status: cMW.Status,
		}
		mwList.Items = append(mwList.Items, mw)
	}
	return mwList, nil
}

func (r *RecommendationMaintenance) isMaintenanceDateWindow(dates []api.DateWindow) bool {
	for _, d := range dates {
		start := d.Start.UTC().Unix()
		end := d.End.UTC().Unix()
		now := r.clock.Now().UTC().Unix()

		if now >= start && now <= end {
			return true
		}
	}
	return false
}

func (r *RecommendationMaintenance) isMaintenanceDateWindowPassed(dates []api.DateWindow) bool {
	for _, d := range dates {
		end := d.End.UTC().Unix()
		now := r.clock.Now().UTC().Unix()

		if now <= end {
			return false
		}
	}
	return true
}

func (r *RecommendationMaintenance) isMaintenanceTimeWindow(timeWindows []api.TimeWindow, location *time.Location) bool {
	for _, tw := range timeWindows {
		now := kmapi.NewTime(r.clock.Now().In(location))
		start := kmapi.NewTime(tw.Start.Time)
		end := kmapi.NewTime(tw.End.Time)

		if now.Before(&end) && start.Before(&now) {
			return true
		}
	}
	return false
}

func (r *RecommendationMaintenance) getAvailableMaintenanceWindowList() (*api.MaintenanceWindowList, error) {
	aw := r.rcmd.Status.ApprovedWindow
	mwList := &api.MaintenanceWindowList{}
	if aw == nil {
		mw, err := r.getDefaultMaintenanceWindow()
		if err != nil {
			return nil, err
		}
		if mw != nil {
			mwList.Items = append(mwList.Items, *mw)
		}

		if len(mwList.Items) == 0 {
			cMW, err := r.getDefaultClusterMaintenanceWindow()
			if err != nil {
				return nil, err
			}
			if cMW != nil {
				mwList.Items = append(mwList.Items, *cMW)
			}
		}
	} else if aw.MaintenanceWindow != nil {
		mw, err := r.getMaintenanceWindow(client.ObjectKey{Namespace: aw.MaintenanceWindow.Namespace, Name: aw.MaintenanceWindow.Name})
		if err != nil {
			return nil, err
		}
		mwList.Items = append(mwList.Items, *mw)
	} else if aw.Window == api.NextAvailable {
		var err error
		mwList, err = r.getMaintenanceWindows()
		if err != nil {
			return nil, err
		}
		if len(mwList.Items) == 0 {
			cMWList, err := r.getMWListFromClusterMWList()
			if err != nil {
				return nil, err
			}
			mwList.Items = append(mwList.Items, cMWList.Items...)
		}
	}
	return mwList, nil
}

func getCurrentDay(clock clockwork.Clock, loc *time.Location) string {
	return clock.Now().In(loc).Weekday().String()
}

func getLocation(location *string) (*time.Location, error) {
	return time.LoadLocation(pointer.String(location))
}
