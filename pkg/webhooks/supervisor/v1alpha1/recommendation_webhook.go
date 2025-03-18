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
	"reflect"

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

// SetupRecommendationWebhookWithManager registers the webhook for Recommendation in the manager.
func SetupRecommendationWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&api.Recommendation{}).
		WithValidator(&RecommendationCustomWebhook{mgr.GetClient()}).
		WithDefaulter(&RecommendationCustomWebhook{mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-supervisor-appscode-com-v1alpha1-recommendation,mutating=true,failurePolicy=fail,sideEffects=None,groups=supervisor.appscode.com,resources=recommendations,verbs=create;update,versions=v1alpha1,name=mrecommendation.kb.io,admissionReviewVersions={v1,v1beta1}

// +kubebuilder:object:generate=false
type RecommendationCustomWebhook struct {
	DefaultClient client.Client
}

// log is for logging in this package.
var (
	recommendationlog = logf.Log.WithName("recommendation-resource")
)

var _ webhook.CustomDefaulter = &RecommendationCustomWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *RecommendationCustomWebhook) Default(ctx context.Context, obj runtime.Object) error {
	rec, ok := obj.(*api.Recommendation)
	if !ok {
		return fmt.Errorf("expected an Recommendation object but got %T", obj)
	}

	recommendationlog.Info("default", "name", rec.Name)

	if rec.Spec.BackoffLimit == nil {
		rec.Spec.BackoffLimit = pointer.Int32P(api.DefaultBackoffLimit)
	}
	return nil
}

var _ webhook.CustomValidator = &RecommendationCustomWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *RecommendationCustomWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	rec, ok := obj.(*api.Recommendation)
	if !ok {
		return nil, fmt.Errorf("expected an Recommendation object but got %T", obj)
	}

	recommendationlog.Info("validate create", "name", rec.Name)

	return nil, r.validateRecommendation(rec)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *RecommendationCustomWebhook) ValidateUpdate(ctx context.Context, old, newObj runtime.Object) (admission.Warnings, error) {
	rec, ok := newObj.(*api.Recommendation)
	if !ok {
		return nil, fmt.Errorf("expected an Recommendation object but got %T", newObj)
	}

	recommendationlog.Info("validate update", "name", rec.Name)

	obj := old.(*api.Recommendation)

	if !reflect.DeepEqual(obj.Spec.Operation, rec.Spec.Operation) || !reflect.DeepEqual(obj.Spec.Target, rec.Spec.Target) {
		return nil, errors.New("can't update operation or target field. fields are immutable")
	}
	return nil, r.validateRecommendation(rec)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *RecommendationCustomWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	rec, ok := obj.(*api.Recommendation)
	if !ok {
		return nil, fmt.Errorf("expected an Recommendation object but got %T", obj)
	}

	recommendationlog.Info("validate delete", "name", rec.Name)

	return nil, nil
}

func (r *RecommendationCustomWebhook) validateRecommendation(rec *api.Recommendation) error {
	klog.Info("Validating Recommendation webhook")
	if rec.Spec.BackoffLimit == nil {
		return errors.New("backoffLimit field .spec.backoffLimit must not be nil")
	}
	if len(rec.Spec.Rules.Success) == 0 || len(rec.Spec.Rules.InProgress) == 0 || len(rec.Spec.Rules.Failed) == 0 {
		return errors.New("success/inProgress/failed rules can't be empty")
	}

	return nil
}
