package maintenance

import (
	"context"
	"errors"
	"fmt"

	"kubeops.dev/supervisor/pkg/shared"

	"github.com/jonboulle/clockwork"
	kmapi "kmodules.xyz/client-go/api/v1"
	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RecommendationMaintenance struct {
	ctx   context.Context
	kc    client.Client
	rcmd  *supervisorv1alpha1.Recommendation
	clock clockwork.Clock
}

func NewRecommendationMaintenance(ctx context.Context, kc client.Client, rcmd *supervisorv1alpha1.Recommendation) *RecommendationMaintenance {
	return &RecommendationMaintenance{
		ctx:   ctx,
		kc:    kc,
		rcmd:  rcmd,
		clock: shared.GetClock(),
	}
}

func (r *RecommendationMaintenance) IsMaintenanceTime() (bool, error) {
	aw := r.rcmd.Status.ApprovedWindow
	if aw != nil && aw.Window == supervisorv1alpha1.Immediately {
		return true, nil
	} else if aw != nil && aw.Window == supervisorv1alpha1.SpecificDates {
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

	day := r.clock.Now().UTC().Weekday().String()

	mwPassedFlag := true

	for _, mw := range mwList.Items {
		if mw.Spec.Days != nil {
			mwPassedFlag = false
		}
		mTimes, found := mw.Spec.Days[supervisorv1alpha1.DayOfWeek(day)]
		if found {
			if r.isMaintenanceTimeWindow(mTimes) {
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
		return false, errors.New("given MaintenanceWindow dates have been already passed")
	}

	return false, nil
}

func (r *RecommendationMaintenance) getDefaultMaintenanceWindow() (*supervisorv1alpha1.MaintenanceWindow, error) {
	mwList := &supervisorv1alpha1.MaintenanceWindowList{}
	if err := r.kc.List(r.ctx, mwList, client.MatchingFields{
		supervisorv1alpha1.DefaultMaintenanceWindowKey: "true",
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

func (r *RecommendationMaintenance) getDefaultClusterMaintenanceWindow() (*supervisorv1alpha1.MaintenanceWindow, error) {
	clusterMWList := &supervisorv1alpha1.ClusterMaintenanceWindowList{}
	if err := r.kc.List(r.ctx, clusterMWList, client.MatchingFields{
		supervisorv1alpha1.DefaultClusterMaintenanceWindowKey: "true",
	}); err != nil {
		return nil, err
	}

	if len(clusterMWList.Items) > 1 {
		return nil, fmt.Errorf("can't get default Maintenance window, expect one default maintenance window but got %v", len(clusterMWList.Items))
	} else if len(clusterMWList.Items) == 0 {
		return nil, nil
	}

	mw := &supervisorv1alpha1.MaintenanceWindow{
		Spec:   clusterMWList.Items[0].Spec,
		Status: clusterMWList.Items[0].Status,
	}
	return mw, nil
}

func (r *RecommendationMaintenance) getMaintenanceWindow(key client.ObjectKey) (*supervisorv1alpha1.MaintenanceWindow, error) {
	mw := &supervisorv1alpha1.MaintenanceWindow{}
	if key.Namespace == "" {
		key.Namespace = r.rcmd.Namespace
	}
	if err := r.kc.Get(r.ctx, key, mw); err != nil {
		return nil, err
	}
	return mw, nil
}

func (r *RecommendationMaintenance) getMaintenanceWindows() (*supervisorv1alpha1.MaintenanceWindowList, error) {
	mwList := &supervisorv1alpha1.MaintenanceWindowList{}
	if err := r.kc.List(r.ctx, mwList, client.InNamespace(r.rcmd.Namespace)); err != nil {
		return nil, err
	}
	return mwList, nil
}

func (r *RecommendationMaintenance) getMWListFromClusterMWList() (*supervisorv1alpha1.MaintenanceWindowList, error) {
	clusterMWList := &supervisorv1alpha1.ClusterMaintenanceWindowList{}
	if err := r.kc.List(r.ctx, clusterMWList); err != nil {
		return nil, err
	}
	mwList := &supervisorv1alpha1.MaintenanceWindowList{}
	for _, cMW := range clusterMWList.Items {
		mw := supervisorv1alpha1.MaintenanceWindow{
			Spec:   cMW.Spec,
			Status: cMW.Status,
		}
		mwList.Items = append(mwList.Items, mw)
	}
	return mwList, nil
}

func (r *RecommendationMaintenance) isMaintenanceDateWindow(dates []supervisorv1alpha1.DateWindow) bool {
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

func (r *RecommendationMaintenance) isMaintenanceDateWindowPassed(dates []supervisorv1alpha1.DateWindow) bool {
	for _, d := range dates {
		end := d.End.UTC().Unix()
		now := r.clock.Now().UTC().Unix()

		if now <= end {
			return false
		}
	}
	return true
}

func (r *RecommendationMaintenance) isMaintenanceTimeWindow(timeWindows []supervisorv1alpha1.TimeWindow) bool {
	for _, tw := range timeWindows {
		now := kmapi.NewTime(r.clock.Now().UTC())
		start := kmapi.NewTime(tw.Start.UTC())
		end := kmapi.NewTime(tw.End.UTC())

		if now.Before(&end) && start.Before(&now) {
			return true
		}
	}
	return false
}

func (r *RecommendationMaintenance) getAvailableMaintenanceWindowList() (*supervisorv1alpha1.MaintenanceWindowList, error) {
	aw := r.rcmd.Status.ApprovedWindow
	mwList := &supervisorv1alpha1.MaintenanceWindowList{}
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
	} else if aw.Window == supervisorv1alpha1.NextAvailable {
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
