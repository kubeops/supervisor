package framework

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"time"

	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gomodules.xyz/pointer"
	"gomodules.xyz/x/crypto/rand"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kmapi "kmodules.xyz/client-go/api/v1"
	kmc "kmodules.xyz/client-go/client"
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

func (f *Framework) getMongoDBRestartOpsRequest(dbKey client.ObjectKey) *opsapi.MongoDBOpsRequest {
	return &opsapi.MongoDBOpsRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       opsapi.ResourceKindMongoDBOpsRequest,
			APIVersion: opsapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dbKey.Namespace,
		},
		Spec: opsapi.MongoDBOpsRequestSpec{
			DatabaseRef: core.LocalObjectReference{
				Name: dbKey.Name,
			},
			Type: opsapi.Restart,
		},
	}
}

func (f *Framework) newRecommendation(dbKey client.ObjectKey) (*api.Recommendation, error) {
	opsData := f.getMongoDBRestartOpsRequest(dbKey)
	byteData, err := json.Marshal(opsData)
	if err != nil {
		return nil, err
	}
	return &api.Recommendation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("supervisor"),
			Namespace: f.getRecommendationNamespace(),
		},
		Spec: api.RecommendationSpec{
			Description: "MongoDB Database Restart",
			Target: core.TypedLocalObjectReference{
				APIGroup: pointer.StringP("kubedb.com/v1alpha2"),
				Kind:     kubedbapi.ResourceKindMongoDB,
				Name:     dbKey.Name,
			},
			Operation: runtime.RawExtension{
				Raw: byteData,
			},
			Recommender: kmapi.ObjectReference{
				Name: "kubedb-ops-manager",
			},
		},
	}, nil
}

func (f *Framework) CreateNewRecommendation(dbKey client.ObjectKey) (*api.Recommendation, error) {
	rcmd, err := f.newRecommendation(dbKey)
	if err != nil {
		return nil, err
	}
	if err := f.kc.Create(f.ctx, rcmd); err != nil {
		return nil, err
	}

	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		obj := &api.Recommendation{}
		key := client.ObjectKey{Name: rcmd.Name, Namespace: rcmd.Namespace}
		if err := f.kc.Get(f.ctx, key, obj); err != nil {
			return false, client.IgnoreNotFound(err)
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}
	return rcmd, nil
}

func (f *Framework) getRecommendationNamespace() string {
	return f.namespace
}

func (f *Framework) WaitForRecommendationToBeSucceeded(key client.ObjectKey) error {
	return wait.PollImmediate(time.Second*5, time.Minute*30, func() (bool, error) {
		rcmd := &api.Recommendation{}
		if err := f.kc.Get(f.ctx, key, rcmd); err != nil {
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

func (f *Framework) ApproveRecommendation(key client.ObjectKey) error {
	rcmd := &api.Recommendation{}
	if err := f.kc.Get(f.ctx, key, rcmd); err != nil {
		return err
	}
	_, _, err := kmc.PatchStatus(f.ctx, f.kc, rcmd, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*api.Recommendation)
		in.Status.ApprovalStatus = api.ApprovalApproved

		return in
	})
	if err != nil {
		return err
	}

	return wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		obj := &api.Recommendation{}
		if err := f.kc.Get(f.ctx, key, obj); err != nil {
			return false, err
		}

		if obj.Status.ApprovalStatus == api.ApprovalApproved {
			return true, nil
		}
		return false, nil
	})
}

func (f *Framework) DeleteRecommendation(key client.ObjectKey) error {
	rcmd := &api.Recommendation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}

	return f.kc.Delete(f.ctx, rcmd)
}

func (f *Framework) UpdateRecommendationApprovedWindow(key client.ObjectKey, aw *api.ApprovedWindow) error {
	rcmd := &api.Recommendation{}
	if err := f.kc.Get(f.ctx, key, rcmd); err != nil {
		return err
	}

	_, _, err := kmc.PatchStatus(f.ctx, f.kc, rcmd, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*api.Recommendation)
		in.Status.ApprovedWindow = aw
		return in
	})
	return err
}
