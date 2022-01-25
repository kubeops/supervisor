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

	"k8s.io/klog/v2"
	kmc "kmodules.xyz/client-go/client"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
)

// ClusterMaintenanceWindowReconciler reconciles a ClusterMaintenanceWindow object
type ClusterMaintenanceWindowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=supervisor.appscode.com,resources=clustermaintenancewindows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=supervisor.appscode.com,resources=clustermaintenancewindows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=supervisor.appscode.com,resources=clustermaintenancewindows/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ClusterMaintenanceWindowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	key := req.NamespacedName
	klog.Info("got event for ClusterMaintenanceWindow: ", key.String())

	clusterMW := &supervisorv1alpha1.ClusterMaintenanceWindow{}
	if err := r.Client.Get(ctx, key, clusterMW); err != nil {
		klog.Infof("ClusterMaintenanceWindow %q doesn't exist anymore", key.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Todo(Pulak): Implement webhook to make sure only one default ClusterMaintenanceWindow at a time
	if clusterMW.Spec.IsDefault {
		if _, ok := clusterMW.Annotations[supervisorv1alpha1.DefaultClusterMaintenanceWindowKey]; !ok {
			_, _, err := kmc.CreateOrPatch(ctx, r.Client, clusterMW, func(obj client.Object, createOp bool) client.Object {
				in := obj.(*supervisorv1alpha1.ClusterMaintenanceWindow)
				in.Annotations[supervisorv1alpha1.DefaultClusterMaintenanceWindowKey] = "true"
				return in
			})
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterMaintenanceWindowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&supervisorv1alpha1.ClusterMaintenanceWindow{}).
		Complete(r)
}
