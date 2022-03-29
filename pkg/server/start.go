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
	"io/ioutil"
	"net"
	"path"

	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"kubeops.dev/supervisor/pkg/controllers/supervisor"

	"github.com/spf13/pflag"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/features"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kutil "kmodules.xyz/client-go"
	reg_util "kmodules.xyz/client-go/admissionregistration/v1"
	"kmodules.xyz/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	defaultEtcdPathPrefix = "/registry/kubeops.dev"
	validatingWebhook     = "validators.supervisor.appscode.com"
	mutatingWebhook       = "mutators.supervisor.appscode.com"
)

type SupervisorOperatorOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions
	ExtraOptions       *ExtraOptions
	ReconcileOptions   *supervisor.RecommendationReconcileConfig

	StdOut io.Writer
	StdErr io.Writer
}

func NewSupervisorOperatorOptions(out, errOut io.Writer) *SupervisorOperatorOptions {
	_ = feature.DefaultMutableFeatureGate.Set(fmt.Sprintf("%s=false", features.APIPriorityAndFairness))
	o := &SupervisorOperatorOptions{
		// TODO we will nil out the etcd storage options.  This requires a later level of k8s.io/apiserver
		RecommendedOptions: genericoptions.NewRecommendedOptions(
			defaultEtcdPathPrefix,
			Codecs.LegacyCodec(admissionv1beta1.SchemeGroupVersion),
		),
		ExtraOptions:     NewExtraOptions(),
		ReconcileOptions: supervisor.NewRecommendationReconcileConfig(),
		StdOut:           out,
		StdErr:           errOut,
	}
	o.RecommendedOptions.Etcd = nil
	o.RecommendedOptions.Admission = nil

	return o
}

func (o SupervisorOperatorOptions) AddFlags(fs *pflag.FlagSet) {
	o.RecommendedOptions.AddFlags(fs)
	o.ExtraOptions.AddFlags(fs)
	o.ReconcileOptions.AddFlags(fs)
}

func (o SupervisorOperatorOptions) Validate(args []string) error {
	var errors []error
	errors = append(errors, o.RecommendedOptions.Validate()...)
	errors = append(errors, o.ReconcileOptions.Validate()...)
	return utilerrors.NewAggregate(errors)
}

func (o *SupervisorOperatorOptions) Complete() error {
	return nil
}

func (o SupervisorOperatorOptions) Config() (*SupervisorOperatorConfig, error) {
	// TODO have a "real" external address
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewRecommendedConfig(Codecs)
	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}
	// Fixes https://github.com/Azure/AKS/issues/522
	clientcmd.Fix(serverConfig.ClientConfig)

	extraConfig := NewConfig(serverConfig.ClientConfig)
	if err := o.ExtraOptions.ApplyTo(extraConfig); err != nil {
		return nil, err
	}

	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(
		supervisorv1alpha1.GetOpenAPIDefinitions,
		openapi.NewDefinitionNamer(Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "Supervisor-operator"
	serverConfig.OpenAPIConfig.Info.Version = "v0.0.1"

	if o.ExtraOptions.EnableValidatingWebhook {
		if err := updateValidatingWebhookCABundle(serverConfig.ClientConfig, validatingWebhook, o.ReconcileOptions.WebhookCertDir); err != nil {
			return nil, err
		}
	}

	if o.ExtraOptions.EnableMutatingWebhook {
		if err := updateMutatingWebhookCABundle(serverConfig.ClientConfig, mutatingWebhook, o.ReconcileOptions.WebhookCertDir); err != nil {
			return nil, err
		}
	}

	cfg := &SupervisorOperatorConfig{
		GenericConfig: serverConfig,
		ExtraConfig: ExtraConfig{
			ClientConfig:    serverConfig.ClientConfig,
			ReconcileConfig: *o.ReconcileOptions,
		},
	}
	return cfg, nil
}

var setupLog = ctrl.Log.WithName("setup")

func (o SupervisorOperatorOptions) Run(ctx context.Context) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	s, err := config.Complete().New(ctx)
	if err != nil {
		return err
	}

	setupLog.Info("starting manager")
	return s.Manager.Start(ctx)
}

func updateValidatingWebhookCABundle(config *rest.Config, webhookConfigName string, certDir string) error {
	crtByte, err := ioutil.ReadFile(path.Join(certDir, "tls.crt"))
	if err != nil {
		return err
	}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, kutil.ReadinessTimeout)
	defer cancel()

	kc, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	vhook, err := kc.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), webhookConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, _, err = reg_util.PatchValidatingWebhookConfiguration(ctx, kc, vhook.DeepCopy(), func(in *v1.ValidatingWebhookConfiguration) *v1.ValidatingWebhookConfiguration {
		for i := range in.Webhooks {
			in.Webhooks[i].ClientConfig.CABundle = crtByte
		}
		return in
	}, metav1.PatchOptions{})

	return err
}

func updateMutatingWebhookCABundle(config *rest.Config, webhookConfigName string, certDir string) error {
	crtByte, err := ioutil.ReadFile(path.Join(certDir, "tls.crt"))
	if err != nil {
		return err
	}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, kutil.ReadinessTimeout)
	defer cancel()

	kc, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	vhook, err := kc.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), webhookConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, _, err = reg_util.PatchMutatingWebhookConfiguration(ctx, kc, vhook.DeepCopy(), func(in *v1.MutatingWebhookConfiguration) *v1.MutatingWebhookConfiguration {
		for i := range in.Webhooks {
			in.Webhooks[i].ClientConfig.CABundle = crtByte
		}
		return in
	}, metav1.PatchOptions{})

	return err
}
