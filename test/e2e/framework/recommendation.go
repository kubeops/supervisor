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
	"context"
	"encoding/json"
	"errors"
	"time"

	opsapi "kubedb.dev/apimachinery/apis/ops/v1alpha1"

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

type TransformFunc func(obj *api.Recommendation) *api.Recommendation

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

func (f *Framework) getPostgresRestartOpsRequest(dbKey client.ObjectKey) *opsapi.PostgresOpsRequest {
	return &opsapi.PostgresOpsRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       opsapi.ResourceKindPostgresOpsRequest,
			APIVersion: opsapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dbKey.Namespace,
		},
		Spec: opsapi.PostgresOpsRequestSpec{
			DatabaseRef: core.LocalObjectReference{
				Name: dbKey.Name,
			},
			Type: opsapi.Restart,
		},
	}
}

func (f *Framework) newMongoDBRecommendation(dbKey client.ObjectKey, deadline *metav1.Time) (*api.Recommendation, error) {
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
				APIGroup: pointer.StringP(kubedbapi.SchemeGroupVersion.Group),
				Kind:     kubedbapi.ResourceKindMongoDB,
				Name:     dbKey.Name,
			},
			Operation: runtime.RawExtension{
				Raw: byteData,
			},
			Recommender: kmapi.ObjectReference{
				Name: "kubedb-ops-manager",
			},
			Deadline: deadline,
			Rules: api.OperationPhaseRules{
				Success:    `self.status.phase == 'Successful'`,
				InProgress: `self.status.phase == 'Progressing'`,
				Failed:     `self.status.phase == 'Failed'`,
			},
		},
	}, nil
}

func (f *Framework) newPostgresRecommendation(dbKey client.ObjectKey, deadline *metav1.Time) (*api.Recommendation, error) {
	opsData := f.getPostgresRestartOpsRequest(dbKey)
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
			Description: "Postgres Database Restart",
			Target: core.TypedLocalObjectReference{
				APIGroup: pointer.StringP(kubedbapi.SchemeGroupVersion.String()),
				Kind:     kubedbapi.ResourceKindPostgres,
				Name:     dbKey.Name,
			},
			Operation: runtime.RawExtension{
				Raw: byteData,
			},
			Recommender: kmapi.ObjectReference{
				Name: "kubedb-ops-manager",
			},
			Deadline: deadline,
			Rules: api.OperationPhaseRules{
				Success:    `self.status.phase == 'Successful'`,
				InProgress: `self.status.phase == 'Progressing'`,
				Failed:     `self.status.phase == 'Failed'`,
			},
		},
	}, nil
}

func (f *Framework) createRecommendation(rcmd *api.Recommendation) (*api.Recommendation, error) {
	if err := f.kc.Create(f.ctx, rcmd); err != nil {
		return nil, err
	}

	err := wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
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

func (f *Framework) createNewPostgresRecommendation(dbKey client.ObjectKey, deadline *metav1.Time) (*api.Recommendation, error) {
	rcmd, err := f.newPostgresRecommendation(dbKey, deadline)
	if err != nil {
		return nil, err
	}

	return f.createRecommendation(rcmd)
}

func (f *Framework) createNewMongoDBRecommendation(dbKey client.ObjectKey, deadline *metav1.Time) (*api.Recommendation, error) {
	rcmd, err := f.newMongoDBRecommendation(dbKey, deadline)
	if err != nil {
		return nil, err
	}

	return f.createRecommendation(rcmd)
}

func (f *Framework) CreateNewMongoDBRecommendation(dbKey client.ObjectKey) (*api.Recommendation, error) {
	return f.createNewMongoDBRecommendation(dbKey, nil)
}

func (f *Framework) CreateNewPostgresRecommendation(dbKey client.ObjectKey) (*api.Recommendation, error) {
	return f.createNewPostgresRecommendation(dbKey, nil)
}

func (f *Framework) CreateNewRecommendationWithDeadline(dbKey client.ObjectKey, deadline *metav1.Time) (*api.Recommendation, error) {
	return f.createNewMongoDBRecommendation(dbKey, deadline)
}

func (f *Framework) getRecommendationNamespace() string {
	return f.namespace
}

func (f *Framework) WaitForRecommendationToBeSucceeded(key client.ObjectKey) error {
	return wait.PollUntilContextTimeout(context.Background(), time.Second*5, time.Minute*30, true, func(ctx context.Context) (bool, error) {
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
	_, err := kmc.PatchStatus(f.ctx, f.kc, rcmd, func(obj client.Object) client.Object {
		in := obj.(*api.Recommendation)
		in.Status.ApprovalStatus = api.ApprovalApproved

		return in
	})
	if err != nil {
		return err
	}

	return wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
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

	_, err := kmc.PatchStatus(f.ctx, f.kc, rcmd, func(obj client.Object) client.Object {
		in := obj.(*api.Recommendation)
		in.Status.ApprovedWindow = aw
		return in
	})
	return err
}

func (f *Framework) CheckRecommendationExecution(key client.ObjectKey, timeout time.Duration, interval time.Duration) error {
	return wait.PollUntilContextTimeout(context.Background(), interval, timeout, true, func(ctx context.Context) (bool, error) {
		rcmd := &api.Recommendation{}
		if err := f.kc.Get(f.ctx, key, rcmd); err != nil {
			return false, err
		}
		if rcmd.Status.Phase == api.InProgress || rcmd.Status.Phase == api.Succeeded {
			return true, nil
		}
		return false, nil
	})
}

func (f *Framework) UpdateRecommendationParallelism(key client.ObjectKey, par api.Parallelism) error {
	rcmd := &api.Recommendation{}
	if err := f.kc.Get(f.ctx, key, rcmd); err != nil {
		return err
	}

	_, err := kmc.PatchStatus(f.ctx, f.kc, rcmd, func(obj client.Object) client.Object {
		in := obj.(*api.Recommendation)
		in.Status.Parallelism = par
		return in
	})
	return err
}
