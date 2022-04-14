/*
Copyright AppsCode Inc. and Contributors.

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

package controllers

import (
	"time"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"

	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/apiextensions"
	hooks "kmodules.xyz/webhook-runtime/admission/v1"
)

type Config struct {
	ClientConfig *rest.Config

	ResyncPeriod           time.Duration
	MaxConcurrentReconcile int // NumThreads
	RequeueAfterDuration   time.Duration
	MaxRetryOnFailure      int // MaxNumRequeues
	RetryAfterDuration     time.Duration
	BeforeDeadlineDuration time.Duration

	EnableValidatingWebhook bool
	EnableMutatingWebhook   bool

	AdmissionHooks []hooks.AdmissionHook
}

func NewConfig(cfg *rest.Config) *Config {
	return &Config{
		ClientConfig: cfg,
	}
}

func EnsureCustomResourceDefinitions(client crd_cs.Interface) error {
	klog.Infoln("Ensuring CustomResourceDefinition...")
	crds := []*apiextensions.CustomResourceDefinition{
		api.ApprovalPolicy{}.CustomResourceDefinition(),
		api.ClusterMaintenanceWindow{}.CustomResourceDefinition(),
		api.MaintenanceWindow{}.CustomResourceDefinition(),
		api.Recommendation{}.CustomResourceDefinition(),
	}
	return apiextensions.RegisterCRDs(client, crds)
}
