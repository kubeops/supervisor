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
	"errors"
	"reflect"

	"gomodules.xyz/pointer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var (
	recommendationlog = logf.Log.WithName("recommendation-resource")
)

func (r *Recommendation) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

var _ webhook.Defaulter = &Recommendation{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Recommendation) Default() {
	recommendationlog.Info("default", "name", r.Name)

	if r.Spec.BackoffLimit == nil {
		r.Spec.BackoffLimit = pointer.Int32P(DefaultBackoffLimit)
	}
}

var _ webhook.Validator = &Recommendation{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Recommendation) ValidateCreate() error {
	recommendationlog.Info("validate create", "name", r.Name)

	return r.validateRecommendation()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Recommendation) ValidateUpdate(old runtime.Object) error {
	recommendationlog.Info("validate update", "name", r.Name)

	obj := old.(*Recommendation)

	if !reflect.DeepEqual(obj.Spec.Operation, r.Spec.Operation) || !reflect.DeepEqual(obj.Spec.Target, r.Spec.Target) {
		return errors.New("can't update operation or target field. fields are immutable")
	}
	return r.validateRecommendation()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Recommendation) ValidateDelete() error {
	recommendationlog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *Recommendation) validateRecommendation() error {
	klog.Info("Validating Recommendation webhook")
	if r.Spec.BackoffLimit == nil {
		return errors.New("backoffLimit field .spec.backoffLimit must not be nil")
	}
	if len(r.Spec.Rules.Success) == 0 || len(r.Spec.Rules.InProgress) == 0 || len(r.Spec.Rules.Failed) == 0 {
		return errors.New("success/inProgress/failed rules can't be empty")
	}

	return nil
}
