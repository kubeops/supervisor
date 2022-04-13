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

package framework

import (
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	supervisorv1alpha1 "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
)

func (f *Framework) EnsureCRD() GomegaAsyncAssertion {
	return Eventually(func() error {
		if err := f.kc.List(f.ctx, &supervisorv1alpha1.RecommendationList{}); err != nil {
			return fmt.Errorf("CRD Recommendation is not ready, Reason: %v", err)
		}

		if err := f.kc.List(f.ctx, &supervisorv1alpha1.MaintenanceWindowList{}); err != nil {
			return fmt.Errorf("CRD MaintainenceWindow is not ready, Reason: %v", err)
		}

		if err := f.kc.List(f.ctx, &supervisorv1alpha1.ClusterMaintenanceWindowList{}); err != nil {
			return fmt.Errorf("CRD ClusterMaintainenceWindow is not ready, Reason: %v", err)
		}

		if err := f.kc.List(f.ctx, &supervisorv1alpha1.ApprovalPolicyList{}); err != nil {
			return fmt.Errorf("CRD ApprovalPolicy is not ready, Reason: %v", err)
		}

		return nil
	},
		time.Minute*2,
		time.Second*10,
	)
}
