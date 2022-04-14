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

package parallelism

import (
	"context"

	api "kubeops.dev/supervisor/apis/supervisor/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ParallelRunner struct {
	ctx  context.Context
	kc   client.Client
	rcmd *api.Recommendation
}

func NewParallelRunner(ctx context.Context, kc client.Client, rcmd *api.Recommendation) *ParallelRunner {
	return &ParallelRunner{
		ctx:  ctx,
		kc:   kc,
		rcmd: rcmd,
	}
}

func (r *ParallelRunner) MaintainParallelism() (bool, error) {
	if r.rcmd.Status.Parallelism == api.QueuePerTarget {
		return r.isMaintainingQueuePerTarget()
	} else if r.rcmd.Status.Parallelism == api.QueuePerTargetAndNamespace {
		return r.isMaintainingQueuePerTargetAndNamespace()
	} else {
		return r.isMaintainingQueuePerNamespace()
	}
}

func (r *ParallelRunner) isMaintainingQueuePerNamespace() (bool, error) {
	rcmdList := &api.RecommendationList{}

	if err := r.kc.List(r.ctx, rcmdList, client.InNamespace(r.rcmd.Namespace)); err != nil {
		return false, err
	}

	for _, rc := range rcmdList.Items {
		if rc.Status.Phase == api.InProgress {
			return false, nil
		}
	}
	return true, nil
}

func (r *ParallelRunner) isMaintainingQueuePerTargetAndNamespace() (bool, error) {
	gv, err := schema.ParseGroupVersion(*r.rcmd.Spec.Target.APIGroup)
	if err != nil {
		return false, err
	}
	reqGK := schema.GroupKind{
		Group: gv.Group,
		Kind:  r.rcmd.Spec.Target.Kind,
	}

	rcmdList := &api.RecommendationList{}
	if err := r.kc.List(r.ctx, rcmdList, client.InNamespace(r.rcmd.Namespace)); err != nil {
		return false, err
	}
	return isMaintainingQueuePerGK(reqGK, rcmdList)
}

func (r *ParallelRunner) isMaintainingQueuePerTarget() (bool, error) {
	gv, err := schema.ParseGroupVersion(*r.rcmd.Spec.Target.APIGroup)
	if err != nil {
		return false, err
	}
	reqGK := schema.GroupKind{
		Group: gv.Group,
		Kind:  r.rcmd.Spec.Target.Kind,
	}

	rcmdList := &api.RecommendationList{}
	if err := r.kc.List(r.ctx, rcmdList); err != nil {
		return false, err
	}
	return isMaintainingQueuePerGK(reqGK, rcmdList)
}

func isMaintainingQueuePerGK(reqGK schema.GroupKind, rcList *api.RecommendationList) (bool, error) {
	for _, rc := range rcList.Items {
		gv, err := schema.ParseGroupVersion(*rc.Spec.Target.APIGroup)
		if err != nil {
			return false, err
		}

		gk := schema.GroupKind{
			Group: gv.Group,
			Kind:  rc.Spec.Target.Kind,
		}
		if reqGK != gk {
			continue
		}

		if rc.Status.Phase == api.InProgress {
			return false, nil
		}
	}
	return true, nil
}
