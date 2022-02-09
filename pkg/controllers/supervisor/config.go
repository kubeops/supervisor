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

package supervisor

import (
	"errors"
	"flag"
	"time"

	"github.com/spf13/pflag"
)

type RecommendationReconcileConfig struct {
	MaxConcurrentReconcile int
	RequeueAfterDuration   time.Duration
	MaxRetryOnFailure      int
	RetryAfterDuration     time.Duration
	BeforeDeadlineDuration time.Duration
	WebhookCertDir         string
}

func NewRecommendationReconcileConfig() *RecommendationReconcileConfig {
	return &RecommendationReconcileConfig{}
}

func (c *RecommendationReconcileConfig) AddGoFlags(fs *flag.FlagSet) {
	fs.IntVar(&c.MaxConcurrentReconcile, "max-concurrent-reconcile", c.MaxConcurrentReconcile, "Maximum number of Recommendation object that will be reconciled concurrently")
	fs.DurationVar(&c.RequeueAfterDuration, "requeue-after-duration", c.RequeueAfterDuration, "Duration after the Recommendation object will be requeue when it is waiting for MaintenanceWindow. The flag accepts a value acceptable to time.ParseDuration. Ref: https://pkg.go.dev/time#ParseDuration")
	fs.IntVar(&c.MaxRetryOnFailure, "max-retry-on-failure", c.MaxRetryOnFailure, "Maximum number of retry on any kind of failure in Recommendation execution")
	fs.DurationVar(&c.RetryAfterDuration, "retry-after-duration", c.RetryAfterDuration, "Duration after the failure events will be requeue again. The flag accepts a value acceptable to time.ParseDuration. Ref: https://pkg.go.dev/time#ParseDuration")
	fs.DurationVar(&c.BeforeDeadlineDuration, "before-deadline-duration", c.BeforeDeadlineDuration, "When there is less time than `BeforeDeadlineDuration` before deadline, Recommendations are free to execute regardless of Parallelism")
	fs.StringVar(&c.WebhookCertDir, "webhook-cert-dir", c.WebhookCertDir, "Cert files directory location for admission webhook server")
}

func (c *RecommendationReconcileConfig) AddFlags(fs *pflag.FlagSet) {
	pfs := flag.NewFlagSet("recommendation-controller", flag.ExitOnError)
	c.AddGoFlags(pfs)
	fs.AddGoFlagSet(pfs)
}

func (c *RecommendationReconcileConfig) Validate() []error {
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
