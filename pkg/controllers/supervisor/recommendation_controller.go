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
	"sync"
	"time"

	"kubeops.dev/supervisor/apis"
	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	deadline_manager "kubeops.dev/supervisor/pkg/deadline-manager"
	"kubeops.dev/supervisor/pkg/maintenance"
	"kubeops.dev/supervisor/pkg/parallelism"
	"kubeops.dev/supervisor/pkg/policy"
	"kubeops.dev/supervisor/pkg/shared"

	"gomodules.xyz/x/crypto/rand"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	kmc "kmodules.xyz/client-go/client"
	meta_util "kmodules.xyz/client-go/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// RecommendationReconciler reconciles a Recommendation object
type RecommendationReconciler struct {
	client.Client
	Scheme                 *runtime.Scheme
	Mutex                  sync.Mutex
	RequeueAfterDuration   time.Duration
	RetryAfterDuration     time.Duration
	BeforeDeadlineDuration time.Duration
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
	obj = obj.DeepCopy()

	if obj.Status.Phase == "" {
		_, _, err := kmc.PatchStatus(ctx, r.Client, obj, func(obj client.Object, createOp bool) client.Object {
			in := obj.(*supervisorv1alpha1.Recommendation)
			in.Status.Phase = supervisorv1alpha1.Pending
			return in
		})
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if obj.Status.ApprovalStatus == supervisorv1alpha1.ApprovalApproved {
		if obj.Status.Outdated {
			_, _, err := kmc.PatchStatus(ctx, r.Client, obj, func(obj client.Object, createOp bool) client.Object {
				in := obj.(*supervisorv1alpha1.Recommendation)
				in.Status.ObservedGeneration = in.Generation

				return in
			})
			return ctrl.Result{}, err
		}

		if obj.Status.Phase == supervisorv1alpha1.InProgress && obj.Status.CreatedOperationRef != nil {
			return r.checkOpsRequestStatus(ctx, obj)
		}

		rcmdMaintenance := maintenance.NewRecommendationMaintenance(ctx, r.Client, obj)
		isMaintenanceTime, err := rcmdMaintenance.IsMaintenanceTime()
		if err != nil {
			return r.handleErr(ctx, obj, err, supervisorv1alpha1.Pending)
		}

		if !isMaintenanceTime {
			if obj.Status.Phase == supervisorv1alpha1.Pending {
				_, _, err := kmc.PatchStatus(ctx, r.Client, obj, func(obj client.Object, createOp bool) client.Object {
					in := obj.(*supervisorv1alpha1.Recommendation)
					in.Status.Phase = supervisorv1alpha1.Waiting
					in.Status.Reason = supervisorv1alpha1.WaitingForMaintenanceWindow
					return in
				})
				if err != nil {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{RequeueAfter: r.RequeueAfterDuration}, nil
		}

		return r.runMaintenanceWork(ctx, obj)
	} else if obj.Status.ApprovalStatus == supervisorv1alpha1.ApprovalRejected {
		_, _, err := kmc.PatchStatus(ctx, r.Client, obj, func(obj client.Object, createOp bool) client.Object {
			in := obj.(*supervisorv1alpha1.Recommendation)
			in.Status.Phase = supervisorv1alpha1.Skipped
			in.Status.Reason = supervisorv1alpha1.RecommendationRejected
			in.Status.ObservedGeneration = in.Generation

			return in
		})
		return ctrl.Result{}, err
	}

	if !obj.Spec.RequireExplicitApproval {
		policyFinder := policy.NewApprovalPolicyFinder(ctx, r.Client, obj)
		approvalPolicy, err := policyFinder.FindApprovalPolicy()
		if err != nil {
			return ctrl.Result{}, err
		}
		if approvalPolicy != nil {
			_, _, err = kmc.PatchStatus(ctx, r.Client, obj, func(obj client.Object, createOp bool) client.Object {
				in := obj.(*supervisorv1alpha1.Recommendation)
				in.Status.ApprovalStatus = supervisorv1alpha1.ApprovalApproved
				in.Status.ApprovedWindow = &supervisorv1alpha1.ApprovedWindow{
					MaintenanceWindow: &approvalPolicy.MaintenanceWindowRef,
				}
				return in
			})
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{RequeueAfter: r.RequeueAfterDuration}, nil
}

func (r *RecommendationReconciler) checkOpsRequestStatus(ctx context.Context, rcmd *supervisorv1alpha1.Recommendation) (ctrl.Result, error) {
	opsReq, err := shared.GetOpsRequestObject(rcmd.Spec.Operation, rcmd.Status.CreatedOperationRef.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	succeeded, err := opsReq.IsSucceeded(ctx, r.Client)
	if err != nil {
		return r.handleErr(ctx, rcmd, err, supervisorv1alpha1.Failed)
	}
	if succeeded {
		_, _, err = kmc.PatchStatus(ctx, r.Client, rcmd, func(obj client.Object, createOp bool) client.Object {
			in := obj.(*supervisorv1alpha1.Recommendation)
			in.Status.Phase = supervisorv1alpha1.Succeeded
			in.Status.Reason = supervisorv1alpha1.SuccessfullyExecutedOperation
			in.Status.Conditions = kmapi.SetCondition(in.Status.Conditions, kmapi.Condition{
				Type:               supervisorv1alpha1.SuccessfullyExecutedOperation,
				Status:             core.ConditionTrue,
				LastTransitionTime: metav1.Time{Time: time.Now().UTC()},
				Reason:             supervisorv1alpha1.SuccessfullyExecutedOperation,
				Message:            "OpsRequest is successfully executed",
			})
			in.Status.ObservedGeneration = in.Generation
			return in
		})
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: r.RequeueAfterDuration}, nil
}

func (r *RecommendationReconciler) runMaintenanceWork(ctx context.Context, rcmd *supervisorv1alpha1.Recommendation) (ctrl.Result, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	runner := parallelism.NewParallelRunner(ctx, r.Client, rcmd)
	maintainParallelism, err := runner.MaintainParallelism()
	if err != nil {
		return ctrl.Result{}, err
	}

	deadlineMgr := deadline_manager.NewManager(rcmd)
	deadlineKnocking := deadlineMgr.IsDeadlineLessThan(r.BeforeDeadlineDuration)

	if !(maintainParallelism || deadlineKnocking) {
		return ctrl.Result{RequeueAfter: r.RequeueAfterDuration}, nil
	}

	opsReqName := rand.WithUniqSuffix("supervisor")
	exeObj, err := shared.GetOpsRequestObject(rcmd.Spec.Operation, opsReqName)
	if err != nil {
		return r.handleErr(ctx, rcmd, err, supervisorv1alpha1.Failed)
	}

	if err := r.executeOpsRequest(ctx, exeObj); err != nil {
		return r.handleErr(ctx, rcmd, err, supervisorv1alpha1.Failed)
	}

	_, _, err = kmc.PatchStatus(ctx, r.Client, rcmd, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*supervisorv1alpha1.Recommendation)
		in.Status.Phase = supervisorv1alpha1.InProgress
		in.Status.Reason = supervisorv1alpha1.StartedExecutingOperation
		in.Status.Conditions = kmapi.SetCondition(in.Status.Conditions, kmapi.Condition{
			Type:               supervisorv1alpha1.SuccessfullyCreatedOperation,
			Status:             core.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: time.Now().UTC()},
			Reason:             supervisorv1alpha1.SuccessfullyCreatedOperation,
			Message:            "OpsRequest is successfully created",
		})
		in.Status.CreatedOperationRef = &core.LocalObjectReference{Name: opsReqName}
		return in
	})
	return ctrl.Result{}, err
}

func (r *RecommendationReconciler) executeOpsRequest(ctx context.Context, e apis.OpsRequest) error {
	return e.Execute(ctx, r.Client)
}

//func (r *RecommendationReconciler) addCondition(ctx context.Context, rcmd *supervisorv1alpha1.Recommendation, condition kmapi.Condition) (*supervisorv1alpha1.Recommendation, error) {
//	obj, _, err := kmc.PatchStatus(ctx, r.Client, rcmd, func(obj client.Object, createOp bool) client.Object {
//		in := obj.(*supervisorv1alpha1.Recommendation)
//		in.Status.Conditions = kmapi.SetCondition(rcmd.Status.Conditions, condition)
//		return in
//	})
//	if err != nil {
//		return nil, err
//	}
//
//	return obj.(*supervisorv1alpha1.Recommendation).DeepCopy(), nil
//}

func (r *RecommendationReconciler) handleErr(ctx context.Context, rcmd *supervisorv1alpha1.Recommendation, err error, phase supervisorv1alpha1.RecommendationPhase) (ctrl.Result, error) {
	_, _, pErr := kmc.PatchStatus(ctx, r.Client, rcmd, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*supervisorv1alpha1.Recommendation)
		in.Status.Phase = phase
		in.Status.Reason = err.Error()
		return in
	})

	return ctrl.Result{RequeueAfter: r.RetryAfterDuration}, pErr
}

// SetupWithManager sets up the controller with the Manager.
func (r *RecommendationReconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
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
		WithOptions(opts).
		Complete(r)
}
