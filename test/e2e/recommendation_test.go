package e2e_test

import (
	"time"

	kmapi "kmodules.xyz/client-go/api/v1"

	kubedbapi "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"

	"sigs.k8s.io/controller-runtime/pkg/client"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"

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
		createNewStandaloneMongoDB = func() *kubedbapi.MongoDB {
			By("Creating Standalone MongoDB")
			mg, err := f.CreateNewStandaloneMongoDB()
			Expect(err).NotTo(HaveOccurred())
			return mg
		}
		createDefaultMaintenanceWindow = func() {
			By("Creating Default MaintenanceWindow")
			Expect(f.CreateDefaultMaintenanceWindow()).Should(Succeed())
		}
		createMaintenanceWindow = func(days map[api.DayOfWeek][]api.TimeWindow, dates []api.DateWindow) *api.MaintenanceWindow {
			By("Creating MaintenanceWindow")
			mw, err := f.CreateMaintenanceWindow(days, dates)
			Expect(err).NotTo(HaveOccurred())
			return mw
		}
		createDefaultClusterMaintenanceWindow = func() {
			By("Creating Default Cluster MaintenanceWindow")
			Expect(f.CreateDefaultClusterMaintenanceWindow()).Should(Succeed())
		}
		createRecommendation = func(dbKey client.ObjectKey) *api.Recommendation {
			By("Creating a Recommendation for MongoDB restart OpsRequest")
			rcmd, err := f.CreateNewRecommendation(dbKey)
			Expect(err).NotTo(HaveOccurred())
			return rcmd
		}
		approveRecommendation = func(key client.ObjectKey) {
			By("Approving Recommendation")
			Expect(f.ApproveRecommendation(key)).Should(Succeed())
		}
		updateRecommendationApprovedWindow = func(key client.ObjectKey, aw *api.ApprovedWindow) {
			By("Updating Recommendation " + key.String() + "with given ApprovedWindow")
			Expect(f.UpdateRecommendationApprovedWindow(key, aw)).Should(Succeed())
		}
		waitingForRecommendationExecution = func(key client.ObjectKey) {
			By("Waiting for Recommendation execution")
			Expect(f.WaitForRecommendationToBeSucceeded(key)).Should(Succeed())
		}
		cleanupRecommendation = func(key client.ObjectKey) {
			By("Deleting Recommendation")
			Expect(f.DeleteRecommendation(key)).Should(Succeed())
		}
		cleanupDefaultMaintenanceWindow = func() {
			By("Deleting Default Maintenance Window")
			Expect(f.DeleteDefaultMaintenanceWindow()).Should(Succeed())
		}
		cleanupDefaultClusterMaintenanceWindow = func() {
			By("Deleting Default Cluster Maintenance Window")
			Expect(f.DeleteDefaultClusterMaintenanceWindow()).Should(Succeed())
		}
		cleanupMongoDB = func(key client.ObjectKey) {
			By("Deleting MongoDB" + key.String())
			Expect(f.DeleteMongoDB(key)).Should(Succeed())
		}
		cleanupMaintenanceWindow = func(key client.ObjectKey) {
			By("Deleting MaintenanceWindow " + key.String())
			Expect(f.DeleteMaintenanceWindow(key)).Should(Succeed())
		}
	)

	BeforeEach(func() {
		f = root.Invoke()
	})

	Describe("Supervisor operation", func() {
		Context("Successful execution of operation", func() {
			It("Should execute the operation successfully with default maintenance window", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				createDefaultMaintenanceWindow()
				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				approveRecommendation(key)
				waitingForRecommendationExecution(key)
				cleanupRecommendation(key)
				cleanupDefaultMaintenanceWindow()
				cleanupMongoDB(mgKey)
			})

			It("Should execute the operation successfully with default cluster maintenance window", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				createDefaultClusterMaintenanceWindow()
				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				approveRecommendation(key)
				waitingForRecommendationExecution(key)
				cleanupRecommendation(key)
				cleanupDefaultClusterMaintenanceWindow()
				cleanupMongoDB(mgKey)
			})

			It("Should execute the operation successfully with given maintenance window", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				days := f.GetAllDayOfWeekTimeWindow()
				mw := createMaintenanceWindow(days, nil)
				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				aw := &api.ApprovedWindow{
					MaintenanceWindow: &kmapi.TypedObjectReference{
						Name:      mw.Name,
						Namespace: mw.Namespace,
					},
				}
				updateRecommendationApprovedWindow(key, aw)
				approveRecommendation(key)
				waitingForRecommendationExecution(key)
				cleanupRecommendation(key)
				cleanupMaintenanceWindow(client.ObjectKey{Name: mw.Name, Namespace: mw.Namespace})
				cleanupMongoDB(mgKey)
			})

			It("Should execute the operation successfully with ApproveWindow type Immediate", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				aw := &api.ApprovedWindow{
					Window: api.Immediately,
				}
				updateRecommendationApprovedWindow(key, aw)
				approveRecommendation(key)
				waitingForRecommendationExecution(key)
				cleanupRecommendation(key)
				cleanupMongoDB(mgKey)
			})

			It("Should execute the operation successfully with ApproveWindow type NextAvailable", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				aw := &api.ApprovedWindow{
					Window: api.NextAvailable,
				}
				updateRecommendationApprovedWindow(key, aw)
				dates := f.GetCurrentDateWindow()
				mw := createMaintenanceWindow(nil, dates)
				approveRecommendation(key)
				waitingForRecommendationExecution(key)
				cleanupRecommendation(key)
				cleanupMaintenanceWindow(client.ObjectKey{Name: mw.Name, Namespace: mw.Namespace})
				cleanupMongoDB(mgKey)
			})

			It("Should execute the operation successfully with ApproveWindow type SpecificDates", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				aw := &api.ApprovedWindow{
					Window: api.SpecificDates,
					Dates:  f.GetCurrentDateWindow(),
				}
				updateRecommendationApprovedWindow(key, aw)
				approveRecommendation(key)
				waitingForRecommendationExecution(key)
				cleanupRecommendation(key)
				cleanupMongoDB(mgKey)
			})
		})
	})
})
