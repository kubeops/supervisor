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

package cmds

import (
	"context"
	"crypto/tls"
	"os"
	"path/filepath"
	"sync"
	"time"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	supervisorcontrollers "kubeops.dev/supervisor/pkg/controllers/supervisor"
	webhooks "kubeops.dev/supervisor/pkg/webhooks/supervisor/v1alpha1"

	"github.com/spf13/cobra"
	v "gomodules.xyz/x/version"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog/v2"
	cu "kmodules.xyz/client-go/client"
	clustermeta "kmodules.xyz/client-go/cluster"
	"kmodules.xyz/client-go/meta"
	_ "kmodules.xyz/client-go/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(api.AddToScheme(scheme))
}

func NewCmdRun() *cobra.Command {
	var (
		QPS   float32 = 1e6
		Burst int     = 1e6

		metricsAddr          string = "0"
		certDir              string
		enableLeaderElection bool   = false
		probeAddr            string = ":8081"
		secureMetrics        bool   = true
		enableHTTP2          bool   = false

		ResyncPeriod           time.Duration = 10 * time.Minute
		MaxConcurrentReconcile int           // NumThreads = 2
		RequeueAfterDuration   time.Duration
		MaxRetryOnFailure      int // MaxNumRequeues = 5
		RetryAfterDuration     time.Duration
		BeforeDeadlineDuration time.Duration
	)
	cmd := &cobra.Command{
		Use:               "run",
		Short:             "Launch Sidekick Operator",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			klog.Infof("Starting binary version %s+%s ...", v.Version.Version, v.Version.CommitHash)

			var tlsOpts []func(*tls.Config)
			ctrl.SetLogger(klog.NewKlogr())

			cfg := ctrl.GetConfigOrDie()
			cfg.QPS = QPS
			cfg.Burst = Burst

			// if the enable-http2 flag is false (the default), http/2 should be disabled
			// due to its vulnerabilities. More specifically, disabling http/2 will
			// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
			// Rapid Reset CVEs. For more information see:
			// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
			// - https://github.com/advisories/GHSA-4374-p667-p6c8
			disableHTTP2 := func(c *tls.Config) {
				setupLog.Info("disabling http/2")
				c.NextProtos = []string{"http/1.1"}
			}

			if !enableHTTP2 {
				tlsOpts = append(tlsOpts, disableHTTP2)
			}

			// Create watchers for metrics and webhooks certificates
			var certWatcher *certwatcher.CertWatcher

			// Initial TLS options
			webhookTLSOpts := tlsOpts
			metricsTLSOpts := tlsOpts

			if len(certDir) > 0 {
				setupLog.Info("Initializing webhook certificate watcher using provided certificates",
					"cert-dir", certDir, "cert-name", core.TLSCertKey, "webhook-cert-key", core.TLSPrivateKeyKey)

				var err error
				certWatcher, err = certwatcher.New(
					filepath.Join(certDir, core.TLSCertKey),
					filepath.Join(certDir, core.TLSPrivateKeyKey),
				)
				if err != nil {
					setupLog.Error(err, "Failed to initialize webhook certificate watcher")
					os.Exit(1)
				}

				webhookTLSOpts = append(webhookTLSOpts, func(config *tls.Config) {
					config.GetCertificate = certWatcher.GetCertificate
				})

				metricsTLSOpts = append(metricsTLSOpts, func(config *tls.Config) {
					config.GetCertificate = certWatcher.GetCertificate
				})
			}

			webhookServer := webhook.NewServer(webhook.Options{
				TLSOpts: webhookTLSOpts,
			})

			// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
			// More info:
			// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/metrics/server
			// - https://book.kubebuilder.io/reference/metrics.html
			metricsServerOptions := metricsserver.Options{
				BindAddress:   metricsAddr,
				SecureServing: secureMetrics,
				TLSOpts:       metricsTLSOpts,
			}

			if secureMetrics {
				// FilterProvider is used to protect the metrics endpoint with authn/authz.
				// These configurations ensure that only authorized users and service accounts
				// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
				// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/metrics/filters#WithAuthenticationAndAuthorization
				metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
			}

			mgr, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme:                 scheme,
				Metrics:                metricsServerOptions,
				WebhookServer:          webhookServer,
				HealthProbeBindAddress: probeAddr,
				LeaderElection:         enableLeaderElection,
				LeaderElectionID:       "3a57c480.supervisor.appscode.com",
				// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
				// when the Manager ends. This requires the binary to immediately end when the
				// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
				// speeds up voluntary leader transitions as the new leader don't have to wait
				// LeaseDuration time first.
				//
				// In the default scaffold provided, the program ends immediately after
				// the manager stops, so would be fine to enable this option. However,
				// if you are doing or is intended to do any operation such as perform cleanups
				// after the manager stops then its usage might be unsafe.
				// LeaderElectionReleaseOnCancel: true,
				Cache: cache.Options{
					SyncPeriod: &ResyncPeriod,
				},
				NewClient: cu.NewClient,
			})
			if err != nil {
				setupLog.Error(err, "unable to start manager")
				os.Exit(1)
			}

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
				MaxConcurrentReconciles: MaxConcurrentReconcile,
			}
			if err = (&supervisorcontrollers.RecommendationReconciler{
				Client:                 mgr.GetClient(),
				Scheme:                 mgr.GetScheme(),
				Mutex:                  &sync.Mutex{},
				RequeueAfterDuration:   RequeueAfterDuration,
				RetryAfterDuration:     RetryAfterDuration,
				BeforeDeadlineDuration: BeforeDeadlineDuration,
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

			if err = webhooks.SetupRecommendationWebhookWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to setup Recommendation webhook")
				os.Exit(1)
			}
			if err = webhooks.SetupMaintenanceWindowWebhookWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to setup MaintenanceWindow webhook")
				os.Exit(1)
			}
			if err = webhooks.SetupClusterMaintenanceWindowWebhookWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to setup ClusterMaintenanceWindow webhook")
				os.Exit(1)
			}

			if certWatcher != nil {
				setupLog.Info("Adding certificate watcher to manager")
				if err := mgr.Add(certWatcher); err != nil {
					setupLog.Error(err, "unable to add certificate watcher to manager")
					os.Exit(1)
				}
			}

			if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
				setupLog.Error(err, "unable to set up health check")
				os.Exit(1)
			}
			if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
				setupLog.Error(err, "unable to set up ready check")
				os.Exit(1)
			}

			setupLog.Info("starting manager")
			if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
				setupLog.Error(err, "problem running manager")
				os.Exit(1)
			}
		},
	}

	meta.AddLabelBlacklistFlag(cmd.Flags())
	clustermeta.AddFlags(cmd.Flags())
	cmd.Flags().Float32Var(&QPS, "qps", QPS, "The maximum QPS to the master from this client")
	cmd.Flags().IntVar(&Burst, "burst", Burst, "The maximum burst for throttle")
	cmd.Flags().StringVar(&metricsAddr, "metrics-bind-address", metricsAddr, "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	cmd.Flags().StringVar(&probeAddr, "health-probe-bind-address", probeAddr, "The address the probe endpoint binds to.")
	cmd.Flags().BoolVar(&enableLeaderElection, "leader-elect", enableLeaderElection,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	cmd.Flags().BoolVar(&secureMetrics, "metrics-secure", secureMetrics,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	cmd.Flags().StringVar(&certDir, "cert-dir", certDir,
		"The directory that contains the metrics and webhook server certificate.")
	cmd.Flags().BoolVar(&enableHTTP2, "enable-http2", enableHTTP2,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")

	cmd.Flags().DurationVar(&ResyncPeriod, "resync-period", ResyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")
	cmd.Flags().IntVar(&MaxConcurrentReconcile, "max-concurrent-reconcile", MaxConcurrentReconcile, "Maximum number of Recommendation object that will be reconciled concurrently")
	cmd.Flags().DurationVar(&RequeueAfterDuration, "requeue-after-duration", RequeueAfterDuration, "Duration after the Recommendation object will be requeue when it is waiting for MaintenanceWindow. The flag accepts a value acceptable to time.ParseDuration. Ref: https://pkg.go.dev/time#ParseDuration")
	cmd.Flags().IntVar(&MaxRetryOnFailure, "max-retry-on-failure", MaxRetryOnFailure, "Maximum number of retry on any kind of failure in Recommendation execution")
	cmd.Flags().DurationVar(&RetryAfterDuration, "retry-after-duration", RetryAfterDuration, "Duration after the failure events will be requeue again. The flag accepts a value acceptable to time.ParseDuration. Ref: https://pkg.go.dev/time#ParseDuration")
	cmd.Flags().DurationVar(&BeforeDeadlineDuration, "before-deadline-duration", BeforeDeadlineDuration, "When there is less time than `BeforeDeadlineDuration` before deadline, Recommendations are free to execute regardless of Parallelism")

	return cmd
}
