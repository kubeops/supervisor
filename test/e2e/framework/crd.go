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
