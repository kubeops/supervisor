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

package e2e_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	kubedbapi "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"kubeops.dev/supervisor/test/e2e/framework"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
	root   *framework.Framework
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(supervisorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(opsapi.AddToScheme(scheme))
	utilruntime.Must(kubedbapi.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	cfg := getKubeConfig()

	By("Using kubeconfig from " + cfg)
	clientConfig, err := clientcmd.BuildConfigFromFlags("", cfg)
	Expect(err).NotTo(HaveOccurred())
	// raise throttling time. ref: https://github.com/appscode/voyager/issues/640
	clientConfig.Burst = 100
	clientConfig.QPS = 100

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     "0",
		Port:                   0,
		HealthProbeBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	go func() {
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			klog.Errorf("error from manager: %v", err)
		}
	}()

	root = framework.New(context.TODO(), clientConfig, mgr.GetClient())

	By("Creating namespace: " + root.Namespace())
	Expect(root.CreateNamespace()).Should(Succeed())

	By("Setting test env val")
	Expect(root.SetTestEnv()).Should(Succeed())

	By("Ensuring CRDs")
	root.EnsureCRD().Should(Succeed())

}, 60)

var _ = AfterSuite(func() {
	By("Deleting namespace: " + root.Namespace())
	Expect(root.DeleteNamespace()).Should(Succeed())
})

func getKubeConfig() string {
	cfg := os.Getenv("KUBECONFIG")
	if cfg != "" {
		return cfg
	}
	return filepath.Join(homedir.HomeDir(), ".kube", "config")
}
