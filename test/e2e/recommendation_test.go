package e2e_test

import (
	"errors"
	"fmt"
	"time"

	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kmapi "kmodules.xyz/client-go/api/v1"

	kubedbapi "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"

	"sigs.k8s.io/controller-runtime/pkg/client"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"kubeops.dev/supervisor/test/e2e/framework"
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
		createApprovalPolicy = func(target []api.TargetRef, mwRef client.ObjectKey) *api.ApprovalPolicy {
			By("Creating an ApprovalPolicy")
			ap, err := f.CreateNewApprovalPolicy(target, mwRef)
			Expect(err).NotTo(HaveOccurred())
			return ap
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
		ensureQueuePerNamespaceParallelism = func(stopCh chan bool, errCh chan bool) {
			By("Ensuring Parallelism")
			go func() {
				err := f.EnsureQueuePerNamespaceParallelism(stopCh)
				if err != nil {
					errCh <- true
				}
			}()
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
		cleanupApprovalPolicy = func(key client.ObjectKey) {
			By("Deleting Approval Policy " + key.String())
			Expect(f.DeleteApprovalPolicy(key)).Should(Succeed())
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
				defer cleanupMongoDB(mgKey)

				createDefaultMaintenanceWindow()
				defer cleanupDefaultMaintenanceWindow()

				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				defer cleanupRecommendation(key)

				approveRecommendation(key)
				waitingForRecommendationExecution(key)
			})

			It("Should execute the operation successfully with default cluster maintenance window", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				defer cleanupMongoDB(mgKey)

				createDefaultClusterMaintenanceWindow()
				defer cleanupDefaultClusterMaintenanceWindow()

				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				defer cleanupRecommendation(key)

				approveRecommendation(key)
				waitingForRecommendationExecution(key)
			})

			It("Should execute the operation successfully with given maintenance window", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				defer cleanupMongoDB(mgKey)

				days := f.GetAllDayOfWeekTimeWindow()
				mw := createMaintenanceWindow(days, nil)
				defer cleanupMaintenanceWindow(client.ObjectKey{Name: mw.Name, Namespace: mw.Namespace})

				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				defer cleanupRecommendation(key)

				aw := &api.ApprovedWindow{
					MaintenanceWindow: &kmapi.TypedObjectReference{
						Name:      mw.Name,
						Namespace: mw.Namespace,
					},
				}
				updateRecommendationApprovedWindow(key, aw)

				approveRecommendation(key)
				waitingForRecommendationExecution(key)
			})

			It("Should execute the operation successfully with ApproveWindow type Immediate", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				defer cleanupMongoDB(mgKey)

				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				defer cleanupRecommendation(key)

				aw := &api.ApprovedWindow{
					Window: api.Immediately,
				}
				updateRecommendationApprovedWindow(key, aw)

				approveRecommendation(key)
				waitingForRecommendationExecution(key)
			})

			It("Should execute the operation successfully with ApproveWindow type NextAvailable", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				defer cleanupMongoDB(mgKey)

				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				defer cleanupRecommendation(key)

				aw := &api.ApprovedWindow{
					Window: api.NextAvailable,
				}
				updateRecommendationApprovedWindow(key, aw)

				dates := f.GetCurrentDateWindow()
				mw := createMaintenanceWindow(nil, dates)
				defer cleanupMaintenanceWindow(client.ObjectKey{Name: mw.Name, Namespace: mw.Namespace})

				approveRecommendation(key)
				waitingForRecommendationExecution(key)
			})

			It("Should execute the operation successfully with ApproveWindow type SpecificDates", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				defer cleanupMongoDB(mgKey)

				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				defer cleanupRecommendation(key)

				aw := &api.ApprovedWindow{
					Window: api.SpecificDates,
					Dates:  f.GetCurrentDateWindow(),
				}
				updateRecommendationApprovedWindow(key, aw)

				approveRecommendation(key)
				waitingForRecommendationExecution(key)
			})

			It("Should execute the operation successfully with auto approval by ApprovalPolicy", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				defer cleanupMongoDB(mgKey)

				dates := f.GetCurrentDateWindow()
				mw := createMaintenanceWindow(nil, dates)
				defer cleanupMaintenanceWindow(client.ObjectKey{Name: mw.Name, Namespace: mw.Namespace})

				target := []api.TargetRef{
					{
						GroupKind: metav1.GroupKind{
							Group: kubedbapi.SchemeGroupVersion.Group,
							Kind:  kubedbapi.ResourceKindMongoDB,
						},
						Operations: []api.Operation{
							{
								GroupKind: metav1.GroupKind{
									Group: opsapi.SchemeGroupVersion.Group,
									Kind:  opsapi.ResourceKindMongoDBOpsRequest,
								},
								Type: opsapi.Restart,
							},
						},
					},
				}
				ap := createApprovalPolicy(target, client.ObjectKey{Name: mw.Name, Namespace: mw.Namespace})
				defer cleanupApprovalPolicy(client.ObjectKey{Name: ap.Name, Namespace: ap.Namespace})

				rcmd := createRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				defer cleanupRecommendation(key)

				waitingForRecommendationExecution(key)
			})

			FIt("Should execute two operations successfully maintaining default QueuePerNamespace Parallelism", func() {
				mg1 := createNewStandaloneMongoDB()
				mg1Key := client.ObjectKey{Name: mg1.Name, Namespace: mg1.Namespace}
				defer cleanupMongoDB(mg1Key)

				mg2 := createNewStandaloneMongoDB()
				mg2Key := client.ObjectKey{Name: mg2.Name, Namespace: mg2.Namespace}
				defer cleanupMongoDB(mg2Key)

				rcmd1 := createRecommendation(mg1Key)
				rcmd1Key := client.ObjectKey{Name: rcmd1.Name, Namespace: rcmd1.Namespace}
				defer cleanupRecommendation(rcmd1Key)

				rcmd2 := createRecommendation(mg2Key)
				rcmd2Key := client.ObjectKey{Name: rcmd2.Name, Namespace: rcmd2.Namespace}
				defer cleanupRecommendation(rcmd2Key)

				aw := &api.ApprovedWindow{
					Window: api.Immediately,
				}
				updateRecommendationApprovedWindow(rcmd1Key, aw)
				updateRecommendationApprovedWindow(rcmd2Key, aw)

				approveRecommendation(rcmd1Key)
				time.Sleep(time.Second * 5)
				approveRecommendation(rcmd2Key)

				stopCh := make(chan bool)
				errCh := make(chan bool, 2)
				ensureQueuePerNamespaceParallelism(stopCh, errCh)

				waitingForRecommendationExecution(rcmd1Key)
				waitingForRecommendationExecution(rcmd2Key)

				if len(errCh) == 0 {
					stopCh <- true
				}

				errFunc := func() error {
					fmt.Println(len(errCh))
					if len(errCh) > 0 {
						return errors.New("parallelism is not maintained")
					}
					return nil
				}
				err := errFunc()
				Expect(err).ShouldNot(HaveOccurred())
			})
		})
	})
})
