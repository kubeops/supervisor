/*
Copyright AppsCode Inc. and Contributors

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

package supervisor

import (
	"context"
	"errors"
	"fmt"

	"github.com/jonboulle/clockwork"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	kmc "kmodules.xyz/client-go/client"
	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MaintenanceWindowReconciler reconciles a MaintenanceWindow object
type MaintenanceWindowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=supervisor.appscode.com,resources=maintenancewindows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=supervisor.appscode.com,resources=maintenancewindows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=supervisor.appscode.com,resources=maintenancewindows/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *MaintenanceWindowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	key := req.NamespacedName
	klog.Info("got event for MaintenanceWindow: ", key.String())

	mw := &supervisorv1alpha1.MaintenanceWindow{}
	if err := r.Client.Get(ctx, key, mw); err != nil {
		klog.Infof("MaintenanceWindow %q doesn't exist anymore", key.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Todo(Pulak): Implement webhook to make sure only one default MaintenanceWindow at a time
	if mw.Spec.IsDefault {
		if _, ok := mw.Annotations[supervisorv1alpha1.DefaultMaintenanceWindowKey]; !ok {
			_, _, err := kmc.CreateOrPatch(ctx, r.Client, mw, func(obj client.Object, createOp bool) client.Object {
				in := obj.(*supervisorv1alpha1.MaintenanceWindow)
				in.Annotations[supervisorv1alpha1.DefaultMaintenanceWindowKey] = "true"
				return in
			})
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MaintenanceWindowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&supervisorv1alpha1.MaintenanceWindow{}).
		Complete(r)
}

func getDefaultMaintenanceWindow(ctx context.Context, kc client.Client) (*supervisorv1alpha1.MaintenanceWindow, error) {
	mwList := &supervisorv1alpha1.MaintenanceWindowList{}
	if err := kc.List(ctx, mwList, client.MatchingFields{
		supervisorv1alpha1.DefaultMaintenanceWindowKey: "true",
	}); err != nil {
		return nil, err
	}

	if len(mwList.Items) != 1 {
		return nil, fmt.Errorf("can't get default Maintenance window, expect one default maintenance window but got %v", len(mwList.Items))
	}
	return &mwList.Items[0], nil
}

func getMaintenanceWindow(ctx context.Context, kc client.Client, key client.ObjectKey) (*supervisorv1alpha1.MaintenanceWindow, error) {
	mw := &supervisorv1alpha1.MaintenanceWindow{}
	if key.Namespace == "" {
		key.Namespace = core.NamespaceDefault
	}
	if err := kc.Get(ctx, key, mw); err != nil {
		return nil, err
	}
	return mw, nil
}

func getMaintenanceWindows(ctx context.Context, kc client.Client, ns string) (*supervisorv1alpha1.MaintenanceWindowList, error) {
	mwList := &supervisorv1alpha1.MaintenanceWindowList{}
	if err := kc.List(ctx, mwList, client.InNamespace(ns)); err != nil {
		return nil, err
	}
	return mwList, nil
}

func isMaintenanceTime(ctx context.Context, kc client.Client, rcmd *supervisorv1alpha1.Recommendation, clock clockwork.Clock) (bool, error) {
	aw := rcmd.Status.ApprovedWindow
	if aw != nil && aw.Window == supervisorv1alpha1.Immediately {
		return true, nil
	} else if aw != nil && aw.Window == supervisorv1alpha1.SpecificDates {
		if len(aw.Dates) == 0 {
			return false, errors.New("WindowType is SpecificDates but no DateWindow is provided")
		}
		if isMaintenanceDateWindow(clock, aw.Dates) {
			return true, nil
		}
		return false, nil
	}

	mwList, err := getAvailableMaintenanceWindowList(ctx, kc, aw, rcmd.Namespace)
	if err != nil {
		return false, err
	}
	day := clock.Now().UTC().Weekday().String()

	for _, mw := range mwList.Items {
		mTimes, found := mw.Spec.Days[supervisorv1alpha1.DayOfWeek(day)]
		if found {
			if isMaintenanceTimeWindow(clock, mTimes) {
				return true, nil
			}
		}

		if isMaintenanceDateWindow(clock, mw.Spec.Dates) {
			return true, nil
		}
	}

	return false, nil
}

func isMaintenanceDateWindow(clock clockwork.Clock, dates []supervisorv1alpha1.DateWindow) bool {
	for _, d := range dates {
		start := d.Start.UTC().Unix()
		end := d.End.UTC().Unix()
		now := clock.Now().UTC().Unix()

		if now >= start && now <= end {
			return true
		}
	}
	return false
}

func isMaintenanceTimeWindow(clock clockwork.Clock, timeWindows []supervisorv1alpha1.TimeWindow) bool {
	for _, tw := range timeWindows {
		now := kmapi.NewTime(clock.Now().UTC())
		start := kmapi.NewTime(tw.Start.UTC())
		end := kmapi.NewTime(tw.End.UTC())

		if now.Before(&end) && start.Before(&now) {
			return true
		}
	}
	return false
}

func getAvailableMaintenanceWindowList(ctx context.Context, kc client.Client, aw *supervisorv1alpha1.ApprovedWindow, ns string) (*supervisorv1alpha1.MaintenanceWindowList, error) {
	mwList := &supervisorv1alpha1.MaintenanceWindowList{}
	if aw == nil {
		mw, err := getDefaultMaintenanceWindow(ctx, kc)
		if err != nil {
			return nil, err
		}
		mwList.Items = append(mwList.Items, *mw)
	} else if aw.MaintenanceWindow != nil {
		mw, err := getMaintenanceWindow(ctx, kc, client.ObjectKey{Namespace: aw.MaintenanceWindow.Namespace, Name: aw.MaintenanceWindow.Name})
		if err != nil {
			return nil, err
		}
		mwList.Items = append(mwList.Items, *mw)
	} else if aw.Window == supervisorv1alpha1.NextAvailable {
		var err error
		mwList, err = getMaintenanceWindows(ctx, kc, ns)
		if err != nil {
			return nil, err
		}
	}
	return mwList, nil
}
