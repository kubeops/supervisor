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
	"time"

	"gomodules.xyz/pointer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var (
	maintenancewindowlog = logf.Log.WithName("maintenancewindow-resource")
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-supervisor-appscode-com-v1alpha1-maintenancewindow,mutating=true,failurePolicy=fail,sideEffects=None,groups=supervisor.appscode.com,resources=maintenancewindows,verbs=create;update,versions=v1alpha1,name=mmaintenancewindow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &MaintenanceWindow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *MaintenanceWindow) Default() {
	maintenancewindowlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-supervisor-appscode-com-v1alpha1-maintenancewindow,mutating=false,failurePolicy=fail,sideEffects=None,groups=supervisor.appscode.com,resources=maintenancewindows,verbs=create;update,versions=v1alpha1,name=vmaintenancewindow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &MaintenanceWindow{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MaintenanceWindow) ValidateCreate() error {
	maintenancewindowlog.Info("validate create", "name", r.Name)

	return r.validateMaintenanceWindow(context.TODO())
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MaintenanceWindow) ValidateUpdate(old runtime.Object) error {
	maintenancewindowlog.Info("validate update", "name", r.Name)

	return r.validateMaintenanceWindow(context.TODO())
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MaintenanceWindow) ValidateDelete() error {
	maintenancewindowlog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *MaintenanceWindow) validateMaintenanceWindow(ctx context.Context) error {
	klog.Info("Validating MaintenanceWindow webhook")
	if r.Spec.Timezone != nil {
		if err := validateTimeZone(pointer.String(r.Spec.Timezone)); err != nil {
			return err
		}
	}
	if !r.Spec.IsDefault {
		return nil
	}
	if webhookClient == nil {
		return errors.New("webhook client is not set")
	}
	mwList := &MaintenanceWindowList{}
	if err := webhookClient.List(ctx, mwList, client.InNamespace(r.Namespace), client.MatchingFields{
		DefaultMaintenanceWindowKey: "true",
	}); err != nil {
		return err
	}

	if len(mwList.Items) == 0 {
		return nil
	} else if len(mwList.Items) == 1 && mwList.Items[0].Name == r.Name { // to handle update operation of the Default MaintenanceWindow
		return nil
	} else {
		return fmt.Errorf("%q is already present as default MaintenanceWindow in namespace %q", mwList.Items[0].Name, mwList.Items[0].Namespace)
	}
}

func validateTimeZone(location string) error {
	_, err := time.LoadLocation(location)
	return err
}
