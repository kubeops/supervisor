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

package deadline_manager

import (
	"time"

	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"

	"github.com/jonboulle/clockwork"
)

type manager struct {
	rcmd  *supervisorv1alpha1.Recommendation
	clock clockwork.Clock
}

func NewManager(rcmd *supervisorv1alpha1.Recommendation, clock clockwork.Clock) *manager {
	return &manager{
		rcmd:  rcmd,
		clock: clock,
	}
}

func (m *manager) IsDeadlineLessThanADay() bool {
	return m.IsDeadlineLessThan(time.Hour * 24)
}

func (m *manager) IsDeadlineLessThan(duration time.Duration) bool {
	if m.rcmd.Spec.Deadline == nil {
		return false
	}
	now := m.clock.Now().UTC().Unix()
	deadline := m.rcmd.Spec.Deadline.UTC().Unix()
	dur := int64(duration.Seconds())

	return deadline-now < dur
}
