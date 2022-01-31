package e2e_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"kubeops.dev/supervisor/test/e2e/framework"
)

const (
	timeout  = 5 * time.Minute
	interval = 250 * time.Millisecond
)

var _ = Describe("Supervisor E2E Testing", func() {
	var (
		f *framework.Invocation
	)

	var (
		createDefaultMaintenanceWindow = func() {
			By("Creating Default MaintenanceWindow")
			Expect(f.CreateDefaultMaintenanceWindow()).Should(Succeed())
		}
		createRecommendation = func() {
			By("Creating a Recommendation for MongoDB restart OpsRequest")
			Expect(f.CreateRecommendation()).Should(Succeed())
		}
		approveRecommendation = func() {
			By("Approving Recommendation")
			Expect(f.ApproveRecommendation()).Should(Succeed())
		}
		waitingForRecommendationExecution = func() {
			By("Waiting for Recommendation execution")
			Expect(f.WaitForRecommendationToBeSucceeded()).Should(Succeed())
		}
		cleanupRecommendation = func() {
			By("Deleting Recommendation")
			Expect(f.DeleteRecommendation()).Should(Succeed())
		}
		cleanupDefaultMaintenanceWindow = func() {
			By("Deleting Default Maintenance Window")
			Expect(f.DeleteDefaultMaintenanceWindow()).Should(Succeed())
		}
	)

	BeforeEach(func() {
		f = root.Invoke()
	})

	Describe("Supervisor operation", func() {
		Context("Successful execution of operation with default MaintenanceWindow", func() {
			It("Should execute the operation successfully", func() {
				createDefaultMaintenanceWindow()
				createRecommendation()
				approveRecommendation()
				waitingForRecommendationExecution()
				cleanupRecommendation()
				cleanupDefaultMaintenanceWindow()
			})
		})
	})
})
