package framework

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	kmc "kmodules.xyz/client-go/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kmapi "kmodules.xyz/client-go/api/v1"
	kubedbapi "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (f *Framework) getOperationModel(filename string) []byte {
	By("Get Operation Model")
	file, err := os.Open(filename)
	Expect(err).NotTo(HaveOccurred())

	modelData, err := ioutil.ReadAll(file)
	Expect(err).NotTo(HaveOccurred())

	return modelData
}

func (f *Framework) CreateRecommendation() error {
	rcmd := &api.Recommendation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.getRecommendationName(),
			Namespace: f.getRecommendationNamespace(),
		},
		Spec: api.RecommendationSpec{
			Description: "MongoDB Database Restart",
			Target: core.TypedLocalObjectReference{
				APIGroup: pointer.StringP("kubedb.com/v1alpha2"),
				Kind:     kubedbapi.ResourceKindMongoDB,
				Name:     "mg-rs",
			},
			Operation: runtime.RawExtension{
				Raw: f.getOperationModel("../../testdata/mongodb-restart-ops-request.json"),
			},
			Recommender: kmapi.ObjectReference{
				Name: "kubedb-ops-manager",
			},
		},
	}

	return f.kc.Create(f.ctx, rcmd)
}

func (f *Framework) getRecommendationName() string {
	return f.name
}

func (f *Framework) getRecommendationNamespace() string {
	return f.namespace
}

func (f *Framework) WaitForRecommendationToBeSucceeded() error {
	return wait.PollImmediate(time.Second*5, time.Minute*30, func() (bool, error) {
		rcmd := &api.Recommendation{}
		if err := f.kc.Get(f.ctx, client.ObjectKey{Name: f.getRecommendationName(), Namespace: f.getRecommendationNamespace()}, rcmd); err != nil {
			return false, err
		}

		if rcmd.Status.Phase == api.Succeeded {
			return true, nil
		} else if rcmd.Status.Phase == api.Failed {
			return false, errors.New("operation failed")
		} else {
			return false, nil
		}
	})
}

func (f *Framework) ApproveRecommendation() error {
	rcmd := &api.Recommendation{}
	if err := f.kc.Get(f.ctx, client.ObjectKey{Name: f.getRecommendationName(), Namespace: f.getRecommendationNamespace()}, rcmd); err != nil {
		return err
	}
	_, _, err := kmc.PatchStatus(f.ctx, f.kc, rcmd, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*api.Recommendation)
		in.Status.ApprovalStatus = api.ApprovalApproved

		return in
	})
	return err
}

func (f *Framework) DeleteRecommendation() error {
	rcmd := &api.Recommendation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.getRecommendationName(),
			Namespace: f.getRecommendationNamespace(),
		},
	}

	return f.kc.Delete(f.ctx, rcmd)
}
