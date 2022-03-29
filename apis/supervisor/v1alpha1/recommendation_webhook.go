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

	"gomodules.xyz/pointer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var (
	recommendationlog = logf.Log.WithName("recommendation-resource")
	kc                client.Client
)

func (r *Recommendation) SetupWebhookWithManager(mgr ctrl.Manager) error {
	kc = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

var _ webhook.Defaulter = &Recommendation{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Recommendation) Default() {
	recommendationlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
	if r.Spec.MaxRetry == nil {
		r.Spec.MaxRetry = pointer.Int32P(0)
	}
}

var _ webhook.Validator = &Recommendation{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Recommendation) ValidateCreate() error {
	recommendationlog.Info("validate create", "name", r.Name)

	return r.validateRecommendation(context.TODO())
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Recommendation) ValidateUpdate(old runtime.Object) error {
	recommendationlog.Info("validate update", "name", r.Name)

	return r.validateRecommendation(context.TODO())
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Recommendation) ValidateDelete() error {
	recommendationlog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *Recommendation) validateRecommendation(ctx context.Context) error {
	klog.Info("Validating Recommendation webhook")
	if r.Spec.MaxRetry == nil {
		return errors.New(".spec.maxRetry must not be nil")
	}
	obj := &Recommendation{}
	err := kc.Get(ctx, client.ObjectKey{Name: r.Name, Namespace: r.Namespace}, obj)
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	if client.IgnoreNotFound(err) == nil && obj.Spec.MaxRetry != nil && obj.Spec.MaxRetry != r.Spec.MaxRetry {
		return errors.New("failed to update. reason: .spec.maxRetry field is immutable")
	}
	return nil
}
