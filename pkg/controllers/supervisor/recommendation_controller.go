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
	"sync"
	"time"

	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	deadline_manager "kubeops.dev/supervisor/pkg/deadline-manager"
	"kubeops.dev/supervisor/pkg/evaluator"
	"kubeops.dev/supervisor/pkg/maintenance"
	"kubeops.dev/supervisor/pkg/parallelism"
	"kubeops.dev/supervisor/pkg/policy"
	"kubeops.dev/supervisor/pkg/shared"

	"github.com/jonboulle/clockwork"
	"gomodules.xyz/pointer"
	"gomodules.xyz/x/crypto/rand"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	Mutex                  *sync.Mutex
	RequeueAfterDuration   time.Duration
	RetryAfterDuration     time.Duration
	BeforeDeadlineDuration time.Duration
	Clock                  clockwork.Clock
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

	// Ignore any update in the recommendation object if the recommendation is already succeeded
	if obj.Status.Phase == supervisorv1alpha1.Succeeded {
		_, _, err := kmc.PatchStatus(ctx, r.Client, obj, func(obj client.Object, createOp bool) client.Object {
			in := obj.(*supervisorv1alpha1.Recommendation)
			in.Status.ObservedGeneration = in.Generation
			return in
		})
		return ctrl.Result{}, err
	}

	if obj.Status.FailedAttempt > pointer.Int32(obj.Spec.BackoffLimit) {
		_, _, err := kmc.PatchStatus(ctx, r.Client, obj, func(obj client.Object, createOp bool) client.Object {
			in := obj.(*supervisorv1alpha1.Recommendation)
			in.Status.ObservedGeneration = in.Generation
			in.Status.Phase = supervisorv1alpha1.Failed
			in.Status.Reason = supervisorv1alpha1.BackoffLimitExceeded
			return in
		})
		return ctrl.Result{}, err
	}

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

		rcmdMaintenance := maintenance.NewRecommendationMaintenance(ctx, r.Client, obj, r.Clock)
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
	gvk, err := shared.GetGVK(rcmd.Spec.Operation)
	if err != nil {
		return ctrl.Result{RequeueAfter: r.RetryAfterDuration}, err
	}
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	key := client.ObjectKey{Name: rcmd.Status.CreatedOperationRef.Name, Namespace: rcmd.Namespace}
	err = r.Client.Get(ctx, key, obj)
	if err != nil {
		return ctrl.Result{RequeueAfter: r.RetryAfterDuration}, err
	}

	eval := evaluator.New(obj, rcmd.Spec.Rules)
	success, err := eval.EvaluateSuccessfulOperation()
	if err != nil {
		return ctrl.Result{RequeueAfter: r.RetryAfterDuration}, err
	}

	if success == nil {
		return ctrl.Result{RequeueAfter: r.RequeueAfterDuration}, nil
	}

	if pointer.Bool(success) {
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
	} else {
		return r.recordFailedAttempt(ctx, rcmd, errors.New("operation has been failed"))
	}
}

func (r *RecommendationReconciler) runMaintenanceWork(ctx context.Context, rcmd *supervisorv1alpha1.Recommendation) (ctrl.Result, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	runner := parallelism.NewParallelRunner(ctx, r.Client, rcmd)
	maintainParallelism, err := runner.MaintainParallelism()
	if err != nil {
		return ctrl.Result{}, err
	}

	deadlineMgr := deadline_manager.NewManager(rcmd, r.Clock)
	deadlineKnocking := deadlineMgr.IsDeadlineLessThan(r.BeforeDeadlineDuration)

	if !(maintainParallelism || deadlineKnocking) {
		_, _, err = kmc.PatchStatus(ctx, r.Client, rcmd, func(obj client.Object, createOp bool) client.Object {
			in := obj.(*supervisorv1alpha1.Recommendation)
			in.Status.Phase = supervisorv1alpha1.Waiting
			in.Status.Reason = supervisorv1alpha1.WaitingForExecution
			return in
		})
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: r.RequeueAfterDuration}, nil
	}

	// Creating OpsRequest from given raw object
	opsReqName := rand.WithUniqSuffix("supervisor")
	unObj, err := shared.GetUnstructuredObj(rcmd.Spec.Operation)
	if err != nil {
		return r.handleErr(ctx, rcmd, err, supervisorv1alpha1.Failed)
	}
	unObj.SetName(opsReqName)

	err = r.Client.Create(ctx, unObj)
	if err != nil {
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

func (r *RecommendationReconciler) handleErr(ctx context.Context, rcmd *supervisorv1alpha1.Recommendation, err error, phase supervisorv1alpha1.RecommendationPhase) (ctrl.Result, error) {
	_, _, pErr := kmc.PatchStatus(ctx, r.Client, rcmd, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*supervisorv1alpha1.Recommendation)
		in.Status.Phase = phase
		in.Status.Reason = err.Error()
		return in
	})

	return ctrl.Result{RequeueAfter: r.RetryAfterDuration}, pErr
}

func (r *RecommendationReconciler) recordFailedAttempt(ctx context.Context, obj *supervisorv1alpha1.Recommendation, err error) (ctrl.Result, error) {
	_, _, pErr := kmc.PatchStatus(ctx, r.Client, obj, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*supervisorv1alpha1.Recommendation)
		in.Status.Phase = supervisorv1alpha1.Failed
		in.Status.Reason = supervisorv1alpha1.OperationFailed
		in.Status.Conditions = kmapi.SetCondition(in.Status.Conditions, kmapi.Condition{
			Type:               supervisorv1alpha1.SuccessfullyExecutedOperation,
			Status:             core.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: time.Now().UTC()},
			Reason:             err.Error(),
			Message:            err.Error(),
		})
		in.Status.FailedAttempt += 1
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
