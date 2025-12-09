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

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	kmc "kmodules.xyz/client-go/client"
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

	mw := &api.MaintenanceWindow{}
	if err := r.Get(ctx, key, mw); err != nil {
		klog.Infof("MaintenanceWindow %q doesn't exist anymore", key.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if mw.Spec.IsDefault {
		if _, ok := mw.Annotations[api.DefaultMaintenanceWindowKey]; !ok {
			_, err := kmc.CreateOrPatch(ctx, r.Client, mw, func(obj client.Object, createOp bool) client.Object {
				in := obj.(*api.MaintenanceWindow)
				if in.Annotations == nil {
					in.Annotations = make(map[string]string)
				}
				in.Annotations[api.DefaultMaintenanceWindowKey] = "true"
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
		For(&api.MaintenanceWindow{}).
		Complete(r)
}
