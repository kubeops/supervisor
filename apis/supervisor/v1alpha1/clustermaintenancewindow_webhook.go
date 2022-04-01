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

package v1alpha1

import (
	"context"
	"fmt"

	"gomodules.xyz/pointer"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var (
	clustermaintenancewindowlog = logf.Log.WithName("clustermaintenancewindow-resource")
	cmwClient                   client.Client
)

func (r *ClusterMaintenanceWindow) SetupWebhookWithManager(mgr ctrl.Manager) error {
	cmwClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-supervisor-appscode-com-v1alpha1-clustermaintenancewindow,mutating=true,failurePolicy=fail,sideEffects=None,groups=supervisor.appscode.com,resources=clustermaintenancewindows,verbs=create;update,versions=v1alpha1,name=mclustermaintenancewindow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &ClusterMaintenanceWindow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ClusterMaintenanceWindow) Default() {
	clustermaintenancewindowlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-supervisor-appscode-com-v1alpha1-clustermaintenancewindow,mutating=false,failurePolicy=fail,sideEffects=None,groups=supervisor.appscode.com,resources=clustermaintenancewindows,verbs=create;update,versions=v1alpha1,name=vclustermaintenancewindow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &ClusterMaintenanceWindow{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterMaintenanceWindow) ValidateCreate() error {
	clustermaintenancewindowlog.Info("validate create", "name", r.Name)

	return r.validateClusterMaintenanceWindow(context.TODO())
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterMaintenanceWindow) ValidateUpdate(old runtime.Object) error {
	clustermaintenancewindowlog.Info("validate update", "name", r.Name)

	return r.validateClusterMaintenanceWindow(context.TODO())
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterMaintenanceWindow) ValidateDelete() error {
	clustermaintenancewindowlog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *ClusterMaintenanceWindow) validateClusterMaintenanceWindow(ctx context.Context) error {
	if r.Spec.TimeZone != nil {
		if err := validateTimeZone(pointer.String(r.Spec.TimeZone)); err != nil {
			return err
		}
	}
	if !r.Spec.IsDefault {
		return nil
	}
	cmwList := &ClusterMaintenanceWindowList{}
	if err := cmwClient.List(ctx, cmwList, client.MatchingFields{
		DefaultClusterMaintenanceWindowKey: "true",
	}); err != nil {
		return err
	}

	if len(cmwList.Items) == 0 {
		return nil
	} else if len(cmwList.Items) == 1 && cmwList.Items[0].Name == r.Name { // to handle update operation of the Default ClusterMaintenanceWindow
		return nil
	} else {
		return fmt.Errorf("%q is already present as default ClusterMaintenanceWindow", cmwList.Items[0].Name)
	}
}
