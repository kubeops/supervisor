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
	"time"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"

	"gomodules.xyz/pointer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupMaintenanceWindowWebhookWithManager registers the webhook for MaintenanceWindow in the manager.
func SetupMaintenanceWindowWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&api.MaintenanceWindow{}).
		WithValidator(&MaintenanceWindowCustomWebhook{mgr.GetClient()}).
		WithDefaulter(&MaintenanceWindowCustomWebhook{mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-supervisor-appscode-com-v1alpha1-maintenancewindow,mutating=true,failurePolicy=fail,sideEffects=None,groups=supervisor.appscode.com,resources=maintenancewindows,verbs=create;update,versions=v1alpha1,name=mmaintenancewindow.kb.io,admissionReviewVersions={v1,v1beta1}

// +kubebuilder:object:generate=false
type MaintenanceWindowCustomWebhook struct {
	DefaultClient client.Client
}

var _ webhook.CustomDefaulter = &MaintenanceWindowCustomWebhook{}

// log is for logging in this package.
var (
	maintenancewindowlog = logf.Log.WithName("maintenancewindow-resource")
)

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *MaintenanceWindowCustomWebhook) Default(ctx context.Context, obj runtime.Object) error {
	window, ok := obj.(*api.MaintenanceWindow)
	if !ok {
		return fmt.Errorf("expected an MaintenanceWindow object but got %T", obj)
	}
	maintenancewindowlog.Info("default", "name", window.Name)
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-supervisor-appscode-com-v1alpha1-maintenancewindow,mutating=false,failurePolicy=fail,sideEffects=None,groups=supervisor.appscode.com,resources=maintenancewindows,verbs=create;update,versions=v1alpha1,name=vmaintenancewindow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.CustomValidator = &MaintenanceWindowCustomWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MaintenanceWindowCustomWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	window, ok := obj.(*api.MaintenanceWindow)
	if !ok {
		return nil, fmt.Errorf("expected an MaintenanceWindow object but got %T", obj)
	}

	maintenancewindowlog.Info("validate create", "name", window.Name)

	return nil, r.validateMaintenanceWindow(context.TODO(), window)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MaintenanceWindowCustomWebhook) ValidateUpdate(ctx context.Context, old, newObj runtime.Object) (admission.Warnings, error) {
	window, ok := newObj.(*api.MaintenanceWindow)
	if !ok {
		return nil, fmt.Errorf("expected an MaintenanceWindow object but got %T", newObj)
	}

	maintenancewindowlog.Info("validate update", "name", window.Name)

	return nil, r.validateMaintenanceWindow(context.TODO(), window)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MaintenanceWindowCustomWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	window, ok := obj.(*api.MaintenanceWindow)
	if !ok {
		return nil, fmt.Errorf("expected an MaintenanceWindow object but got %T", obj)
	}

	maintenancewindowlog.Info("validate delete", "name", window.Name)

	return nil, nil
}

func (r *MaintenanceWindowCustomWebhook) validateMaintenanceWindow(ctx context.Context, window *api.MaintenanceWindow) error {
	klog.Info("Validating MaintenanceWindow webhook")
	if window.Spec.Timezone != nil {
		if err := validateTimeZone(pointer.String(window.Spec.Timezone)); err != nil {
			return err
		}
	}
	if !window.Spec.IsDefault {
		return nil
	}
	var list api.MaintenanceWindowList
	if err := r.DefaultClient.List(ctx, &list, client.InNamespace(window.Namespace), client.MatchingFields{
		api.DefaultMaintenanceWindowKey: "true",
	}); err != nil {
		return err
	}

	if len(list.Items) == 0 {
		return nil
	} else if len(list.Items) == 1 && list.Items[0].Name == window.Name { // to handle update operation of the Default MaintenanceWindow
		return nil
	} else {
		return fmt.Errorf("%q is already present as default MaintenanceWindow in namespace %q", list.Items[0].Name, list.Items[0].Namespace)
	}
}

func validateTimeZone(location string) error {
	_, err := time.LoadLocation(location)
	return err
}
