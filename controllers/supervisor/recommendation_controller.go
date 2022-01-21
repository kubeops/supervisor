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
	"os"
	"time"

	"kubeops.dev/supervisor/pkg/maintenance"

	"github.com/jonboulle/clockwork"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	kmc "kmodules.xyz/client-go/client"
	meta_util "kmodules.xyz/client-go/meta"
	"kubeops.dev/supervisor/apis"
	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"kubeops.dev/supervisor/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	RequeueAfterDuration = time.Minute
)

// RecommendationReconciler reconciles a Recommendation object
type RecommendationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=supervisor.appscode.com,resources=recommendations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=supervisor.appscode.com,resources=recommendations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=supervisor.appscode.com,resources=recommendations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *RecommendationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	key := req.NamespacedName
	klog.Info("got event for Recommendation: ", key.String())

	obj := &supervisorv1alpha1.Recommendation{}
	if err := r.Client.Get(ctx, key, obj); err != nil {
		klog.Infof("Recommendation %q doesn't exist anymore", key.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	rcmd := obj.DeepCopy()

	if len(rcmd.Status.Conditions) == 0 {
		err := r.addCondition(ctx, rcmd, kmapi.Condition{
			Type:               "Create",
			Status:             core.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: time.Now().UTC()},
			Reason:             supervisorv1alpha1.RecommendationSuccessfullyCreated,
			Message:            "Recommendation has created successfully",
		})
		return ctrl.Result{}, err
	}

	if rcmd.Status.ApprovalStatus == supervisorv1alpha1.ApprovalApproved {
		if isConditionTrue(rcmd.Status.Conditions, supervisorv1alpha1.OpsRequestSuccessfullyCreated) {
			return ctrl.Result{}, nil
		}

		rcmdMaintenance := maintenance.NewRecommendationMaintenance(ctx, r.Client, rcmd, GetClock())
		ok, err := rcmdMaintenance.IsMaintenanceTime()
		if err != nil {
			return ctrl.Result{}, err
		}
		if ok {
			exeObj, err := shared.GetOpsRequestObject(rcmd.Spec.Operation)
			if err != nil {
				return ctrl.Result{}, err
			}

			if err := r.executeOpsRequest(exeObj); err != nil {
				return ctrl.Result{}, err
			}

			_, err = r.updateObservedGeneration(ctx, rcmd)
			if err != nil {
				return ctrl.Result{}, err
			}

			err = r.addCondition(ctx, rcmd, kmapi.Condition{
				Type:               "Create",
				Status:             core.ConditionTrue,
				LastTransitionTime: metav1.Time{Time: time.Now().UTC()},
				Reason:             supervisorv1alpha1.OpsRequestSuccessfullyCreated,
				Message:            "OpsRequest is successfully created",
			})

			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: RequeueAfterDuration}, nil
	}

	return ctrl.Result{}, nil
}

func (r *RecommendationReconciler) executeOpsRequest(e apis.OpsRequest) error {
	return e.Execute(r.Client)
}

func (r *RecommendationReconciler) updateObservedGeneration(ctx context.Context, rcmd *supervisorv1alpha1.Recommendation) (*supervisorv1alpha1.Recommendation, error) {
	obj, _, err := kmc.PatchStatus(ctx, r.Client, rcmd, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*supervisorv1alpha1.Recommendation)
		in.Status.ObservedGeneration = in.Generation
		return in
	})
	if err != nil {
		return nil, err
	}
	rcmd = obj.(*supervisorv1alpha1.Recommendation)
	return rcmd, nil
}

func isConditionTrue(conditions []kmapi.Condition, key string) bool {
	for _, c := range conditions {
		if c.Reason == key && c.Status == core.ConditionTrue {
			return true
		}
	}
	return false
}

func (r *RecommendationReconciler) addCondition(ctx context.Context, rcmd *supervisorv1alpha1.Recommendation, condition kmapi.Condition) error {
	_, _, err := kmc.PatchStatus(ctx, r.Client, rcmd, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*supervisorv1alpha1.Recommendation)
		in.Status.Conditions = append(in.Status.Conditions, condition)
		return in
	})

	return err
}

func GetClock() clockwork.Clock {
	if os.Getenv("APPSCODE_SUPERVISOR_TEST") == "TRUE" {
		return clockwork.NewFakeClock()
	}
	return clockwork.NewRealClock()
}

// SetupWithManager sets up the controller with the Manager.
func (r *RecommendationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&supervisorv1alpha1.Recommendation{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return !meta_util.MustAlreadyReconciled(e.Object)
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return !meta_util.MustAlreadyReconciled(e.ObjectNew)
			},
		}).
		Complete(r)
}
