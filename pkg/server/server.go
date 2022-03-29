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

package server

import (
	"context"
	"os"
	"sync"

	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	supervisorcontrollers "kubeops.dev/supervisor/pkg/controllers/supervisor"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/version"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	kubedbapi "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme = runtime.NewScheme()
	// Codecs provides methods for retrieving codecs and serializers for specific
	// versions and content types.
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(supervisorv1alpha1.AddToScheme(Scheme))
	utilruntime.Must(opsapi.AddToScheme(Scheme))
	utilruntime.Must(kubedbapi.AddToScheme(Scheme))

	// we need to add the options to empty v1
	// TODO: fix the server code to avoid this
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

// ExtraConfig holds custom apiserver config
type ExtraConfig struct {
	ClientConfig    *restclient.Config
	ReconcileConfig supervisorcontrollers.RecommendationReconcileConfig
}

// SupervisorOperatorConfig defines the config for the apiserver
type SupervisorOperatorConfig struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

// SupervisorOperator contains state for a Kubernetes cluster master/api server.
type SupervisorOperator struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
	Manager          manager.Manager
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *SupervisorOperatorConfig) Complete() CompletedConfig {
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}

	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&c}
}

// New returns a new instance of SupervisorOperator from the given config.
func (c completedConfig) New(ctx context.Context) (*SupervisorOperator, error) {
	genericServer, err := c.GenericConfig.New("ui-server", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	// ctrl.SetLogger(...)
	log.SetLogger(klogr.New())

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 Scheme,
		MetricsBindAddress:     "",
		Port:                   9443,
		HealthProbeBindAddress: "",
		LeaderElection:         false,
		LeaderElectionID:       "3a57c480.supervisor.appscode.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &supervisorv1alpha1.MaintenanceWindow{}, supervisorv1alpha1.DefaultMaintenanceWindowKey, func(rawObj client.Object) []string {
		app := rawObj.(*supervisorv1alpha1.MaintenanceWindow)
		if v, ok := app.Annotations[supervisorv1alpha1.DefaultMaintenanceWindowKey]; ok && v == "true" {
			return []string{"true"}
		}
		return nil
	}); err != nil {
		klog.Error(err, "unable to set up MaintenanceWindow Indexer", "field", supervisorv1alpha1.DefaultMaintenanceWindowKey)
		os.Exit(1)
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &supervisorv1alpha1.ClusterMaintenanceWindow{}, supervisorv1alpha1.DefaultClusterMaintenanceWindowKey, func(rawObj client.Object) []string {
		app := rawObj.(*supervisorv1alpha1.ClusterMaintenanceWindow)
		if v, ok := app.Annotations[supervisorv1alpha1.DefaultClusterMaintenanceWindowKey]; ok && v == "true" {
			return []string{"true"}
		}
		return nil
	}); err != nil {
		klog.Error(err, "unable to set up ClusterMaintenanceWindow Indexer", "field", supervisorv1alpha1.DefaultClusterMaintenanceWindowKey)
		os.Exit(1)
	}

	recommendationControllerOpts := controller.Options{
		MaxConcurrentReconciles: c.ExtraConfig.ReconcileConfig.MaxConcurrentReconcile,
	}
	if err = (&supervisorcontrollers.RecommendationReconciler{
		Client:                 mgr.GetClient(),
		Scheme:                 mgr.GetScheme(),
		Mutex:                  &sync.Mutex{},
		RequeueAfterDuration:   c.ExtraConfig.ReconcileConfig.RequeueAfterDuration,
		RetryAfterDuration:     c.ExtraConfig.ReconcileConfig.RetryAfterDuration,
		BeforeDeadlineDuration: c.ExtraConfig.ReconcileConfig.BeforeDeadlineDuration,
		Clock:                  supervisorv1alpha1.GetClock(),
	}).SetupWithManager(mgr, recommendationControllerOpts); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Recommendation")
		os.Exit(1)
	}
	if err = (&supervisorcontrollers.MaintenanceWindowReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MaintenanceWindow")
		os.Exit(1)
	}
	if err = (&supervisorcontrollers.ClusterMaintenanceWindowReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterMaintenanceWindow")
		os.Exit(1)
	}
	if err = (&supervisorcontrollers.ApprovalPolicyReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApprovalPolicy")
		os.Exit(1)
	}
	mgr.GetWebhookServer().CertDir = c.ExtraConfig.ReconcileConfig.WebhookCertDir
	if err = (&supervisorv1alpha1.MaintenanceWindow{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "MaintenanceWindow")
		os.Exit(1)
	}
	if err = (&supervisorv1alpha1.ClusterMaintenanceWindow{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ClusterMaintenanceWindow")
		os.Exit(1)
	}
	if err = (&supervisorv1alpha1.Recommendation{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Recommendation")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	s := &SupervisorOperator{
		GenericAPIServer: genericServer,
		Manager:          mgr,
	}

	return s, nil
}
