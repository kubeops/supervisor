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
	"fmt"
	"os"
	"strings"
	"sync"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"kubeops.dev/supervisor/pkg/controllers"
	supervisorcontrollers "kubeops.dev/supervisor/pkg/controllers/supervisor"

	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	cu "kmodules.xyz/client-go/client"
	hooks "kmodules.xyz/webhook-runtime/admission/v1"
	admissionreview "kmodules.xyz/webhook-runtime/registry/admissionreview/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme = runtime.NewScheme()
	// Codecs provides methods for retrieving codecs and serializers for specific
	// versions and content types.
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	utilruntime.Must(api.AddToScheme(Scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(admissionv1.AddToScheme(Scheme))
	utilruntime.Must(admissionv1beta1.AddToScheme(Scheme))

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

// SupervisorOperatorConfig defines the config for the apiserver
type SupervisorOperatorConfig struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   *controllers.Config
}

// SupervisorOperator contains state for a Kubernetes cluster master/api server.
type SupervisorOperator struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
	Manager          manager.Manager
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *controllers.Config
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *SupervisorOperatorConfig) Complete() CompletedConfig {
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		cfg.ExtraConfig,
	}

	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&c}
}

// New returns a new instance of SupervisorOperator from the given config.
func (c completedConfig) New() (*SupervisorOperator, error) {
	genericServer, err := c.GenericConfig.New("supervisor", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	// ctrl.SetLogger(...)
	log.SetLogger(klogr.New()) // nolint:staticcheck
	setupLog := log.Log.WithName("setup")

	cfg := c.ExtraConfig.ClientConfig

	crdClient, err := crd_cs.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "failed to create crd client")
		os.Exit(1)
	}
	err = controllers.EnsureCustomResourceDefinitions(crdClient)
	if err != nil {
		setupLog.Error(err, "failed to register crds")
		os.Exit(1)
	}

	mgr, err := manager.New(cfg, manager.Options{
		Scheme:                 Scheme,
		Metrics:                metricsserver.Options{BindAddress: ""},
		HealthProbeBindAddress: "0",
		LeaderElection:         false,
		LeaderElectionID:       "3a57c480.supervisor.appscode.com",
		Cache: cache.Options{
			SyncPeriod: &c.ExtraConfig.ResyncPeriod,
		},
		NewClient: cu.NewClient,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	api.SetupWebhookClient(mgr.GetClient())

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &api.MaintenanceWindow{}, api.DefaultMaintenanceWindowKey, func(rawObj client.Object) []string {
		app := rawObj.(*api.MaintenanceWindow)
		if v, ok := app.Annotations[api.DefaultMaintenanceWindowKey]; ok && v == "true" {
			return []string{"true"}
		}
		return nil
	}); err != nil {
		klog.Error(err, "unable to set up MaintenanceWindow Indexer", "field", api.DefaultMaintenanceWindowKey)
		os.Exit(1)
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &api.ClusterMaintenanceWindow{}, api.DefaultClusterMaintenanceWindowKey, func(rawObj client.Object) []string {
		app := rawObj.(*api.ClusterMaintenanceWindow)
		if v, ok := app.Annotations[api.DefaultClusterMaintenanceWindowKey]; ok && v == "true" {
			return []string{"true"}
		}
		return nil
	}); err != nil {
		klog.Error(err, "unable to set up ClusterMaintenanceWindow Indexer", "field", api.DefaultClusterMaintenanceWindowKey)
		os.Exit(1)
	}

	recommendationControllerOpts := controller.Options{
		MaxConcurrentReconciles: c.ExtraConfig.MaxConcurrentReconcile,
	}
	if err = (&supervisorcontrollers.RecommendationReconciler{
		Client:                 mgr.GetClient(),
		Scheme:                 mgr.GetScheme(),
		Mutex:                  &sync.Mutex{},
		RequeueAfterDuration:   c.ExtraConfig.RequeueAfterDuration,
		RetryAfterDuration:     c.ExtraConfig.RetryAfterDuration,
		BeforeDeadlineDuration: c.ExtraConfig.BeforeDeadlineDuration,
		Clock:                  api.GetClock(),
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
	//+kubebuilder:scaffold:builder

	s := &SupervisorOperator{
		GenericAPIServer: genericServer,
		Manager:          mgr,
	}

	for _, versionMap := range admissionHooksByGroupThenVersion(c.ExtraConfig.AdmissionHooks...) {
		// TODO we're going to need a later k8s.io/apiserver so that we can get discovery to list a different group version for
		// our endpoint which we'll use to back some custom storage which will consume the AdmissionReview type and give back the correct response
		apiGroupInfo := genericapiserver.APIGroupInfo{
			VersionedResourcesStorageMap: map[string]map[string]rest.Storage{},
			// TODO unhardcode this.  It was hardcoded before, but we need to re-evaluate
			OptionsExternalVersion: &schema.GroupVersion{Version: "v1"},
			Scheme:                 Scheme,
			ParameterCodec:         metav1.ParameterCodec,
			NegotiatedSerializer:   Codecs,
		}

		for _, admissionHooks := range versionMap {
			for i := range admissionHooks {
				admissionHook := admissionHooks[i]
				admissionResource, _ := admissionHook.Resource()
				admissionVersion := admissionResource.GroupVersion()

				// just overwrite the groupversion with a random one.  We don't really care or know.
				apiGroupInfo.PrioritizedVersions = appendUniqueGroupVersion(apiGroupInfo.PrioritizedVersions, admissionVersion)

				admissionReview := admissionreview.NewREST(admissionHook.Admit)
				v1alpha1storage, ok := apiGroupInfo.VersionedResourcesStorageMap[admissionVersion.Version]
				if !ok {
					v1alpha1storage = map[string]rest.Storage{}
				}
				v1alpha1storage[admissionResource.Resource] = admissionReview
				apiGroupInfo.VersionedResourcesStorageMap[admissionVersion.Version] = v1alpha1storage
			}
		}

		if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
			return nil, err
		}
	}

	for i := range c.ExtraConfig.AdmissionHooks {
		admissionHook := c.ExtraConfig.AdmissionHooks[i]
		postStartName := postStartHookName(admissionHook)
		if len(postStartName) == 0 {
			continue
		}
		s.GenericAPIServer.AddPostStartHookOrDie(postStartName,
			func(context genericapiserver.PostStartHookContext) error {
				return admissionHook.Initialize(c.ExtraConfig.ClientConfig, context.StopCh)
			},
		)
	}

	return s, nil
}

func appendUniqueGroupVersion(slice []schema.GroupVersion, elems ...schema.GroupVersion) []schema.GroupVersion {
	m := map[schema.GroupVersion]bool{}
	for _, gv := range slice {
		m[gv] = true
	}
	for _, e := range elems {
		m[e] = true
	}
	out := make([]schema.GroupVersion, 0, len(m))
	for gv := range m {
		out = append(out, gv)
	}
	return out
}

func postStartHookName(hook hooks.AdmissionHook) string {
	var ns []string
	gvr, _ := hook.Resource()
	ns = append(ns, fmt.Sprintf("admit-%s.%s.%s", gvr.Resource, gvr.Version, gvr.Group))
	if len(ns) == 0 {
		return ""
	}
	return strings.Join(append(ns, "init"), "-")
}

func admissionHooksByGroupThenVersion(admissionHooks ...hooks.AdmissionHook) map[string]map[string][]hooks.AdmissionHook {
	ret := map[string]map[string][]hooks.AdmissionHook{}
	for i := range admissionHooks {
		hook := admissionHooks[i]
		gvr, _ := hook.Resource()
		group, ok := ret[gvr.Group]
		if !ok {
			group = map[string][]hooks.AdmissionHook{}
			ret[gvr.Group] = group
		}
		group[gvr.Version] = append(group[gvr.Version], hook)
	}
	return ret
}
