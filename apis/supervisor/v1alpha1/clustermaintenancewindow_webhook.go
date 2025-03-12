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
	"errors"
	"fmt"

	"gomodules.xyz/pointer"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupClusterMaintenanceWindowWebhookWithManager registers the webhook for ClusterMaintenanceWindow in the manager.
func SetupClusterMaintenanceWindowWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&ClusterMaintenanceWindow{}).
		WithValidator(&ClusterMaintenanceWindowCustomWebhook{mgr.GetClient()}).
		WithDefaulter(&ClusterMaintenanceWindowCustomWebhook{mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-supervisor-appscode-com-v1alpha1-clustermaintenancewindow,mutating=true,failurePolicy=fail,sideEffects=None,groups=supervisor.appscode.com,resources=clustermaintenancewindows,verbs=create;update,versions=v1alpha1,name=mclustermaintenancewindow.kb.io,admissionReviewVersions={v1,v1beta1}

// +kubebuilder:object:generate=false
type ClusterMaintenanceWindowCustomWebhook struct {
	DefaultClient client.Client
}

var _ webhook.CustomDefaulter = &ClusterMaintenanceWindowCustomWebhook{}

// log is for logging in this package.
var (
	clustermaintenancewindowlog = logf.Log.WithName("clustermaintenancewindow-resource")
)

var _ webhook.CustomDefaulter = &ClusterMaintenanceWindowCustomWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ClusterMaintenanceWindowCustomWebhook) Default(ctx context.Context, obj runtime.Object) error {
	window, ok := obj.(*ClusterMaintenanceWindow)
	if !ok {
		return fmt.Errorf("expected an ClusterMaintenanceWindow object but got %T", obj)
	}
	clustermaintenancewindowlog.Info("default", "name", window.Name)

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-supervisor-appscode-com-v1alpha1-clustermaintenancewindow,mutating=false,failurePolicy=fail,sideEffects=None,groups=supervisor.appscode.com,resources=clustermaintenancewindows,verbs=create;update,versions=v1alpha1,name=vclustermaintenancewindow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.CustomValidator = &ClusterMaintenanceWindowCustomWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterMaintenanceWindowCustomWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	window, ok := obj.(*ClusterMaintenanceWindow)
	if !ok {
		return nil, fmt.Errorf("expected an ClusterMaintenanceWindow object but got %T", obj)
	}
	clustermaintenancewindowlog.Info("validate create", "name", window.Name)

	return nil, r.validateClusterMaintenanceWindow(context.TODO(), window)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterMaintenanceWindowCustomWebhook) ValidateUpdate(ctx context.Context, old, newObj runtime.Object) (admission.Warnings, error) {
	window, ok := newObj.(*ClusterMaintenanceWindow)
	if !ok {
		return nil, fmt.Errorf("expected an ClusterMaintenanceWindow object but got %T", newObj)
	}
	clustermaintenancewindowlog.Info("validate update", "name", window.Name)

	return nil, r.validateClusterMaintenanceWindow(context.TODO(), window)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterMaintenanceWindowCustomWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	window, ok := obj.(*ClusterMaintenanceWindow)
	if !ok {
		return nil, fmt.Errorf("expected an ClusterMaintenanceWindow object but got %T", obj)
	}
	clustermaintenancewindowlog.Info("validate delete", "name", window.Name)

	return nil, nil
}

func (r *ClusterMaintenanceWindowCustomWebhook) validateClusterMaintenanceWindow(ctx context.Context, window *ClusterMaintenanceWindow) error {
	if window.Spec.Timezone != nil {
		if err := validateTimeZone(pointer.String(window.Spec.Timezone)); err != nil {
			return err
		}
	}
	if !window.Spec.IsDefault {
		return nil
	}

	if webhookClient == nil {
		return errors.New("webhook client is not set")
	}
	cmwList := &ClusterMaintenanceWindowList{}
	if err := webhookClient.List(ctx, cmwList, client.MatchingFields{
		DefaultClusterMaintenanceWindowKey: "true",
	}); err != nil {
		return err
	}

	if len(cmwList.Items) == 0 {
		return nil
	} else if len(cmwList.Items) == 1 && cmwList.Items[0].Name == window.Name { // to handle update operation of the Default ClusterMaintenanceWindow
		return nil
	} else {
		return fmt.Errorf("%q is already present as default ClusterMaintenanceWindow", cmwList.Items[0].Name)
	}
}
