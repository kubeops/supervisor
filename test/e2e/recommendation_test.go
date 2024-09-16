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

package e2e_test

import (
	"errors"
	"time"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"kubeops.dev/supervisor/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gomodules.xyz/pointer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
	kubedbv1 "kubedb.dev/apimachinery/apis/kubedb/v1"
	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Supervisor E2E Testing", func() {
	var f *framework.Invocation

	var (
		createNewStandaloneMongoDB = func() *kubedbv1.MongoDB {
			By("Creating Standalone MongoDB")
			mg, err := f.CreateNewStandaloneMongoDB()
			Expect(err).NotTo(HaveOccurred())
			return mg
		}
		createNewStandalonePostgres = func() *kubedbv1.Postgres {
			By("Creating Standalone Postgres")
			pg, err := f.CreateNewStandalonePostgres()
			Expect(err).NotTo(HaveOccurred())
			return pg
		}
		createDefaultMaintenanceWindow = func() {
			By("Creating Default MaintenanceWindow")
			Expect(f.CreateDefaultMaintenanceWindow()).Should(Succeed())
		}
		getDefaultMaintenanceWindow = func() *api.MaintenanceWindow {
			By("Getting Default MaintenanceWindow")
			mw, err := f.GetDefaultMaintenanceWindow()
			Expect(err).NotTo(HaveOccurred())
			return mw
		}
		checkDefaultMaintenanceWindowAnnotation = func() {
			By("Checking Default MaintenanceWindow Annotation")
			Eventually(func() bool {
				mw := getDefaultMaintenanceWindow()
				_, found := mw.Annotations[api.DefaultMaintenanceWindowKey]
				return found
			}).WithTimeout(time.Minute).WithPolling(time.Second).Should(BeTrue())
		}
		createTwoDefaultMaintenanceWindow = func() error {
			By("Creating First Default MaintenanceWindow")
			err := f.CreateDefaultMaintenanceWindow()
			Expect(err).ShouldNot(HaveOccurred())

			By("Creating Second Default MaintenanceWindow")
			err = f.CreateDefaultMaintenanceWindow()
			return err
		}
		createMaintenanceWindow = func(days map[api.DayOfWeek][]api.TimeWindow, dates []api.DateWindow) *api.MaintenanceWindow {
			By("Creating MaintenanceWindow")
			mw, err := f.CreateMaintenanceWindow(days, dates)
			Expect(err).NotTo(HaveOccurred())
			return mw
		}
		createDefaultClusterMaintenanceWindow = func(days map[api.DayOfWeek][]api.TimeWindow, dates []api.DateWindow) {
			By("Creating Default Cluster MaintenanceWindow")
			Expect(f.CreateDefaultClusterMaintenanceWindow(days, dates)).Should(Succeed())
		}
		getDefaultClusterMaintenanceWindow = func() *api.ClusterMaintenanceWindow {
			By("Getting Default Cluster MaintenanceWindow")
			cmw, err := f.GetDefaultClusterMaintenanceWindow()
			Expect(err).NotTo(HaveOccurred())
			return cmw
		}
		checkDefaultClusterMaintenanceWindowAnnotation = func() {
			By("Checking Default ClusterMaintenanceWindow Annotation")
			Eventually(func() bool {
				mw := getDefaultClusterMaintenanceWindow()
				_, found := mw.Annotations[api.DefaultClusterMaintenanceWindowKey]
				return found
			}).WithTimeout(time.Minute).WithPolling(time.Second).Should(BeTrue())
		}
		createMongoDBRecommendation = func(dbKey client.ObjectKey) *api.Recommendation {
			By("Creating a Recommendation for MongoDB restart OpsRequest")
			rcmd, err := f.CreateNewMongoDBRecommendation(dbKey)
			Expect(err).NotTo(HaveOccurred())
			return rcmd
		}
		createPostgresRecommendation = func(dbKey client.ObjectKey) *api.Recommendation {
			By("Creating a Recommendation for Postgres restart OpsRequest")
			rcmd, err := f.CreateNewPostgresRecommendation(dbKey)
			Expect(err).NotTo(HaveOccurred())
			return rcmd
		}
		createRecommendationWithDeadline = func(dbKey client.ObjectKey, deadline *metav1.Time) *api.Recommendation {
			By("Creating a Recommendation for MongoDB restart OpsRequest with deadline")
			rcmd, err := f.CreateNewRecommendationWithDeadline(dbKey, deadline)
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
		waitingForRecommendationToBeSucceeded = func(key client.ObjectKey) {
			By("Waiting for Recommendation execution")
			Expect(f.WaitForRecommendationToBeSucceeded(key)).Should(Succeed())
		}
		checkRecommendationExecution = func(key client.ObjectKey, timeout time.Duration, interval time.Duration) {
			defer GinkgoRecover()
			By("Checking for Recommendation execution")
			Expect(f.CheckRecommendationExecution(key, timeout, interval)).Should(Succeed())
		}
		ensureQueuePerNamespaceParallelism = func(stopCh chan bool, errCh chan bool) {
			By("Ensuring QueuePerNamespace Parallelism")
			go func() {
				err := f.EnsureQueuePerNamespaceParallelism(stopCh)
				if err != nil {
					errCh <- true
				}
			}()
		}
		ensureQueuePerTargetParallelism = func(stopCh chan bool, errCh chan bool, target metav1.GroupKind, ns string) {
			By("Ensuring QueuePerTarget Parallelism")
			go func() {
				err := f.EnsureQueuePerTargetParallelism(stopCh, target, ns)
				if err != nil {
					errCh <- true
				}
			}()
		}
		updateRecommendationParallelism = func(key client.ObjectKey, par api.Parallelism) {
			By("Updating Recommendation Parallelism to " + string(par))
			Expect(f.UpdateRecommendationParallelism(key, par)).Should(Succeed())
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
		cleanupPostgres = func(key client.ObjectKey) {
			By("Deleting Postgres " + key.String())
			Expect(f.DeletePostgres(key)).Should(Succeed())
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

				rcmd := createMongoDBRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				defer cleanupRecommendation(key)

				approveRecommendation(key)
				waitingForRecommendationToBeSucceeded(key)
			})

			It("Should execute the operation successfully with default cluster maintenance window", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				defer cleanupMongoDB(mgKey)

				days := f.GetAllDayOfWeekTimeWindow()
				createDefaultClusterMaintenanceWindow(days, nil)
				defer cleanupDefaultClusterMaintenanceWindow()

				rcmd := createMongoDBRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				defer cleanupRecommendation(key)

				approveRecommendation(key)
				waitingForRecommendationToBeSucceeded(key)
			})

			It("Should execute the operation successfully with given maintenance window", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				defer cleanupMongoDB(mgKey)

				days := f.GetAllDayOfWeekTimeWindow()
				mw := createMaintenanceWindow(days, nil)
				defer cleanupMaintenanceWindow(client.ObjectKey{Name: mw.Name, Namespace: mw.Namespace})

				rcmd := createMongoDBRecommendation(mgKey)
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
				waitingForRecommendationToBeSucceeded(key)
			})

			It("Should execute the operation successfully with ApproveWindow type Immediate", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				defer cleanupMongoDB(mgKey)

				rcmd := createMongoDBRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				defer cleanupRecommendation(key)

				aw := &api.ApprovedWindow{
					Window: api.Immediate,
				}
				updateRecommendationApprovedWindow(key, aw)

				approveRecommendation(key)
				waitingForRecommendationToBeSucceeded(key)
			})

			It("Should execute the operation successfully with ApproveWindow type NextAvailable", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				defer cleanupMongoDB(mgKey)

				rcmd := createMongoDBRecommendation(mgKey)
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
				waitingForRecommendationToBeSucceeded(key)
			})

			It("Should execute the operation successfully with ApproveWindow type SpecificDates", func() {
				mg := createNewStandaloneMongoDB()
				mgKey := client.ObjectKey{Name: mg.Name, Namespace: mg.Namespace}
				defer cleanupMongoDB(mgKey)

				rcmd := createMongoDBRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				defer cleanupRecommendation(key)

				aw := &api.ApprovedWindow{
					Window: api.SpecificDates,
					Dates:  f.GetCurrentDateWindow(),
				}
				updateRecommendationApprovedWindow(key, aw)

				approveRecommendation(key)
				waitingForRecommendationToBeSucceeded(key)
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
							Group: kubedbv1.SchemeGroupVersion.Group,
							Kind:  kubedbv1.ResourceKindMongoDB,
						},
						Operations: []api.Operation{
							{
								GroupKind: metav1.GroupKind{
									Group: opsapi.SchemeGroupVersion.Group,
									Kind:  opsapi.ResourceKindMongoDBOpsRequest,
								},
							},
						},
					},
				}
				ap := createApprovalPolicy(target, client.ObjectKey{Name: mw.Name, Namespace: mw.Namespace})
				defer cleanupApprovalPolicy(client.ObjectKey{Name: ap.Name, Namespace: ap.Namespace})

				rcmd := createMongoDBRecommendation(mgKey)
				key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
				defer cleanupRecommendation(key)

				waitingForRecommendationToBeSucceeded(key)
			})

			It("Should execute two operations successfully maintaining default QueuePerNamespace Parallelism", func() {
				mg1 := createNewStandaloneMongoDB()
				mg1Key := client.ObjectKey{Name: mg1.Name, Namespace: mg1.Namespace}
				defer cleanupMongoDB(mg1Key)

				mg2 := createNewStandaloneMongoDB()
				mg2Key := client.ObjectKey{Name: mg2.Name, Namespace: mg2.Namespace}
				defer cleanupMongoDB(mg2Key)

				rcmd1 := createMongoDBRecommendation(mg1Key)
				rcmd1Key := client.ObjectKey{Name: rcmd1.Name, Namespace: rcmd1.Namespace}
				defer cleanupRecommendation(rcmd1Key)

				rcmd2 := createMongoDBRecommendation(mg2Key)
				rcmd2Key := client.ObjectKey{Name: rcmd2.Name, Namespace: rcmd2.Namespace}
				defer cleanupRecommendation(rcmd2Key)

				dates := f.GetDateWindowsAfter(time.Minute, time.Hour)
				mw := createMaintenanceWindow(nil, dates)
				defer cleanupMaintenanceWindow(client.ObjectKey{Name: mw.Name, Namespace: mw.Namespace})

				aw := &api.ApprovedWindow{
					MaintenanceWindow: &kmapi.TypedObjectReference{
						Name:      mw.Name,
						Namespace: mw.Namespace,
					},
				}
				updateRecommendationApprovedWindow(rcmd1Key, aw)
				updateRecommendationApprovedWindow(rcmd2Key, aw)

				approveRecommendation(rcmd1Key)
				approveRecommendation(rcmd2Key)

				stopCh := make(chan bool)
				errCh := make(chan bool, 2)
				ensureQueuePerNamespaceParallelism(stopCh, errCh)

				waitingForRecommendationToBeSucceeded(rcmd1Key)
				waitingForRecommendationToBeSucceeded(rcmd2Key)

				if len(errCh) == 0 {
					stopCh <- true
				}

				errFunc := func() error {
					if len(errCh) > 0 {
						return errors.New("QueuePerNamespace parallelism is not maintained")
					}
					return nil
				}
				err := errFunc()
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("Should execute two operations parallel with immediate deadline", func() {
				mg1 := createNewStandaloneMongoDB()
				mg1Key := client.ObjectKey{Name: mg1.Name, Namespace: mg1.Namespace}
				defer cleanupMongoDB(mg1Key)

				mg2 := createNewStandaloneMongoDB()
				mg2Key := client.ObjectKey{Name: mg2.Name, Namespace: mg2.Namespace}
				defer cleanupMongoDB(mg2Key)

				rcmd1 := createRecommendationWithDeadline(mg1Key, &metav1.Time{Time: time.Now().Add(time.Minute * 10)})
				rcmd1Key := client.ObjectKey{Name: rcmd1.Name, Namespace: rcmd1.Namespace}
				defer cleanupRecommendation(rcmd1Key)

				rcmd2 := createRecommendationWithDeadline(mg2Key, &metav1.Time{Time: time.Now().Add(time.Minute * 10)})
				rcmd2Key := client.ObjectKey{Name: rcmd2.Name, Namespace: rcmd2.Namespace}
				defer cleanupRecommendation(rcmd2Key)

				dates := f.GetDateWindowsAfter(time.Second, time.Hour)
				mw := createMaintenanceWindow(nil, dates)
				defer cleanupMaintenanceWindow(client.ObjectKey{Name: mw.Name, Namespace: mw.Namespace})
				aw := &api.ApprovedWindow{
					MaintenanceWindow: &kmapi.TypedObjectReference{
						Namespace: mw.Namespace,
						Name:      mw.Name,
					},
				}
				updateRecommendationApprovedWindow(rcmd1Key, aw)
				updateRecommendationApprovedWindow(rcmd2Key, aw)

				approveRecommendation(rcmd1Key)
				approveRecommendation(rcmd2Key)

				// Though defaultParallelism is Namespace, both Recommendation should start executing operation simultaneously
				// to ensure their job done before deadline.
				go checkRecommendationExecution(rcmd1Key, time.Minute*5, time.Second)
				go checkRecommendationExecution(rcmd2Key, time.Minute*5, time.Second)

				waitingForRecommendationToBeSucceeded(rcmd1Key)
				waitingForRecommendationToBeSucceeded(rcmd2Key)
			})

			It("Should execute two operations successfully maintaining QueuePerTarget Parallelism", func() {
				pg1 := createNewStandalonePostgres()
				pg1Key := client.ObjectKey{Name: pg1.Name, Namespace: pg1.Namespace}
				defer cleanupPostgres(pg1Key)

				pg2 := createNewStandalonePostgres()
				pg2Key := client.ObjectKey{Name: pg2.Name, Namespace: pg2.Namespace}
				defer cleanupPostgres(pg2Key)

				rcmd1 := createPostgresRecommendation(pg1Key)
				rcmd1Key := client.ObjectKey{Name: rcmd1.Name, Namespace: rcmd1.Namespace}
				defer cleanupRecommendation(rcmd1Key)

				rcmd2 := createPostgresRecommendation(pg2Key)
				rcmd2Key := client.ObjectKey{Name: rcmd2.Name, Namespace: rcmd2.Namespace}
				defer cleanupRecommendation(rcmd2Key)

				dates := f.GetDateWindowsAfter(time.Minute, time.Hour)
				mw := createMaintenanceWindow(nil, dates)
				defer cleanupMaintenanceWindow(client.ObjectKey{Name: mw.Name, Namespace: mw.Namespace})

				aw := &api.ApprovedWindow{
					MaintenanceWindow: &kmapi.TypedObjectReference{
						Name:      mw.Name,
						Namespace: mw.Namespace,
					},
				}
				updateRecommendationApprovedWindow(rcmd1Key, aw)
				updateRecommendationApprovedWindow(rcmd2Key, aw)

				updateRecommendationParallelism(rcmd1Key, api.QueuePerTarget)
				updateRecommendationParallelism(rcmd2Key, api.QueuePerTarget)

				approveRecommendation(rcmd1Key)
				approveRecommendation(rcmd2Key)

				stopCh := make(chan bool)
				errCh := make(chan bool, 2)
				target := metav1.GroupKind{Group: kubedbv1.SchemeGroupVersion.Group, Kind: kubedbv1.ResourceKindPostgres}
				ensureQueuePerTargetParallelism(stopCh, errCh, target, "")

				waitingForRecommendationToBeSucceeded(rcmd1Key)
				waitingForRecommendationToBeSucceeded(rcmd2Key)

				if len(errCh) == 0 {
					stopCh <- true
				}

				errFunc := func() error {
					if len(errCh) > 0 {
						return errors.New("QueuePerTarget parallelism is not maintained")
					}
					return nil
				}
				err := errFunc()
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("Should execute two operations successfully maintaining QueuePerTargetAndNamespace Parallelism", func() {
				mg1 := createNewStandaloneMongoDB()
				mg1Key := client.ObjectKey{Name: mg1.Name, Namespace: mg1.Namespace}
				defer cleanupMongoDB(mg1Key)

				mg2 := createNewStandaloneMongoDB()
				mg2Key := client.ObjectKey{Name: mg2.Name, Namespace: mg2.Namespace}
				defer cleanupMongoDB(mg2Key)

				rcmd1 := createMongoDBRecommendation(mg1Key)
				rcmd1Key := client.ObjectKey{Name: rcmd1.Name, Namespace: rcmd1.Namespace}
				defer cleanupRecommendation(rcmd1Key)

				rcmd2 := createMongoDBRecommendation(mg2Key)
				rcmd2Key := client.ObjectKey{Name: rcmd2.Name, Namespace: rcmd2.Namespace}
				defer cleanupRecommendation(rcmd2Key)

				dates := f.GetDateWindowsAfter(time.Minute, time.Hour)
				mw := createMaintenanceWindow(nil, dates)
				defer cleanupMaintenanceWindow(client.ObjectKey{Name: mw.Name, Namespace: mw.Namespace})

				aw := &api.ApprovedWindow{
					MaintenanceWindow: &kmapi.TypedObjectReference{
						Name:      mw.Name,
						Namespace: mw.Namespace,
					},
				}
				updateRecommendationApprovedWindow(rcmd1Key, aw)
				updateRecommendationApprovedWindow(rcmd2Key, aw)

				updateRecommendationParallelism(rcmd1Key, api.QueuePerTargetAndNamespace)
				updateRecommendationParallelism(rcmd2Key, api.QueuePerTargetAndNamespace)

				approveRecommendation(rcmd1Key)
				approveRecommendation(rcmd2Key)

				stopCh := make(chan bool)
				errCh := make(chan bool, 2)
				target := metav1.GroupKind{Group: kubedbv1.SchemeGroupVersion.Group, Kind: kubedbv1.ResourceKindMongoDB}
				ensureQueuePerTargetParallelism(stopCh, errCh, target, f.Namespace())

				waitingForRecommendationToBeSucceeded(rcmd1Key)
				waitingForRecommendationToBeSucceeded(rcmd2Key)

				if len(errCh) == 0 {
					stopCh <- true
				}

				errFunc := func() error {
					if len(errCh) > 0 {
						return errors.New("QueuePerTargetAndNamespace parallelism is not maintained")
					}
					return nil
				}
				err := errFunc()
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		Context("Failure events", func() {
			It("Should encounter error while creating multiple default MaintenanceWindow in same namespace", func() {
				err := createTwoDefaultMaintenanceWindow()
				defer cleanupDefaultMaintenanceWindow()
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("Webhook test", func() {
			It("Should create Default MaintenanceWindow and validate default maintenance window annotation", func() {
				createDefaultMaintenanceWindow()
				defer cleanupDefaultMaintenanceWindow()
				checkDefaultMaintenanceWindowAnnotation()
			})

			It("Should create Default Cluster MaintenanceWindow and validate default cluster maintenance window annotation", func() {
				days := f.GetAllDayOfWeekTimeWindow()
				createDefaultClusterMaintenanceWindow(days, nil)
				defer cleanupDefaultClusterMaintenanceWindow()
				checkDefaultClusterMaintenanceWindowAnnotation()
			})

			It("Should create Recommendation without backoffLimit and validate default backoffLimit", func() {
				obj := createRecommendationWithDeadline(client.ObjectKey{Name: "test", Namespace: "test"}, nil)
				defer cleanupRecommendation(client.ObjectKey{Namespace: obj.Namespace, Name: obj.Name})
				Expect(pointer.Int32(obj.Spec.BackoffLimit)).Should(Equal(int32(api.DefaultBackoffLimit)))
			})
		})
	})
})
