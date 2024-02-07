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
	"io"
	"net"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"kubeops.dev/supervisor/pkg/controllers"
	"kubeops.dev/supervisor/pkg/server"

	"github.com/spf13/pflag"
	v "gomodules.xyz/x/version"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	reg "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/features"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	reg_util "kmodules.xyz/client-go/admissionregistration/v1"
	"kmodules.xyz/client-go/tools/clientcmd"
	"kmodules.xyz/webhook-runtime/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	defaultEtcdPathPrefix = "/registry/kubeops.dev"
)

type SupervisorOperatorOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions
	ExtraOptions       *ExtraOptions

	StdOut io.Writer
	StdErr io.Writer
}

func NewSupervisorOperatorOptions(out, errOut io.Writer) *SupervisorOperatorOptions {
	_ = feature.DefaultMutableFeatureGate.Set(fmt.Sprintf("%s=false", features.APIPriorityAndFairness))
	o := &SupervisorOperatorOptions{
		// TODO we will nil out the etcd storage options.  This requires a later level of k8s.io/apiserver
		RecommendedOptions: genericoptions.NewRecommendedOptions(
			defaultEtcdPathPrefix,
			server.Codecs.LegacyCodec(admissionv1beta1.SchemeGroupVersion),
		),
		ExtraOptions: NewExtraOptions(),
		StdOut:       out,
		StdErr:       errOut,
	}
	o.RecommendedOptions.Etcd = nil
	o.RecommendedOptions.Admission = nil

	return o
}

func (o SupervisorOperatorOptions) AddFlags(fs *pflag.FlagSet) {
	o.RecommendedOptions.AddFlags(fs)
	o.ExtraOptions.AddFlags(fs)
}

func (o SupervisorOperatorOptions) Validate(args []string) error {
	var errors []error
	errors = append(errors, o.RecommendedOptions.Validate()...)
	errors = append(errors, o.ExtraOptions.Validate()...)
	return utilerrors.NewAggregate(errors)
}

func (o *SupervisorOperatorOptions) Complete() error {
	return nil
}

func (o SupervisorOperatorOptions) Config() (*server.SupervisorOperatorConfig, error) {
	// TODO have a "real" external address
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewRecommendedConfig(server.Codecs)
	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}
	// Fixes https://github.com/Azure/AKS/issues/522
	clientcmd.Fix(serverConfig.ClientConfig)

	ignorePrefixes := []string{
		"/swaggerapi",

		"/apis/mutators.supervisor.appscode.com/v1alpha1",
		"/apis/mutators.supervisor.appscode.com/v1alpha1/recommendationwebhooks",

		"/apis/validators.supervisor.appscode.com/v1alpha1",
		"/apis/validators.supervisor.appscode.com/v1alpha1/clustermaintenancewindowwebhooks",
		"/apis/validators.supervisor.appscode.com/v1alpha1/maintenancewindowwebhooks",
		"/apis/validators.supervisor.appscode.com/v1alpha1/recommendationwebhooks",
	}
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(
		api.GetOpenAPIDefinitions,
		openapi.NewDefinitionNamer(server.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "supervisor"
	serverConfig.OpenAPIConfig.Info.Version = v.Version.Version
	serverConfig.OpenAPIConfig.IgnorePrefixes = ignorePrefixes

	serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(
		api.GetOpenAPIDefinitions,
		openapi.NewDefinitionNamer(server.Scheme))
	serverConfig.OpenAPIV3Config.Info.Title = "supervisor"
	serverConfig.OpenAPIV3Config.Info.Version = v.Version.Version
	serverConfig.OpenAPIV3Config.IgnorePrefixes = ignorePrefixes

	extraConfig := controllers.NewConfig(serverConfig.ClientConfig)
	if err := o.ExtraOptions.ApplyTo(extraConfig); err != nil {
		return nil, err
	}

	// serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(
	//	api.GetOpenAPIDefinitions,
	//	openapi.NewDefinitionNamer(server.Scheme))
	// serverConfig.OpenAPIConfig.Info.Title = "Supervisor"
	// serverConfig.OpenAPIConfig.Info.Version = "v0.0.1"

	cfg := &server.SupervisorOperatorConfig{
		GenericConfig: serverConfig,
		ExtraConfig:   extraConfig,
	}
	return cfg, nil
}

func (o SupervisorOperatorOptions) Run(ctx context.Context) error {
	cfg, err := o.Config()
	if err != nil {
		return err
	}

	s, err := cfg.Complete().New()
	if err != nil {
		return err
	}

	{
		err = rest.LoadTLSFiles(cfg.ExtraConfig.ClientConfig)
		if err != nil {
			return err
		}

		apiGroups := []string{
			api.GroupVersion.Group,
		}
		kc := kubernetes.NewForConfigOrDie(cfg.ExtraConfig.ClientConfig)
		for _, g := range apiGroups {
			if wc, err := kc.AdmissionregistrationV1().
				MutatingWebhookConfigurations().
				Get(context.TODO(), string(builder.MutatorGroupPrefix)+g, metav1.GetOptions{}); client.IgnoreNotFound(err) != nil {
				return err
			} else if err == nil {
				_, _, err := reg_util.PatchMutatingWebhookConfiguration(
					context.TODO(),
					kc,
					wc,
					func(in *reg.MutatingWebhookConfiguration) *reg.MutatingWebhookConfiguration {
						for i := range in.Webhooks {
							in.Webhooks[i].ClientConfig.CABundle = cfg.ExtraConfig.ClientConfig.CAData
						}
						return in
					}, metav1.PatchOptions{})
				if client.IgnoreNotFound(err) != nil {
					return err
				}
			}

			if wc, err := kc.AdmissionregistrationV1().
				ValidatingWebhookConfigurations().
				Get(context.TODO(), string(builder.ValidatorGroupPrefix)+g, metav1.GetOptions{}); client.IgnoreNotFound(err) != nil {
				return err
			} else if err == nil {
				_, _, err := reg_util.PatchValidatingWebhookConfiguration(
					context.TODO(),
					kc,
					wc,
					func(in *reg.ValidatingWebhookConfiguration) *reg.ValidatingWebhookConfiguration {
						for i := range in.Webhooks {
							in.Webhooks[i].ClientConfig.CABundle = cfg.ExtraConfig.ClientConfig.CAData
						}
						return in
					}, metav1.PatchOptions{})
				if client.IgnoreNotFound(err) != nil {
					return err
				}
			}
		}

	}

	s.GenericAPIServer.AddPostStartHookOrDie("start-supervisor-informers", func(context genericapiserver.PostStartHookContext) error {
		cfg.GenericConfig.SharedInformerFactory.Start(context.StopCh)
		return nil
	})

	err = s.Manager.Add(manager.RunnableFunc(func(ctx context.Context) error {
		return s.GenericAPIServer.PrepareRun().Run(ctx.Done())
	}))
	if err != nil {
		return err
	}

	setupLog := log.Log.WithName("setup")
	setupLog.Info("starting manager")
	return s.Manager.Start(ctx)
}
