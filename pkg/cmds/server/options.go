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
	"errors"
	"flag"
	"time"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"kubeops.dev/supervisor/pkg/controllers"
	"kubeops.dev/supervisor/pkg/server"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"kmodules.xyz/client-go/tools/clusterid"
	"kmodules.xyz/webhook-runtime/builder"
)

type ExtraOptions struct {
	QPS   float64
	Burst int

	ResyncPeriod           time.Duration
	MaxConcurrentReconcile int // NumThreads
	RequeueAfterDuration   time.Duration
	MaxRetryOnFailure      int // MaxNumRequeues
	RetryAfterDuration     time.Duration
	BeforeDeadlineDuration time.Duration

	EnableValidatingWebhook bool
	EnableMutatingWebhook   bool
}

func NewExtraOptions() *ExtraOptions {
	return &ExtraOptions{
		MaxRetryOnFailure:      5,
		MaxConcurrentReconcile: 2,
		QPS:                    1e6,
		Burst:                  1e6,
		ResyncPeriod:           10 * time.Minute,
	}
}

func (s *ExtraOptions) AddGoFlags(fs *flag.FlagSet) {
	clusterid.AddGoFlags(fs)

	fs.Float64Var(&s.QPS, "qps", s.QPS, "The maximum QPS to the master from this client")
	fs.IntVar(&s.Burst, "burst", s.Burst, "The maximum burst for throttle")

	fs.DurationVar(&s.ResyncPeriod, "resync-period", s.ResyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")
	fs.IntVar(&s.MaxConcurrentReconcile, "max-concurrent-reconcile", s.MaxConcurrentReconcile, "Maximum number of Recommendation object that will be reconciled concurrently")
	fs.DurationVar(&s.RequeueAfterDuration, "requeue-after-duration", s.RequeueAfterDuration, "Duration after the Recommendation object will be requeue when it is waiting for MaintenanceWindow. The flag accepts a value acceptable to time.ParseDuration. Ref: https://pkg.go.dev/time#ParseDuration")
	fs.IntVar(&s.MaxRetryOnFailure, "max-retry-on-failure", s.MaxRetryOnFailure, "Maximum number of retry on any kind of failure in Recommendation execution")
	fs.DurationVar(&s.RetryAfterDuration, "retry-after-duration", s.RetryAfterDuration, "Duration after the failure events will be requeue again. The flag accepts a value acceptable to time.ParseDuration. Ref: https://pkg.go.dev/time#ParseDuration")
	fs.DurationVar(&s.BeforeDeadlineDuration, "before-deadline-duration", s.BeforeDeadlineDuration, "When there is less time than `BeforeDeadlineDuration` before deadline, Recommendations are free to execute regardless of Parallelism")

	fs.BoolVar(&s.EnableMutatingWebhook, "enable-mutating-webhook", s.EnableMutatingWebhook, "If true, enables mutating webhooks for Supervisor CRDs.")
	fs.BoolVar(&s.EnableValidatingWebhook, "enable-validating-webhook", s.EnableValidatingWebhook, "If true, enables validating webhooks for Supervisor CRDs.")
}

func (s *ExtraOptions) AddFlags(fs *pflag.FlagSet) {
	pfs := flag.NewFlagSet("grafana-tools", flag.ExitOnError)
	s.AddGoFlags(pfs)
	fs.AddGoFlagSet(pfs)
}

func (c *ExtraOptions) Validate() []error {
	errs := make([]error, 0)
	if c.MaxConcurrentReconcile <= 0 {
		errs = append(errs, errors.New("max-concurrent-reconcile must be greater than 0"))
	}
	if _, err := time.ParseDuration(c.RetryAfterDuration.String()); err != nil {
		errs = append(errs, err)
	}
	if _, err := time.ParseDuration(c.RequeueAfterDuration.String()); err != nil {
		errs = append(errs, err)
	}
	if _, err := time.ParseDuration(c.BeforeDeadlineDuration.String()); err != nil {
		errs = append(errs, err)
	}

	return errs
}

func (s *ExtraOptions) ApplyTo(cfg *controllers.Config) error {
	cfg.ClientConfig.QPS = float32(s.QPS)
	cfg.ClientConfig.Burst = s.Burst

	cfg.ResyncPeriod = s.ResyncPeriod
	cfg.MaxConcurrentReconcile = s.MaxConcurrentReconcile
	cfg.RequeueAfterDuration = s.RequeueAfterDuration
	cfg.MaxRetryOnFailure = s.MaxRetryOnFailure
	cfg.RetryAfterDuration = s.RetryAfterDuration
	cfg.BeforeDeadlineDuration = s.BeforeDeadlineDuration

	cfg.EnableMutatingWebhook = s.EnableMutatingWebhook
	cfg.EnableValidatingWebhook = s.EnableValidatingWebhook

	apiTypes := []runtime.Object{
		&api.ApprovalPolicy{},
		&api.ClusterMaintenanceWindow{},
		&api.MaintenanceWindow{},
		&api.Recommendation{},
	}
	for _, apiType := range apiTypes {
		mutator, validator, err := builder.WebhookManagedBy(server.Scheme).
			For(apiType).
			Complete()
		if err != nil {
			return err
		}
		if s.EnableMutatingWebhook && mutator != nil {
			cfg.AdmissionHooks = append(cfg.AdmissionHooks, mutator)
		}
		if s.EnableValidatingWebhook && validator != nil {
			cfg.AdmissionHooks = append(cfg.AdmissionHooks, validator)
		}
	}
	return nil
}
